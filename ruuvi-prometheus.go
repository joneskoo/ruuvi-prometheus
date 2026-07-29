// SPDX-License-Identifier: BSD-2-Clause
// Copyright (c) 2018- Joonas Kuorilehto

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

	"github.com/joneskoo/ruuvi-prometheus/internal/bluetooth"
	"github.com/joneskoo/ruuvi-prometheus/internal/filter"
	"github.com/joneskoo/ruuvi-prometheus/internal/host"
	"github.com/joneskoo/ruuvi-prometheus/internal/ruuvi"
	"github.com/joneskoo/ruuvi-prometheus/metrics"
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
		// The vendored BLE stack (internal/) logs through the standard
		// global logger, so silence it globally unless -debug is set.
		log.SetOutput(ioutil.Discard)
	}

	server := http.Server{
		Addr:    cmdline.listen,
		Handler: metrics.Handler,
	}
	scanner := bluetooth.New(bluetooth.ScannerOpts{
		Device:  cmdline.device,
		Logger:  getDebugLogger(cmdline.debug),
		Filters: []filter.AdFilter{ruuviAdFilter()},
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
	os.Exit(1)
}

func getDebugLogger(debug bool) *log.Logger {
	var output io.Writer = os.Stderr
	if !debug {
		output = ioutil.Discard
	}
	return log.New(output, "DEBUG: ", log.LstdFlags)
}

// ruuviAdFilter matches advertisements that carry Ruuvi manufacturer specific
// data. This is where the generic BLE scanner and the Ruuvi domain are
// combined: the scanner only knows about filters, and the Ruuvi identity comes
// from the ruuvi package.
func ruuviAdFilter() filter.AdFilter {
	// Manufacturer specific data is prefixed with the company id in
	// little-endian byte order.
	preamble := []byte{byte(ruuvi.ManufacturerID & 0xff), byte(ruuvi.ManufacturerID >> 8)}
	return filter.ByVendor(preamble)
}

func handleRuuviAdvertisement(sr *host.ScanReport) {
	for _, ads := range sr.Data {
		ruuviData, err := ruuvi.Decode(ads.Data)
		if err != nil {
			log.Printf("Unable to parse ruuvi data: %v; ads.Data=%x, len=%d, address=%x", err, ads.Data, len(ads.Data), sr.Address)
			continue
		}

		reading := metrics.RuuviReading{ScanReport: sr, Data: ruuviData}
		metrics.ObserveRuuvi(reading)
	}
}
