// Copyright Joonas Kuorilehto 2018.

package metrics

import (
	"github.com/joneskoo/ruuvi-prometheus/bluetooth"
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
)

func ObserveRuuvi(o bluetooth.RuuviReading) {
	ruuviFrames.WithLabelValues(o.Address).Inc()
	signalRSSI.WithLabelValues(o.Address).Set(o.RSSI)
	voltage.WithLabelValues(o.Address).Set(o.Voltage)
	pressure.WithLabelValues(o.Address).Set(o.Pressure)
	temperature.WithLabelValues(o.Address).Set(o.Temperature)
	humidity.WithLabelValues(o.Address).Set(o.Humidity)
	acceleration.WithLabelValues(o.Address, "X").Set(o.AccelerationX)
	acceleration.WithLabelValues(o.Address, "Y").Set(o.AccelerationY)
	acceleration.WithLabelValues(o.Address, "Z").Set(o.AccelerationZ)
}
