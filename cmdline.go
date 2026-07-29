// SPDX-License-Identifier: BSD-2-Clause
// Copyright (c) 2018- Joonas Kuorilehto

package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
)

type settings struct {
	device string
	debug  bool
	listen string
}

func parseSettings() (cmdline settings) {
	cmdline.device = "hci0"
	device := &deviceFlag{&cmdline.device}
	versionFlag := flag.Bool("version", false, "Show version number and quit")
	flag.Var(device, "device", "HCI device to use")
	flag.BoolVar(&cmdline.debug, "debug", false, "Debug output")
	flag.StringVar(&cmdline.listen, "listen", defaultListen, "Listen address for Prometheus metrics")
	flag.Parse()
	if *versionFlag {
		printVersion()
	}
	return cmdline
}

func printVersion() {
	fmt.Printf("%s %s (%s/%s %s)\n", commandName, version, runtime.GOOS, runtime.GOARCH, runtime.Version())
	os.Exit(0)
}

type deviceFlag struct{ value *string }

func (f deviceFlag) String() string {
	return f.Get()
}

func (f deviceFlag) Get() string {
	if f.value == nil {
		return ""
	}
	return *f.value
}

func (f *deviceFlag) Set(value string) error {
	if value == "" {
		return fmt.Errorf("missing device name")
	}
	if f.value == nil {
		f.value = new(string)
	}
	// Write through the pointer so the wrapped settings.device is updated;
	// reassigning f.value would orphan it and silently drop the -device flag.
	*f.value = value
	return nil
}
