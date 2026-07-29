// SPDX-License-Identifier: BSD-2-Clause
// Copyright (c) 2018- Joonas Kuorilehto

package metrics

import (
	"strconv"
	"sync"
	"time"

	"github.com/joneskoo/ruuvi-prometheus/internal/host"
	"github.com/joneskoo/ruuvi-prometheus/internal/ruuvi"
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

	// formatInfo is an info-style metric: the value is always 1 and the frame
	// format version is carried in the "version" label. Encoding the version as
	// a label rather than the metric value keeps it aggregatable/joinable and
	// avoids the enum-as-value anti-pattern.
	formatInfo = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ruuvi_format_info",
		Help: "Ruuvi frame format version; value is always 1, version is in the version label",
	}, []string{"device", "version"})

	txPower = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ruuvi_txpower_dbm",
		Help: "Ruuvi transmit power in dBm",
	}, []string{"device"})

	moveCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ruuvi_movement_count",
		Help: "Ruuvi movement counter as reported by the sensor",
	}, []string{"device"})

	seqno = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ruuvi_seqno_current",
		Help: "Ruuvi frame sequence number",
	}, []string{"device"})

	pm25 = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ruuvi_pm2_5_ug_m3",
		Help: "Ruuvi sensor PM2.5 particulate matter concentration",
	}, []string{"device"})

	co2 = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ruuvi_co2_ppm",
		Help: "Ruuvi sensor CO2 concentration",
	}, []string{"device"})

	vocIndex = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ruuvi_voc_index",
		Help: "Ruuvi sensor VOC (volatile organic compounds) index",
	}, []string{"device"})

	noxIndex = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ruuvi_nox_index",
		Help: "Ruuvi sensor NOx (nitrous oxides) index",
	}, []string{"device"})

	luminosity = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ruuvi_luminosity_lux",
		Help: "Ruuvi sensor ambient light level",
	}, []string{"device"})

	soundAvg = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ruuvi_sound_avg_dba",
		Help: "Ruuvi sensor A-weighted average sound level",
	}, []string{"device"})

	calibrating = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ruuvi_calibrating",
		Help: "1 while the Ruuvi sensor calibration is in progress; air quality readings are not exported during calibration",
	}, []string{"device"})
)

// deviceVecs lists every metric vector with a device label, so that all
// series of an expired device can be removed without maintaining a
// per-metric list.
var deviceVecs = []interface {
	DeletePartialMatch(prometheus.Labels) int
}{
	ruuviFrames, humidity, temperature, pressure, acceleration, voltage,
	signalRSSI, formatInfo, txPower, moveCount, seqno,
	pm25, co2, vocIndex, noxIndex, luminosity, soundAvg, calibrating,
}

// ttl is the duration after which sensors are forgotten if signal is lost.
const ttl = 1 * time.Minute

var mu sync.Mutex
var deviceLastSeen map[string]time.Time

func init() {
	deviceLastSeen = make(map[string]time.Time)

	go func() {
		for range time.Tick(time.Minute) {
			clearExpired()
		}
	}()
}

func ObserveRuuvi(o RuuviReading) {
	addr := o.Address.String()

	mu.Lock()
	deviceLastSeen[addr] = time.Now()
	mu.Unlock()

	ruuviFrames.WithLabelValues(addr).Inc()
	signalRSSI.WithLabelValues(addr).Set(float64(o.Rssi))
	formatInfo.WithLabelValues(addr, strconv.Itoa(int(o.DataFormat))).Set(1)

	if o.VoltageValid() {
		voltage.WithLabelValues(addr).Set(float64(o.Voltage) / 1000)
	}
	if o.PressureValid() {
		pressure.WithLabelValues(addr).Set(float64(o.Pressure) / 100)
	}
	if o.TemperatureValid() {
		temperature.WithLabelValues(addr).Set(float64(o.Temperature))
	}
	if o.HumidityValid() {
		humidity.WithLabelValues(addr).Set(float64(o.Humidity) / 100)
	}
	if o.AccelerationValid() {
		acceleration.WithLabelValues(addr, "X").Set(float64(o.AccelerationX))
		acceleration.WithLabelValues(addr, "Y").Set(float64(o.AccelerationY))
		acceleration.WithLabelValues(addr, "Z").Set(float64(o.AccelerationZ))
	}
	if o.TxPowerValid() {
		txPower.WithLabelValues(addr).Set(float64(o.TxPower))
	}
	if o.MoveCountValid() {
		moveCount.WithLabelValues(addr).Set(float64(o.MoveCount))
	}
	if o.SeqnoValid() {
		seqno.WithLabelValues(addr).Set(float64(o.Seqno))
	}
	if o.LuminosityValid() {
		luminosity.WithLabelValues(addr).Set(float64(o.Luminosity))
	}
	if o.SoundAvgValid() {
		soundAvg.WithLabelValues(addr).Set(float64(o.SoundAvg))
	}

	if o.DataFormat == ruuvi.FormatV6 {
		if o.Calibrating {
			calibrating.WithLabelValues(addr).Set(1)
		} else {
			calibrating.WithLabelValues(addr).Set(0)
		}
	}
	// Air quality readings are unreliable while the sensor calibration
	// is in progress and are not exported until calibration completes.
	if o.Calibrating {
		return
	}
	if o.PM25Valid() {
		pm25.WithLabelValues(addr).Set(float64(o.PM25))
	}
	if o.CO2Valid() {
		co2.WithLabelValues(addr).Set(float64(o.CO2))
	}
	if o.VOCIndexValid() {
		vocIndex.WithLabelValues(addr).Set(float64(o.VOCIndex))
	}
	if o.NOXIndexValid() {
		noxIndex.WithLabelValues(addr).Set(float64(o.NOXIndex))
	}
}

func clearExpired() {
	mu.Lock()
	defer mu.Unlock()

	now := time.Now()
	for addr, ls := range deviceLastSeen {
		if now.Sub(ls) > ttl {
			for _, vec := range deviceVecs {
				vec.DeletePartialMatch(prometheus.Labels{"device": addr})
			}
			delete(deviceLastSeen, addr)
		}
	}
}

type RuuviReading struct {
	*host.ScanReport
	*ruuvi.Data
}
