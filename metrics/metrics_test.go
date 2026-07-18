package metrics

import (
	"encoding/hex"
	"testing"

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
	if got := testutil.ToFloat64(format.WithLabelValues(testAddr)); got != 6 {
		t.Fatalf("format = %v, expected 6", got)
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
