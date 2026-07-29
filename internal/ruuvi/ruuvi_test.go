package ruuvi

import (
	"encoding/hex"
	"testing"
)

type unmarshallTest struct {
	name   string
	data   string
	result *Data
}

// naDefaults returns a copy of d with the measurements not transmitted
// in data formats 3 and 5 marked as not available, as the decoders do.
func naDefaults(d Data) *Data {
	d.PM25 = PMNA
	d.CO2 = CO2NA
	d.VOCIndex = VOCIndexNA
	d.NOXIndex = NOXIndexNA
	d.Luminosity = LuminosityNA
	d.SoundAvg = SoundNA
	return &d
}

var unmarshallTests = []unmarshallTest{
	{
		"empty",
		"",
		nil,
	},
	{
		"Invalid format",
		"99040a00000000000000000000000000",
		nil,
	},
	{
		"Invalid length v3",
		"990403000000000000000000000000",
		nil,
	},
	{
		"Invalid Length v5",
		"990405000000000000000000000000",
		nil,
	},
	{
		"Invalid length v6",
		"990406000000000000000000000000",
		nil,
	},
	{
		"Zero",
		"99040300000000000000000000000000",
		naDefaults(Data{Humidity: 0, Temperature: 0, Pressure: 50000,
			AccelerationX: 0, AccelerationY: 0, AccelerationZ: 0, Voltage: 0,
			TxPower: TxPowerNA, MoveCount: 255, Seqno: 0xffff,
			DataFormat: FormatV3}),
	},
	{
		"Zero - v5",
		"9904050000000000000000000000000000000000000000000000",
		naDefaults(Data{Temperature: 0, Humidity: 0, Pressure: 50000,
			AccelerationX: 0, AccelerationY: 0, AccelerationZ: 0,
			Voltage: 1600, TxPower: -40, MoveCount: 0, Seqno: 0,
			DataFormat: FormatV5,
		}),
	},
	{
		"Zero - No Company identifier",
		"0300000000000000000000000000",
		naDefaults(Data{Humidity: 0, Temperature: 0, Pressure: 50000,
			AccelerationX: 0, AccelerationY: 0, AccelerationZ: 0, Voltage: 0,
			TxPower: TxPowerNA, MoveCount: 255, Seqno: 0xffff,
			DataFormat: FormatV3}),
	},
	{
		"vector1",
		"990403808145c87dfc1803e8fc180e10",
		naDefaults(Data{
			Humidity: 64, Temperature: -1.69, Pressure: 101325,
			AccelerationX: -1, AccelerationY: 1, AccelerationZ: -1,
			Voltage: 3600, TxPower: TxPowerNA, MoveCount: 255, Seqno: 0xffff,
			DataFormat: FormatV3,
		}),
	},
	{
		"vector2",
		"990403800145c87dfc1803e8fc170e10",
		naDefaults(Data{
			Humidity: 64, Temperature: 1.69, Pressure: 101325,
			AccelerationX: -1, AccelerationY: 1, AccelerationZ: -1.001,
			Voltage: 3600, TxPower: TxPowerNA, MoveCount: 255, Seqno: 0xffff,
			DataFormat: FormatV3,
		}),
	},
	{
		"vector 1 - v5",
		"99040501c4271ac87dfc18fc18fc18af166403e8010203040506",
		naDefaults(Data{
			Temperature: 2.26, Humidity: 25.025, Pressure: 101325,
			AccelerationX: -1, AccelerationY: -1, AccelerationZ: -1,
			Voltage: 3000, TxPower: 4, MoveCount: 100, Seqno: 1000,
			DataFormat: FormatV5,
		}),
	},
	{
		"Vector 2 - v5",
		"990405fe3e9c40fffe03e803e803e8af166403e8010203040506",
		naDefaults(Data{
			Temperature: -2.25, Humidity: 100, Pressure: 115534,
			AccelerationX: 1, AccelerationY: 1, AccelerationZ: 1,
			Voltage: 3000, TxPower: 4, MoveCount: 100, Seqno: 1000,
			DataFormat: FormatV5,
		}),
	},
	{
		"Vector 3 - v5 real world Ruuvi Pro",
		"990405fcbaffffffffff8cfe70fc709576c6f88ac3ddedb16a93",
		naDefaults(Data{
			Temperature: -4.19, Humidity: HumidityNA, Pressure: PressureNA,
			AccelerationX: -0.116, AccelerationY: -0.4, AccelerationZ: -0.912,
			Voltage: 2795, TxPower: 4, MoveCount: 198, Seqno: 63626,
			DataFormat: FormatV5,
		}),
	},
	{
		"Vector 4 - v5 doc: valid data",
		"0512FC5394C37C0004FFFC040CAC364200CDCBB8334C884F",
		naDefaults(Data{
			Temperature: 24.3, Humidity: 53.489998, Pressure: 100044,
			AccelerationX: 0.004, AccelerationY: -0.004, AccelerationZ: 1.036,
			Voltage: 2977, TxPower: 4, MoveCount: 66, Seqno: 205,
			DataFormat: FormatV5,
		}),
	},
	{
		"Vector 5 - v5 doc: maximum values",
		"057FFFFFFEFFFE7FFF7FFF7FFFFFDEFEFFFECBB8334C884F",
		naDefaults(Data{
			Temperature: 163.83499, Humidity: 163.83499, Pressure: 115534,
			AccelerationX: 32.767, AccelerationY: 32.767, AccelerationZ: 32.767,
			Voltage: 3646, TxPower: 20, MoveCount: 254, Seqno: 65534,
			DataFormat: FormatV5,
		}),
	},
	{
		"Vector 6 - v5 doc: minimum values",
		"058001000000008001800180010000000000CBB8334C884F",
		naDefaults(Data{
			Temperature: -163.83499, Humidity: 0, Pressure: 50000,
			AccelerationX: -32.767, AccelerationY: -32.767, AccelerationZ: -32.767,
			Voltage: 1600, TxPower: -40, MoveCount: 0, Seqno: 0,
			DataFormat: FormatV5,
		}),
	},
	{
		"Vector 7 - v5 doc: invalid values",
		"058000FFFFFFFF800080008000FFFFFFFFFFFFFFFFFFFFFF",
		naDefaults(Data{
			Temperature: TemperatureNA, Humidity: HumidityNA, Pressure: PressureNA,
			AccelerationX: AccelerationNA, AccelerationY: AccelerationNA, AccelerationZ: AccelerationNA,
			Voltage: VoltageNA, TxPower: TxPowerNA, MoveCount: MoveCountNA, Seqno: SeqnoNA,
			DataFormat: FormatV5,
		}),
	},
	{
		"Vector 1 - v6 real world",
		"990406" +
			"1297" + // temperature 4759 * 0.005 = 23.795
			"4b7c" + // humidity 19324 * 0.0025 = 48.31
			"c625" + // pressure 50725 + 50000 = 100725
			"0009" + // PM 2.5 = 0.9
			"0309" + // CO2 = 777
			"07" + "00" + // VOC = 14, NOx high bits = 0
			"ff" + // luminosity NA
			"ff" + // sound avg high bits
			"8d" + // measurement sequence = 141
			"94" + // flags: NOx low bit, sound low bit, RTC running
			"beecfd", // MAC
		&Data{
			Temperature: float32(4759) * 0.005,
			Humidity:    float32(19324) * 0.0025,
			Pressure:    100725,
			PM25:        float32(9) * 0.1,
			CO2:         777, VOCIndex: 14, NOXIndex: 1,
			Luminosity: LuminosityNA,
			SoundAvg:   SoundNA,
			Seqno:      141, Calibrating: false,
			Voltage:       VoltageNA,
			AccelerationX: AccelerationNA, AccelerationY: AccelerationNA, AccelerationZ: AccelerationNA,
			TxPower: TxPowerNA, MoveCount: MoveCountNA,
			DataFormat: FormatV6,
		},
	},
	{
		"Vector 2 - v6: valid luminosity and sound",
		"990406" +
			"1297" + // temperature = 23.795
			"4b7c" + // humidity = 48.31
			"c625" + // pressure = 100725
			"0009" + // PM 2.5 = 0.9
			"0309" + // CO2 = 777
			"07" + "00" + // VOC = 14, NOx = 0
			"fe" + // luminosity coded 254 = 65535 lux
			"c8" + // sound avg 400 / 5 + 18 = 98 dBA
			"8d" + // measurement sequence = 141
			"00" + // flags
			"beecfd", // MAC
		&Data{
			Temperature: float32(4759) * 0.005,
			Humidity:    float32(19324) * 0.0025,
			Pressure:    100725,
			PM25:        float32(9) * 0.1,
			CO2:         777, VOCIndex: 14, NOXIndex: 0,
			Luminosity: 65535,
			SoundAvg:   98,
			Seqno:      141, Calibrating: false,
			Voltage:       VoltageNA,
			AccelerationX: AccelerationNA, AccelerationY: AccelerationNA, AccelerationZ: AccelerationNA,
			TxPower: TxPowerNA, MoveCount: MoveCountNA,
			DataFormat: FormatV6,
		},
	},
}

