// Copyright Joonas Kuorilehto 2018.

// Command ruuvi-prometheus is a Prometheus exporter that listens to
// Ruuvi sensor Bluetooth Low Energy beacons and exposes them as metrics.
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/joneskoo/ruuvi-prometheus/bluetooth"
	"github.com/joneskoo/ruuvi-prometheus/metrics"
)

const (
	defaultListen = "127.0.0.1:9521"
)

func main() {
	cmdline := parseSettings()

	if !cmdline.debug {
		log.SetOutput(ioutil.Discard)
	}

	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, syscall.SIGTERM, syscall.SIGINT)

	errCh := make(chan error, 1)

	go func() {
		errCh <- metrics.Start(cmdline.listen)
	}()

	go func() {
		observations, err := bluetooth.Listen(cmdline.device, cmdline.debug)
		if err != nil {
			errCh <- err
		}
		for o := range observations {
			metrics.ObserveRuuvi(o)
		}
	}()

	select {
	case sig := <-terminate:
		fmt.Fprintf(os.Stderr, "Exiting on signal %v\n", sig)
		os.Exit(0)
	case err := <-errCh:
		fmt.Fprintf(os.Stderr, "Exiting on error: %v\n", err)
		os.Exit(1)
	}
}
