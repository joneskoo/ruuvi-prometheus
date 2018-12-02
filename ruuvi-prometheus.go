// Copyright Joonas Kuorilehto 2018.

// Command ruuvi-prometheus is a Prometheus exporter that listens to
// Ruuvi sensor Bluetooth Low Energy beacons and exposes them as metrics.
package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/joneskoo/ruuvi-prometheus/bluetooth"
	"github.com/joneskoo/ruuvi-prometheus/metrics"
)

const (
	// defaultListen is the Prometheus metrics port.
	// Allocated in https://github.com/prometheus/prometheus/wiki/Default-port-allocations
	defaultListen = ":9521"
)

func main() {
	cmdline := parseSettings()

	log.Printf("ruuvi-prometheus listening on %v", cmdline.listen)

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

	terminate := make(chan os.Signal, 1)
	errCh := make(chan error, 1)
	signal.Notify(terminate, syscall.SIGTERM, syscall.SIGINT)

	// HTTP listener
	go func() {
		err := server.ListenAndServe()
		errCh <- err
	}()

	// Bluetooth scanner
	go func() {
		observations, err := scanner.Scan()
		if err != nil {
			errCh <- err
		}
		for o := range observations {
			metrics.ObserveRuuvi(o)
		}
	}()

	exitCode := 0
	select {
	case sig := <-terminate:
		fmt.Fprintf(os.Stderr, "Exiting on signal %v\n", sig)
	case err := <-errCh:
		fmt.Fprintf(os.Stderr, "Exiting on error: %v\n", err)
		exitCode = 1
	}

	scanner.Shutdown()
	server.Shutdown(context.TODO())

	os.Exit(exitCode)
}

func getDebugLogger(debug bool) *log.Logger {
	var output io.Writer = os.Stderr
	if !debug {
		output = ioutil.Discard
	}
	return log.New(output, "DEBUG: ", log.LstdFlags)
}
