package main

import (
	"encoding/hex"
	"flag"
	"io"
	"log"
	"os"
	"testing"

	"gitlab.com/jtaimisto/bluewalker/hci"
	"gitlab.com/jtaimisto/bluewalker/host"
)

func TestDeviceFlag(t *testing.T) {
	var dev string
	f := &deviceFlag{&dev}

	if err := f.Set(""); err == nil {
		t.Error("Set(\"\") = nil, want error for empty device name")
	}
	if err := f.Set("hci5"); err != nil {
		t.Fatalf("Set(\"hci5\") = %v, want nil", err)
	}
	if got := f.Get(); got != "hci5" {
		t.Errorf("Get() = %q, want hci5", got)
	}
	if got := f.String(); got != "hci5" {
		t.Errorf("String() = %q, want hci5", got)
	}

	// A deviceFlag with a nil value must not panic and reports empty.
	if got := (deviceFlag{}).Get(); got != "" {
		t.Errorf("Get() on nil value = %q, want empty", got)
	}
}

func TestParseSettings(t *testing.T) {
	oldArgs, oldCL := os.Args, flag.CommandLine
	defer func() { os.Args, flag.CommandLine = oldArgs, oldCL }()

	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	os.Args = []string{"ruuvi-prometheus", "-listen", ":1234", "-device", "hci9", "-debug"}

	s := parseSettings()

	if s.listen != ":1234" {
		t.Errorf("listen = %q, want :1234", s.listen)
	}
	if s.device != "hci9" {
		t.Errorf("device = %q, want hci9", s.device)
	}
	if !s.debug {
		t.Error("debug = false, want true")
	}
}

func TestParseSettingsDefaults(t *testing.T) {
	oldArgs, oldCL := os.Args, flag.CommandLine
	defer func() { os.Args, flag.CommandLine = oldArgs, oldCL }()

	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	os.Args = []string{"ruuvi-prometheus"}

	s := parseSettings()

	if s.listen != defaultListen {
		t.Errorf("listen = %q, want default %q", s.listen, defaultListen)
	}
	if s.device != "hci0" {
		t.Errorf("device = %q, want default hci0", s.device)
	}
	if s.debug {
		t.Error("debug = true, want default false")
	}
}

func TestGetDebugLogger(t *testing.T) {
	if getDebugLogger(true) == nil {
		t.Error("getDebugLogger(true) = nil")
	}
	if getDebugLogger(false) == nil {
		t.Error("getDebugLogger(false) = nil")
	}
}

func TestHandleRuuviAdvertisement(t *testing.T) {
	log.SetOutput(io.Discard)
	defer log.SetOutput(os.Stderr)

	addr, err := hci.BtAddressFromString("ee:36:80:be:ec:fe")
	if err != nil {
		t.Fatalf("invalid address: %v", err)
	}
	frame, err := hex.DecodeString("99040501c4271ac87dfc18fc18fc18af166403e8010203040506")
	if err != nil {
		t.Fatalf("invalid frame hex: %v", err)
	}

	// Valid Ruuvi manufacturer data plus a non-Ruuvi AD structure that fails to
	// decode: the handler must process the good one and skip the bad one without
	// panicking.
	sr := &host.ScanReport{
		Address: addr,
		Data: []*hci.AdStructure{
			{Data: frame},
			{Data: []byte{0x00, 0x01}},
		},
	}
	handleRuuviAdvertisement(sr)
}
