// SPDX-License-Identifier: BSD-2-Clause
// Copyright (c) 2018 Jukka Taimisto
// Copyright (c) 2018- Joonas Kuorilehto

// Package bluetooth scans for Bluetooth LE advertisements. It is a generic,
// Ruuvi-agnostic scanner over the vendored HCI/host stack under internal,
// which is derived from bluewalker by Jukka Taimisto (see LICENSE).
// Callers supply their own AD filters (see the filter package).
package bluetooth

import (
	"fmt"
	"sync"

	"github.com/joneskoo/ruuvi-prometheus/internal/filter"
	"github.com/joneskoo/ruuvi-prometheus/internal/hci"
	"github.com/joneskoo/ruuvi-prometheus/internal/host"
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
	// Filters restrict which advertising reports are delivered to handlers.
	// A report passes if it matches all of the filters; an empty slice
	// delivers every report. See the filter package for constructors.
	Filters []filter.AdFilter
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
		filters: opts.Filters,

		quit: make(chan struct{}),
	}
	return s
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
