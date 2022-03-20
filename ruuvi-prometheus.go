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

// Command ruuvi-prometheus is a Prometheus exporter that listens to
// Ruuvi sensor Bluetooth Low Energy beacons and exposes them as metrics.
package main

import (
	"context"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/joneskoo/ruuvi-prometheus/bluetooth"
	"github.com/joneskoo/ruuvi-prometheus/metrics"
	"gitlab.com/jtaimisto/bluewalker/host"
	"gitlab.com/jtaimisto/bluewalker/ruuvi"
)

const (
	// defaultListen is the Prometheus metrics port.
	// Allocated in https://github.com/prometheus/prometheus/wiki/Default-port-allocations
	defaultListen = ":9521"

	// commandName is the name of this command used in help texts.
	commandName = "ruuvi-prometheus"
)

var version = ""

func main() {
	cmdline := parseSettings()

	log.Printf("%s %s listening on %v", commandName, version, cmdline.listen)

	if !cmdline.debug {
		// FIXME: bluewalker outputs to global logger so we need to discard all log globally
		log.SetOutput(ioutil.Discard)
	}

	server := http.Server{
		Addr:    cmdline.listen,
		Handler: metrics.Handler,
	}
	scanner := bluetooth.New(bluetooth.ScannerOpts{
		Device: cmdline.device,
		Logger: getDebugLogger(cmdline.debug),
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint
		cancel()
	}()

	// HTTP listener
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Printf("HTTP server ListenAndServe: %v", err)
		}
		cancel()
	}()

	// Bluetooth scanner
	go func() {
		scanner.HandleAdvertisement(handleRuuviAdvertisement)
		err := scanner.Scan()
		if err != nil {
			log.Printf("Bluetooth scanner Scan: %v", err)
		}
		cancel()
	}()

	<-ctx.Done()

	if err := server.Shutdown(context.Background()); err != nil {
		log.Printf("HTTP server Shutdown: %v", err)
	}

	scanner.Shutdown()
}

func getDebugLogger(debug bool) *log.Logger {
	var output io.Writer = os.Stderr
	if !debug {
		output = ioutil.Discard
	}
	return log.New(output, "DEBUG: ", log.LstdFlags)
}

func handleRuuviAdvertisement(sr *host.ScanReport) {
	for _, ads := range sr.Data {
		ruuviData, err := ruuvi.Decode(ads.Data)
		if err != nil {
			log.Printf("Unable to parse ruuvi data: %v; ads.Data=%x, len=%d, address=%x", err, ads.Data, len(ads.Data), sr.Address)
			continue
		}

		reading := metrics.RuuviReading{sr, ruuviData}
		metrics.ObserveRuuvi(reading)
	}
}
