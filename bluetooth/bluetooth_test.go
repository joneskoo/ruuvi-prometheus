package bluetooth

import (
	"io"
	"log"
	"testing"

	"gitlab.com/jtaimisto/bluewalker/host"
)

func testLogger() Logger { return log.New(io.Discard, "", 0) }

func TestNew(t *testing.T) {
	s := New(ScannerOpts{Device: "hci7", Logger: testLogger()})

	if s.device != "hci7" {
		t.Errorf("device = %q, want hci7", s.device)
	}
	if len(s.filters) != 1 {
		t.Errorf("filters = %d, want 1 (vendor-is-Ruuvi)", len(s.filters))
	}
	if s.quit == nil {
		t.Error("quit channel is nil")
	}
	if s.active {
		t.Error("active = true, want passive scanning by default")
	}
}

func TestFilterVendorIsRuuvi(t *testing.T) {
	if got := len(filterVendorIsRuuvi()); got != 1 {
		t.Errorf("filterVendorIsRuuvi() returned %d filters, want 1", got)
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
