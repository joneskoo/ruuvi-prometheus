// Copyright (c) 2018, Joonas Kuorilehto
// All rights reserved.
//
// Derived from bluewalker by Jukka Taimisto.
//
// Copyright (c) 2018, Jukka Taimisto
// All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met:
//
// * Redistributions of source code must retain the above copyright notice, this
//   list of conditions and the following disclaimer.
//
// * Redistributions in binary form must reproduce the above copyright notice,
//   this list of conditions and the following disclaimer in the documentation
//   and/or other materials provided with the distribution.
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

package bluetooth

import (
	"fmt"

	"gitlab.com/jtaimisto/bluewalker/filter"
	"gitlab.com/jtaimisto/bluewalker/hci"
	"gitlab.com/jtaimisto/bluewalker/host"
	"gitlab.com/jtaimisto/bluewalker/ruuvi"
)

type RuuviReading struct {
	// Address is the sensor Bluetooth address.
	Address string
	// RSSI is the received signal strength in dBm.
	RSSI float64
	// Humidity is the measured relative humidity 0..1.
	Humidity float64
	// Temperature is the measured temperature in °C.
	Temperature float64
	// Pressure is the air pressure in hPa.
	Pressure float64
	// AccelerationX is the acceleration sensor X axis reading in g.
	AccelerationX float64
	// AccelerationY is the acceleration sensor Y axis reading in g.
	AccelerationY float64
	// AccelerationZ is the acceleration sensor Z axis reading in g.
	AccelerationZ float64
	// Voltage is the sensor battery voltage in Volts.
	Voltage float64
}

// Scanner scans for Bluetooth LE advertisements.
type Scanner struct {
	device  string
	active  bool
	filters []filter.AdFilter
	log     Logger

	quit chan struct{}
}

type ScannerOpts struct {
	Device string
	Logger Logger
}

// Logger is a log.Logger compatible logger.
type Logger interface {
	Print(v ...interface{})
	Printf(format string, v ...interface{})
}

func New(opts ScannerOpts) *Scanner {
	s := &Scanner{
		device:  opts.Device,
		log:     opts.Logger,
		active:  false,
		filters: filterVendorIsRuuvi(),

		quit: make(chan struct{}),
	}
	return s
}

func filterVendorIsRuuvi() []filter.AdFilter {
	// Ruuvi Innovations Ltd. vendor id: 1177 (little endian)
	// https://www.bluetooth.com/specifications/assigned-numbers/company-identifiers
	ruuviVendor := []byte{0x99, 0x04}
	flt := filter.ByVendor(ruuviVendor)
	return []filter.AdFilter{flt}
}

func (s *Scanner) Scan() (<-chan RuuviReading, error) {
	scanResults, err := s.startScanning()
	if err != nil {
		return nil, err
	}

	observations := make(chan RuuviReading)
	receiveloop := func() {
		s.log.Print("Starting receive loop")
		defer close(observations)

		for sr := range scanResults {
			s.log.Print("Received frame")
			for _, ads := range sr.Data {
				ruuviData, err := ruuvi.Unmarshall(ads.Data)
				if err != nil {
					s.log.Printf("Unable to parse ruuvi data: %v", err)
					continue
				}

				observations <- RuuviReading{
					Address:       sr.Address.String(),
					RSSI:          float64(sr.Rssi),
					Humidity:      float64(ruuviData.Humidity) / 100,
					Temperature:   float64(ruuviData.Temperature),
					Pressure:      float64(ruuviData.Pressure) / 100,
					AccelerationX: float64(ruuviData.AccelerationX),
					AccelerationY: float64(ruuviData.AccelerationY),
					AccelerationZ: float64(ruuviData.AccelerationZ),
					Voltage:       float64(ruuviData.Voltage) / 1000,
				}
			}
		}
		s.log.Print("End of receive loop")
		close(s.quit)
	}

	go receiveloop()

	return observations, nil
}

func (s *Scanner) Shutdown() {
	s.log.Print("Requesting to stop scan")

	select {
	case s.quit <- struct{}{}:
	default:
	}

	<-s.quit
}

func (s *Scanner) startScanning() (chan *host.ScanReport, error) {
	// FIXME: second use of the same scanner breaks shutdown logic
	s.log.Printf("Using device %v", s.device)

	raw, err := hci.Raw(s.device)
	if err != nil {
		return nil, fmt.Errorf(`Error while opening RAW HCI socket: %v
	Are you running as root and have you run sudo hciconfig %s down?`, err, s.device)
	}

	host := host.New(raw)
	if err = host.Init(); err != nil {
		return nil, fmt.Errorf("Unable to initialize host: %v", err)
	}

	reportChan, err := host.StartScanning(s.active, s.filters)
	if err != nil {
		return nil, fmt.Errorf("Unable to start scanning: %v", err)
	}

	go func() {
		<-s.quit
		s.log.Print("Stopping scan")
		err := host.StopScanning()
		if err != nil {
			s.log.Printf("failed to stop scanning: %v", err)
		}
		host.Deinit()
		for range s.quit {
		}
	}()
	return reportChan, nil
}
