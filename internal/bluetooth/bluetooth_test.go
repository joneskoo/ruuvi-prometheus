// SPDX-License-Identifier: BSD-2-Clause
// Copyright (c) 2018- Joonas Kuorilehto

package bluetooth

import (
	"io"
	"log"
	"testing"

	"github.com/joneskoo/ruuvi-prometheus/internal/filter"
	"github.com/joneskoo/ruuvi-prometheus/internal/host"
)

func testLogger() Logger { return log.New(io.Discard, "", 0) }

func TestNew(t *testing.T) {
	flt := filter.ByVendor([]byte{0xaa, 0xbb})
	s := New(ScannerOpts{Device: "hci7", Logger: testLogger(), Filters: []filter.AdFilter{flt}})

	if s.device != "hci7" {
		t.Errorf("device = %q, want hci7", s.device)
	}
	if len(s.filters) != 1 {
		t.Errorf("filters = %d, want 1 (the filter passed in ScannerOpts)", len(s.filters))
	}
	if s.quit == nil {
		t.Error("quit channel is nil")
	}
	if s.active {
		t.Error("active = true, want passive scanning by default")
	}
}

func TestHandleAdvertisement(t *testing.T) {
	s := New(ScannerOpts{Device: "hci0", Logger: testLogger()})

	s.HandleAdvertisement(func(*host.ScanReport) {})
	s.HandleAdvertisement(func(*host.ScanReport) {})

	if len(s.handlers) != 2 {
		t.Errorf("handlers = %d, want 2", len(s.handlers))
	}
}

// TestShutdownIdempotent verifies Shutdown closes the quit channel exactly once,
// so repeated calls (e.g. from Scan's cleanup and main's defer) do not panic.
func TestShutdownIdempotent(t *testing.T) {
	s := New(ScannerOpts{Device: "hci0", Logger: testLogger()})

	s.Shutdown()
	s.Shutdown() // must not panic on a second close

	select {
	case <-s.quit:
		// closed as expected
	default:
		t.Error("quit channel not closed after Shutdown")
	}
}