func TestUnmarshall(t *testing.T) {

	for _, test := range unmarshallTests {

		data, err := hex.DecodeString(test.data)
		if err != nil {
			t.Fatalf("Invalid data definition on test %s : %s ", test.name, err.Error())
		}
		t.Run(test.name, func(t *testing.T) {

			ret, err := Unmarshall(data)
			if test.result != nil {
				if err != nil {
					t.Errorf("Unexpected error while unmarshalling: %s", err.Error())
				} else {
					if *test.result != *ret {
						t.Errorf("Unexpected result. Expected %+v, got %+v", *test.result, *ret)
					}
				}
			} else {
				if err == nil {
					t.Errorf("Expected error, got none")

				}
			}
		})
	}
}

// TestV6Validity checks that the validity methods interpret the data
// format 6 "not available" values correctly, including the sequence
// number which has no "not available" marker in data format 6.
func TestV6Validity(t *testing.T) {
	data, err := Unmarshall(mustHex(t,
		"99040612974b7cc625000903090700ffff8d94beecfd"))
	if err != nil {
		t.Fatalf("Unexpected error while unmarshalling: %s", err.Error())
	}
	for name, valid := range map[string]bool{
		"Temperature": data.TemperatureValid(),
		"Humidity":    data.HumidityValid(),
		"Pressure":    data.PressureValid(),
		"PM25":        data.PM25Valid(),
		"CO2":         data.CO2Valid(),
		"VOCIndex":    data.VOCIndexValid(),
		"NOXIndex":    data.NOXIndexValid(),
		"Seqno":       data.SeqnoValid(),
	} {
		if !valid {
			t.Errorf("%sValid() = false, expected true", name)
		}
	}
	for name, valid := range map[string]bool{
		"Luminosity": data.LuminosityValid(),
		"SoundAvg":   data.SoundAvgValid(),
		"Voltage":    data.VoltageValid(),
		"TxPower":    data.TxPowerValid(),
		"MoveCount":  data.MoveCountValid(),
	} {
		if valid {
			t.Errorf("%sValid() = true, expected false", name)
		}
	}
	if data.AccelerationValid() {
		t.Errorf("AccelerationValid() = true for v6 data, expected false")
	}
}

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	data, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("Invalid hex data: %s", err.Error())
	}
	return data
}
