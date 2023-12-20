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
	f.value = &value
	return nil
}
