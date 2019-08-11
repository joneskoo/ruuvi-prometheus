// Copyright Joonas Kuorilehto 2018.

package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	ruuviFrames = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ruuvi_frames_total",
		Help: "Total Ruuvi frames received",
	}, []string{"device"})

	humidity = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ruuvi_humidity_ratio",
		Help: "Ruuvi tag sensor relative humidity",
	}, []string{"device"})

	temperature = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ruuvi_temperature_celsius",
		Help: "Ruuvi tag sensor temperature",
	}, []string{"device"})

	pressure = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ruuvi_pressure_hpa",
		Help: "Ruuvi tag sensor air pressure",
	}, []string{"device"})

	acceleration = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ruuvi_acceleration_g",
		Help: "Ruuvi tag sensor acceleration X/Y/Z",
	}, []string{"device", "axis"})

	voltage = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ruuvi_battery_volts",
		Help: "Ruuvi tag battery voltage",
	}, []string{"device"})

	signalRSSI = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ruuvi_rssi_dbm",
		Help: "Ruuvi tag received signal strength RSSI",
	}, []string{"device"})

	format = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ruuvi_format",
		Help: "Ruuvi frame format version (e.g. 3 or 5)",
	}, []string{"device"})

	txPower = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ruuvi_txpower_dbm",
		Help: "Ruuvi transmit power in dBm",
	}, []string{"device"})

	moveCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ruuvi_movecount_total",
		Help: "Ruuvi movement counter",
	}, []string{"device"})

	seqno = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ruuvi_seqno_current",
		Help: "Ruuvi frame sequence number",
	}, []string{"device"})
)

// ttl is the duration after which sensors are forgotten if signal is lost.
const ttl = 1 * time.Minute

var deviceLastSeen map[string]time.Time
var mu sync.Mutex

func init() {
	deviceLastSeen = make(map[string]time.Time)

	go func() {
		for range time.Tick(time.Minute) {
			clearExpired()
		}
	}()
}

func ObserveRuuvi(o RuuviReading) {
	addr := o.Address()

	mu.Lock()
	deviceLastSeen[addr] = time.Now()
	mu.Unlock()

	ruuviFrames.WithLabelValues(addr).Inc()
	signalRSSI.WithLabelValues(addr).Set(o.RSSI())
	voltage.WithLabelValues(addr).Set(o.Voltage())
	pressure.WithLabelValues(addr).Set(o.Pressure())
	temperature.WithLabelValues(addr).Set(o.Temperature())
	humidity.WithLabelValues(addr).Set(o.Humidity())
	acceleration.WithLabelValues(addr, "X").Set(o.AccelerationX())
	acceleration.WithLabelValues(addr, "Y").Set(o.AccelerationY())
	acceleration.WithLabelValues(addr, "Z").Set(o.AccelerationZ())
	format.WithLabelValues(addr).Set(float64(o.DataFormat()))
	txPower.WithLabelValues(addr).Set(float64(o.TxPower()))
	moveCount.WithLabelValues(addr).Set(float64(o.MoveCount()))
	seqno.WithLabelValues(addr).Set(float64(o.Seqno()))
}

func clearExpired() {
	mu.Lock()
	defer mu.Unlock()

	// log.Println("Checking for expired devices")
	now := time.Now()
	for addr, ls := range deviceLastSeen {
		if now.Sub(ls) > ttl {
			// log.Printf("%v expired", addr)
			ruuviFrames.DeleteLabelValues(addr)
			signalRSSI.DeleteLabelValues(addr)
			voltage.DeleteLabelValues(addr)
			pressure.DeleteLabelValues(addr)
			temperature.DeleteLabelValues(addr)
			humidity.DeleteLabelValues(addr)
			acceleration.DeleteLabelValues(addr, "X")
			acceleration.DeleteLabelValues(addr, "Y")
			acceleration.DeleteLabelValues(addr, "Z")
			format.DeleteLabelValues(addr)
			txPower.DeleteLabelValues(addr)
			moveCount.DeleteLabelValues(addr)
			seqno.DeleteLabelValues(addr)

			delete(deviceLastSeen, addr)
		}
	}
}

type RuuviReading interface {
	// Address is the sensor Bluetooth address.
	Address() string
	// RSSI is the received signal strength in dBm.
	RSSI() float64
	// Humidity is the measured relative humidity 0..1.
	Humidity() float64
	// Temperature is the measured temperature in °C.
	Temperature() float64
	// Pressure is the air pressure in hPa.
	Pressure() float64
	// AccelerationX is the acceleration sensor X axis reading in g.
	AccelerationX() float64
	// AccelerationY is the acceleration sensor Y axis reading in g.
	AccelerationY() float64
	// AccelerationZ is the acceleration sensor Z axis reading in g.
	AccelerationZ() float64
	// Voltage is the sensor battery voltage in Volts.
	Voltage() float64
	// DataFormat is the version of the Ruuvi protocol
	DataFormat() int
	TxPower() int
	MoveCount() int
	Seqno() int
}
