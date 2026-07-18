// Copyright (c) 2018, Joonas Kuorilehto
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met:
//
// 1. Redistributions of source code must retain the above copyright notice, this
//    list of conditions and the following disclaimer.
//
// 2. Redistributions in binary form must reproduce the above copyright notice,
//    this list of conditions and the following disclaimer in the documentation
//    and/or other materials provided with the distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
// FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
// DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
// SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
// CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
// OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"gitlab.com/jtaimisto/bluewalker/host"
	"gitlab.com/jtaimisto/bluewalker/ruuvi"
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
		Help: "Ruuvi frame format version (e.g. 3, 5 or 6)",
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
	signalRSSI, format, txPower, moveCount, seqno,
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
	format.WithLabelValues(addr).Set(float64(o.DataFormat))

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
