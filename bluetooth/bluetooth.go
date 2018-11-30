// Copyright Joonas Kuorilehto 2018.

package bluetooth

import (
	"fmt"
	"log"

	"gitlab.com/jtaimisto/bluewalker/filter"
	"gitlab.com/jtaimisto/bluewalker/hci"
	"gitlab.com/jtaimisto/bluewalker/host"
	"gitlab.com/jtaimisto/bluewalker/ruuvi"
)

type RuuviReading struct {
	// Address is the sensor Bluetooth address.
	Address string
	// RSSI is the received signal strength in dBm.
	RSSI float64
	// Humidity is the measured relative humidity 0..1.
	Humidity float64
	// Temperature is the measured temperature in °C.
	Temperature float64
	// Pressure is the air pressure in hPa.
	Pressure float64
	// AccelerationX is the acceleration sensor X axis reading in g.
	AccelerationX float64
	// AccelerationY is the acceleration sensor Y axis reading in g.
	AccelerationY float64
	// AccelerationZ is the acceleration sensor Z axis reading in g.
	AccelerationZ float64
	// Voltage is the sensor battery voltage in Volts.
	Voltage float64
}

func Listen(device string, debug bool) (<-chan RuuviReading, error) {
	log.Printf("Using device %v", device)

	vendorIsRuuvi := filter.ByVendor(ruuviVendor)

	raw, err := hci.Raw(device)
	if err != nil {
		return nil, fmt.Errorf(`Error while opening RAW HCI socket: %v
Are you running as root and have you run sudo hciconfig %s down?`, err, device)
	}

	host := host.New(raw)
	if err = host.Init(); err != nil {
		return nil, fmt.Errorf("Unable to initialize host: %v", err)
	}

	reportChan, err := host.StartScanning(false, []filter.AdFilter{vendorIsRuuvi})
	if err != nil {
		return nil, fmt.Errorf("Unable to start scanning: %v", err)
	}

	observations := make(chan RuuviReading)
	log.Println("Starting receive loop")
	go func() {
		defer close(observations)
		defer host.Deinit()

		for sr := range reportChan {
			log.Println("Received frame")
			for _, ads := range sr.Data {
				ruuviData, err := ruuvi.Unmarshall(ads.Data)
				if err != nil {
					log.Printf("Unable to parse ruuvi data: %v", err)
					continue
				}

				observations <- RuuviReading{
					Address:       sr.Address.String(),
					RSSI:          float64(sr.Rssi),
					Humidity:      float64(ruuviData.Humidity) / 100,
					Temperature:   float64(ruuviData.Temperature),
					Pressure:      float64(ruuviData.Pressure) / 100,
					AccelerationX: float64(ruuviData.AccelerationX),
					AccelerationY: float64(ruuviData.AccelerationY),
					AccelerationZ: float64(ruuviData.AccelerationZ),
					Voltage:       float64(ruuviData.Voltage) / 1000,
				}
			}
		}
		log.Println("End of receive loop")

	}()

	return observations, nil
}

// Ruuvi Innovations Ltd. vendor id: 1177 (little endian)
// https://www.bluetooth.com/specifications/assigned-numbers/company-identifiers
var ruuviVendor = []byte{0x99, 0x04}
