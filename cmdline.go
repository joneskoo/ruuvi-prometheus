// Copyright Joonas Kuorilehto 2018.

package main

import (
	"flag"
	"fmt"
)

type settings struct {
	device string
	debug  bool
	listen string
}

func parseSettings() (cmdline settings) {
	cmdline.device = "hci0"
	device := deviceFlag{&cmdline.device}
	flag.Var(device, "device", "HCI device to use")
	flag.BoolVar(&cmdline.debug, "debug", false, "Debug output")
	flag.StringVar(&cmdline.listen, "listen", defaultListen, "Listen address for Prometheus metrics")
	flag.Parse()
	return cmdline
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

func (f deviceFlag) Set(value string) error {
	if value == "" {
		return fmt.Errorf("missing device name")
	}
	f.value = &value
	return nil
}
