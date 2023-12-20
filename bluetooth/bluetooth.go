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
	"sync"

	"gitlab.com/jtaimisto/bluewalker/filter"
	"gitlab.com/jtaimisto/bluewalker/hci"
	"gitlab.com/jtaimisto/bluewalker/host"
)

// Scanner scans for Bluetooth LE advertisements.
type Scanner struct {
	device   string
	active   bool
	filters  []filter.AdFilter
	log      Logger
	handlers []AdvertisementHandler

	quitOnce sync.Once
	quit     chan struct{}
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

type AdvertisementHandler func(*host.ScanReport)

func (s *Scanner) Scan() error {
	s.log.Printf("Using device %v", s.device)

	raw, err := hci.Raw(s.device)
	if err != nil {
		return fmt.Errorf(`error while opening RAW HCI socket: %v
	Are you running as root and have you run sudo hciconfig %s down?`, err, s.device)
	}

	host := host.New(raw)
	if err = host.Init(); err != nil {
		return fmt.Errorf("unable to initialize host: %v", err)
	}

	reportChan, err := host.StartScanning(s.active, s.filters)
	if err != nil {
		return fmt.Errorf("unable to start scanning: %v", err)
	}

receiveLoop:
	for {
		select {
		case sr := <-reportChan:
			for _, handle := range s.handlers {
				go handle(sr)
			}
		case <-s.quit:
			break receiveLoop
		}
	}

	s.log.Print("Requesting to stop scan")
	err = host.StopScanning()
	if err != nil {
		s.log.Printf("failed to stop scanning: %v", err)
	}
	host.Deinit()

	s.Shutdown()
	return err
}

func (s *Scanner) HandleAdvertisement(h AdvertisementHandler) {
	s.handlers = append(s.handlers, h)
}

func (s *Scanner) Shutdown() {
	s.quitOnce.Do(func() {
		close(s.quit)
	})

}
