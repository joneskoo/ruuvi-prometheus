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
	"time"

	"github.com/joneskoo/ruuvi-prometheus/bluetooth"
	"github.com/joneskoo/ruuvi-prometheus/metrics"
	"gitlab.com/jtaimisto/bluewalker/host"
	"gitlab.com/jtaimisto/bluewalker/ruuvi"
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
	scanner.HandleAdvertisement(handleRuuviAdvertisement)
	err := scanner.Scan()
	if err != nil {
		errCh <- err
	}

	// Expire metrics unless receiving data once per minute
	go func() {
		for range time.Tick(time.Minute) {
			metrics.ClearExpired()
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

	// scanner.Shutdown()
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

func handleRuuviAdvertisement(sr *host.ScanReport) {
	for _, ads := range sr.Data {
		ruuviData, err := ruuvi.Unmarshall(ads.Data)
		if err != nil {
			log.Printf("Unable to parse ruuvi data: %v", err)
			continue
		}

		reading := ruuviReading{sr, ruuviData}
		metrics.ObserveRuuvi(reading)
	}
}

type ruuviReading struct {
	sr   *host.ScanReport
	data *ruuvi.Data
}

func (r ruuviReading) Address() string        { return r.sr.Address.String() }
func (r ruuviReading) RSSI() float64          { return float64(r.sr.Rssi) }
func (r ruuviReading) Humidity() float64      { return float64(r.data.Humidity) / 100 }
func (r ruuviReading) Temperature() float64   { return float64(r.data.Temperature) }
func (r ruuviReading) Pressure() float64      { return float64(r.data.Pressure) / 100 }
func (r ruuviReading) AccelerationX() float64 { return float64(r.data.AccelerationX) }
func (r ruuviReading) AccelerationY() float64 { return float64(r.data.AccelerationY) }
func (r ruuviReading) AccelerationZ() float64 { return float64(r.data.AccelerationZ) }
func (r ruuviReading) Voltage() float64       { return float64(r.data.Voltage) / 1000 }
