package metrics

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"gitlab.com/jtaimisto/bluewalker/hci"
	"gitlab.com/jtaimisto/bluewalker/host"
	"gitlab.com/jtaimisto/bluewalker/ruuvi"
)

func reading(t *testing.T, addr string, frame string) RuuviReading {
	t.Helper()
	data, err := hex.DecodeString(frame)
	if err != nil {
		t.Fatalf("invalid frame hex: %v", err)
	}
	decoded, err := ruuvi.Decode(data)
	if err != nil {
		t.Fatalf("unable to decode frame: %v", err)
	}
	btaddr, err := hci.BtAddressFromString(addr)
	if err != nil {
		t.Fatalf("invalid address: %v", err)
	}
	return RuuviReading{
		ScanReport: &host.ScanReport{Address: btaddr, Rssi: -60},
		Data:       decoded,
	}
}

const testAddr = "ee:36:80:be:ec:fd"

// v6Frame is a real world Ruuvi Air data format 6 advertisement with
// CO2 777 ppm and sequence number 141.
const v6Frame = "990406" + "1297" + "4b7c" + "c625" + "0009" + "0309" +
	"07" + "00" + "ff" + "ff" + "8d" + "94" + "beecfd"

// v6CalibratingFrame is v6Frame with the calibration in progress flag
// set, different measurement values and sequence number 142.
const v6CalibratingFrame = "990406" + "1297" + "4b7c" + "c625" + "000f" + "03e8" +
	"07" + "00" + "ff" + "ff" + "8e" + "95" + "beecfd"

// TestCalibrationGatesAirQuality checks that air quality readings are not
// exported while sensor calibration is in progress, and that the
// calibration status itself is exported.
func TestCalibrationGatesAirQuality(t *testing.T) {
	clearDevice(t)

	ObserveRuuvi(reading(t, testAddr, v6Frame))
	if got := testutil.ToFloat64(formatInfo.WithLabelValues(testAddr, "6")); got != 1 {
		t.Fatalf("format_info{version=6} = %v, expected 1", got)
	}
	if got := testutil.ToFloat64(calibrating.WithLabelValues(testAddr)); got != 0 {
		t.Fatalf("calibrating = %v, expected 0", got)
	}
	if got := testutil.ToFloat64(co2.WithLabelValues(testAddr)); got != 777 {
		t.Fatalf("co2 = %v, expected 777", got)
	}
	if got := testutil.ToFloat64(pm25.WithLabelValues(testAddr)); got != float64(float32(9)*0.1) {
		t.Fatalf("pm25 = %v, expected 0.9", got)
	}
	// The frame reports sound level as not available, so no series may
	// be created for it.
	if got := testutil.CollectAndCount(soundAvg); got != 0 {
		t.Errorf("soundAvg has %d series, expected 0 for unavailable reading", got)
	}

	ObserveRuuvi(reading(t, testAddr, v6CalibratingFrame))
	if got := testutil.ToFloat64(calibrating.WithLabelValues(testAddr)); got != 1 {
		t.Errorf("calibrating = %v, expected 1", got)
	}
	// Air quality readings from the calibrating frame must not be
	// exported; the previous values remain.
	if got := testutil.ToFloat64(co2.WithLabelValues(testAddr)); got != 777 {
		t.Errorf("co2 = %v after calibrating frame, expected unchanged 777", got)
	}
	if got := testutil.ToFloat64(pm25.WithLabelValues(testAddr)); got != float64(float32(9)*0.1) {
		t.Errorf("pm25 = %v after calibrating frame, expected unchanged 0.9", got)
	}
	// Environmental readings are exported also during calibration.
	if got := testutil.ToFloat64(seqno.WithLabelValues(testAddr)); got != 142 {
		t.Errorf("seqno = %v after calibrating frame, expected 142", got)
	}
}

// v5Frame is the "vector 1 - v5" data-format-5 advertisement from the
// bluewalker ruuvi test vectors: temperature 2.26, humidity 25.025 %,
// pressure 101325 Pa, acceleration -1/-1/-1 g, voltage 3000 mV, tx power 4 dBm,
// move count 100 and sequence number 1000.
const v5Frame = "99040501c4271ac87dfc18fc18fc18af166403e8010203040506"

// TestObserveFormat5 exercises the environmental/motion branches of ObserveRuuvi
// that a data-format-6 (air) frame does not carry, and checks unit conversions.
func TestObserveFormat5(t *testing.T) {
	clearDevice(t)

	ObserveRuuvi(reading(t, testAddr, v5Frame))

	checks := []struct {
		name   string
		metric float64
		want   float64
	}{
		{"temperature_celsius", testutil.ToFloat64(temperature.WithLabelValues(testAddr)), float64(float32(2.26))},
		{"humidity_ratio", testutil.ToFloat64(humidity.WithLabelValues(testAddr)), float64(float32(25.025)) / 100},
		{"pressure_hpa", testutil.ToFloat64(pressure.WithLabelValues(testAddr)), 1013.25},
		{"battery_volts", testutil.ToFloat64(voltage.WithLabelValues(testAddr)), 3.0},
		{"acceleration_x", testutil.ToFloat64(acceleration.WithLabelValues(testAddr, "X")), -1},
		{"acceleration_y", testutil.ToFloat64(acceleration.WithLabelValues(testAddr, "Y")), -1},
		{"acceleration_z", testutil.ToFloat64(acceleration.WithLabelValues(testAddr, "Z")), -1},
		{"txpower_dbm", testutil.ToFloat64(txPower.WithLabelValues(testAddr)), 4},
		{"movement_count", testutil.ToFloat64(moveCount.WithLabelValues(testAddr)), 100},
		{"seqno", testutil.ToFloat64(seqno.WithLabelValues(testAddr)), 1000},
		{"format_info_v5", testutil.ToFloat64(formatInfo.WithLabelValues(testAddr, "5")), 1},
	}
	for _, c := range checks {
		if c.metric != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.metric, c.want)
		}
	}

	// Air-quality metrics belong to format 6 only; no series may exist.
	if got := testutil.CollectAndCount(co2); got != 0 {
		t.Errorf("co2 has %d series, expected 0 for a format-5 frame", got)
	}
}

// TestClearExpired verifies that a device whose last-seen time is older than the
// TTL has all of its series removed.
func TestClearExpired(t *testing.T) {
	clearDevice(t)

	ObserveRuuvi(reading(t, testAddr, v5Frame))
	if got := testutil.CollectAndCount(temperature); got == 0 {
		t.Fatal("expected a temperature series after observing a reading")
	}

	// Backdate the last-seen time beyond the TTL, then run the sweep.
	mu.Lock()
	deviceLastSeen[testAddr] = time.Now().Add(-2 * ttl)
	mu.Unlock()

	clearExpired()

	if got := testutil.CollectAndCount(temperature); got != 0 {
		t.Errorf("temperature has %d series after expiry, want 0", got)
	}
	mu.Lock()
	_, stillTracked := deviceLastSeen[testAddr]
	mu.Unlock()
	if stillTracked {
		t.Error("expired device still tracked in deviceLastSeen")
	}
}

// clearDevice removes all state for the test device.
func clearDevice(t *testing.T) {
	t.Helper()
	mu.Lock()
	defer mu.Unlock()
	for _, vec := range deviceVecs {
		vec.DeletePartialMatch(map[string]string{"device": testAddr})
	}
	delete(deviceLastSeen, testAddr)
}
