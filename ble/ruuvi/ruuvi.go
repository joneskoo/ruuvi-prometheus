// Package ruuvi implements parsing for the vendor specific data sent by
// Ruuvi tag. See https://github.com/ruuvi/ruuvi-sensor-protocols for details
// of Ruuvi data protocols.
//
// This package supports protocol specifications for dataformats
// version 3 (RAWv1), 5 (RAWv2) and 6.
package ruuvi

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
)

// Ruuvi Tag data format as specified in
// https://github.com/ruuvi/ruuvi-sensor-protocols

// Offsets for the ruuvi measurement data
// These are for format version 3
const (
	formatOffset          int = 0
	v3HumidityOffset      int = 1
	v3TempOffset          int = 2
	v3TempFractionOffset  int = 3
	v3PressureOffset      int = 4
	v3AccelerationXOffset int = 6
	v3AccelerationYOffet  int = 8
	v3AccelerationZOffset int = 10
	v3BatteryVoltageOffet int = 12
)

// Offsets for the ruuvi measurement data
// These are for format version 6
const (
	v6TempOffset       int = 1
	v6HumidityOffset   int = 3
	v6PressureOffset   int = 5
	v6PM25Offset       int = 7
	v6CO2Offset        int = 9
	v6VOCOffset        int = 11
	v6NOXOffset        int = 12
	v6LuminosityOffset int = 13
	v6SoundAvgOffset   int = 14
	v6SeqnoOffset      int = 15
	v6FlagsOffset      int = 16
)

// Flag bits for data format 6
const (
	flagCalibrating  uint8 = 0x01 // sensor calibration in progress
	flagSoundAvgBit9 uint8 = 0x10 // least significant bit of average sound level
	flagVOCBit9      uint8 = 0x40 // least significant bit of VOC index
	flagNOXBit9      uint8 = 0x80 // least significant bit of NOx index
)

// Sound level is a 9 bit value transmitted as dBA above 18 dBA in steps
// of 0.2 dBA. Luminosity in data format 6 is a logarithmically coded
// 8 bit value.
const (
	soundMin           float32 = 18.0
	soundRatio         float32 = 5.0
	v6LuminosityMax    float64 = 65535
	v6LuminosityValues float64 = 254
)

// number of bytes of data to expect
const (
	v3DataLength int = 14
	v5DataLength int = 24
	v6DataLength int = 20
)

// Data format identifiers as transmitted in the first byte of the
// vendor specific data (after the manufacturer ID).
const (
	FormatV3 int = 3
	FormatV5 int = 5
	FormatV6 int = 6
)

// Values returned for Invalid / Not available" values.
//
// If Ruuvi tag does not have a sensor or data is not available, the following
// values are returned.
const (
	// TemperatureNA indicates temperature is invalid or not available.
	TemperatureNA float32 = -163.84 // 0x8000
	// PressureNA indicates pressure is invalid or not available.
	PressureNA int = 115535 // 65535
	// HumidityNA indicates humidity is invalid or not available.
	HumidityNA float32 = 163.8375 // 65535
	// VoltageNA indicates voltage is invalid or not available.
	VoltageNA int = 3647 // 2047
	// AccelerationNA indicates acceleration is invalid or not available.
	AccelerationNA float32 = -32.768 // 0x8000
	// TxPowerNA indicates transmit power is invalid or not available.
	TxPowerNA int = 22 // 31
	// MoveCountNA indicates movement counter is invalid or not available.
	MoveCountNA int = 255
	// SeqnoNA indicates sequence number is invalid or not available.
	SeqnoNA int = 0xffff
	// PMNA indicates a particulate matter measurement is invalid or not
	// available.
	PMNA float32 = 6553.5 // 65535
	// CO2NA indicates CO2 concentration is invalid or not available.
	CO2NA int = 65535
	// VOCIndexNA indicates VOC index is invalid or not available.
	VOCIndexNA int = 511
	// NOXIndexNA indicates NOx index is invalid or not available.
	NOXIndexNA int = 511
)

// These are variables rather than constants so that the values are
// computed with the same float32 roundings as the decoder applies to
// received measurements; a constant expression would be rounded once at
// full precision and not compare equal.
var (
	// LuminosityNA indicates luminosity is invalid or not available.
	LuminosityNA float32 = float32(uint32(0xffffff)) * 0.01 // 16777215
	// SoundNA indicates a sound level is invalid or not available.
	SoundNA float32 = float32(uint16(0x1ff))/soundRatio + soundMin // 511
)

// Data contains measurement information parsed from the vendor specific
// data sent by Ruuvi tag. See
// https://github.com/ruuvi/ruuvi-sensor-protocols for details of Ruuvi
// data protocols.
//
// If Ruuvi tag does not have a sensor or data is not available, the following
// values are returned accordingly: TemperatureNA, PressureNA, HumidityNA,
// VoltageNA, AccelerationNA, TxPowerNA, MoveCountNA, SeqnoNA.
//
// Ruuvi tag data protocol version 3 does not contain the TxPower, MoveCount
// or Seqno. Some models of Ruuvi tag may not contain all sensors.
type Data struct {
	Humidity      float32 `json:"humidity"`
	Temperature   float32 `json:"temperature"`
	Pressure      int     `json:"pressure"`
	AccelerationX float32 `json:"accelerationX"`
	AccelerationY float32 `json:"accelerationY"`
	AccelerationZ float32 `json:"accelerationZ"`
	Voltage       int     `json:"voltage"`
	// V5 Only
	TxPower   int `json:"txpower"`
	MoveCount int `json:"movementCount"`
	Seqno     int `json:"sequence"`
	// V6 Only
	PM25        float32 `json:"pm25"`
	CO2         int     `json:"co2"`
	VOCIndex    int     `json:"vocIndex"`
	NOXIndex    int     `json:"noxIndex"`
	Luminosity  float32 `json:"luminosity"`
	SoundAvg    float32 `json:"soundAvgDba"`
	Calibrating bool    `json:"calibrating"`
	// DataFormat is the Ruuvi data format the measurement was decoded
	// from: FormatV3, FormatV5 or FormatV6.
	DataFormat int `json:"dataFormat"`
}

// TemperatureValid is true when Temperature is valid.
func (d Data) TemperatureValid() bool { return d.Temperature != TemperatureNA }

// PressureValid is true when Pressure is valid.
func (d Data) PressureValid() bool { return d.Pressure != PressureNA }

// HumidityValid is true when Humidity is valid.
func (d Data) HumidityValid() bool { return d.Humidity != HumidityNA }

// VoltageValid is true when Voltage is valid.
func (d Data) VoltageValid() bool { return d.Voltage != VoltageNA }

// TxPowerValid is true when TxPower is valid.
func (d Data) TxPowerValid() bool { return d.TxPower != TxPowerNA }

// MoveCountValid is true when MoveCount is valid.
func (d Data) MoveCountValid() bool { return d.MoveCount != MoveCountNA }

// SeqnoValid is true when Seqno is valid.
func (d Data) SeqnoValid() bool {
	switch d.DataFormat {
	case FormatV6:
		// data format 6 has no "not available" marker for the
		// sequence number
		return true
	default:
		return d.Seqno != SeqnoNA
	}
}

// PM25Valid is true when PM25 is valid.
func (d Data) PM25Valid() bool { return d.PM25 != PMNA }

// CO2Valid is true when CO2 is valid.
func (d Data) CO2Valid() bool { return d.CO2 != CO2NA }

// VOCIndexValid is true when VOCIndex is valid.
func (d Data) VOCIndexValid() bool { return d.VOCIndex != VOCIndexNA }

// NOXIndexValid is true when NOXIndex is valid.
func (d Data) NOXIndexValid() bool { return d.NOXIndex != NOXIndexNA }

// LuminosityValid is true when Luminosity is valid.
func (d Data) LuminosityValid() bool { return d.Luminosity != LuminosityNA }

// SoundAvgValid is true when SoundAvg is valid.
func (d Data) SoundAvgValid() bool { return d.SoundAvg != SoundNA }

// AccelerationValid is true when AccelerationX, AccelerationY and AccelerationZ
// are valid.
func (d Data) AccelerationValid() bool {
	var valid bool
	switch {
	case d.AccelerationX == AccelerationNA:
		valid = false
	case d.AccelerationY == AccelerationNA:
		valid = false
	case d.AccelerationZ == AccelerationNA:
		valid = false
	default:
		valid = true
	}
	return valid
}

func readAccl(rd *bytes.Reader) float32 {
	var accl int16
	binary.Read(rd, binary.BigEndian, &accl)
	return float32(accl) / 1000
}

func decodeV3Data(data []byte) (*Data, error) {
	ret := new(Data)
	ret.Humidity = float32(data[v3HumidityOffset]) / 2

	//MSB is sign, next 7 bits are decimal value
	temp := int(data[v3TempOffset] & 0x7f)
	if data[v3TempOffset]&0x80 == 0x80 {
		temp = -temp
		ret.Temperature = float32(temp) - float32(data[v3TempFractionOffset])/100
	} else {
		ret.Temperature = float32(temp) + float32(data[v3TempFractionOffset])/100
	}

	ret.Pressure = int(binary.BigEndian.Uint16(data[v3PressureOffset:v3PressureOffset+2])) + 50000
	ret.Voltage = int(binary.BigEndian.Uint16(data[v3BatteryVoltageOffet:]))

	rd := bytes.NewReader(data[v3AccelerationXOffset:v3BatteryVoltageOffet])
	ret.AccelerationX = readAccl(rd)
	ret.AccelerationY = readAccl(rd)
	ret.AccelerationZ = readAccl(rd)

	// Not available is signified by largest presentable number for unsigned
	// values, smallest presentable number for signed values
	ret.TxPower = TxPowerNA
	ret.MoveCount = MoveCountNA
	ret.Seqno = SeqnoNA
	ret.PM25 = PMNA
	ret.CO2 = CO2NA
	ret.VOCIndex = VOCIndexNA
	ret.NOXIndex = NOXIndexNA
	ret.Luminosity = LuminosityNA
	ret.SoundAvg = SoundNA

	ret.DataFormat = FormatV3
	return ret, nil
}

// decode9bit assembles a 9 bit value: the byte holds the high 8 bits and
// the least significant bit is stored in the flags byte.
func decode9bit(high byte, flags uint8, lsb uint8) int {
	v := int(high) << 1
	if flags&lsb != 0 {
		v |= 1
	}
	return v
}

// decodeSound converts a 9 bit raw sound level to dBA (or dB SPL).
func decodeSound(raw int) float32 {
	return float32(uint16(raw))/soundRatio + soundMin
}

func decodeV5Data(data []byte) (*Data, error) {

	be := binary.BigEndian
	var s16 int16
	var u16 uint16
	var u8 uint8

	ret := new(Data)
	rd := bytes.NewReader(data[formatOffset+1:])

	binary.Read(rd, be, &s16)
	ret.Temperature = float32(s16) * 0.005
	binary.Read(rd, be, &u16)
	ret.Humidity = float32(u16) * 0.0025
	binary.Read(rd, be, &u16)
	ret.Pressure = int(u16) + 50000

	ret.AccelerationX = readAccl(rd)
	ret.AccelerationY = readAccl(rd)
	ret.AccelerationZ = readAccl(rd)
	// power info
	binary.Read(rd, be, &u16)
	// Power info (11+5bit unsigned), first 11bits unsigned is the battery
	// voltage above 1.6V, in millivolts (1.6V to 3.647V range). last 5 bits
	// unsigned is the TX power above -40dBm, in 2dBm steps.
	tmp := (u16 & 0xFFE0) >> 5
	ret.Voltage = int(1600 + tmp)
	tmp = (u16 & 0x001f)
	ret.TxPower = -40 + int((tmp * 2))

	binary.Read(rd, be, &u8)
	ret.MoveCount = int(u8)
	binary.Read(rd, be, &u16)
	ret.Seqno = int(u16)

	// Measurements only available in data format 6 are marked not
	// available.
	ret.PM25 = PMNA
	ret.CO2 = CO2NA
	ret.VOCIndex = VOCIndexNA
	ret.NOXIndex = NOXIndexNA
	ret.Luminosity = LuminosityNA
	ret.SoundAvg = SoundNA

	ret.DataFormat = FormatV5
	return ret, nil
}

func decodeV6Data(data []byte) (*Data, error) {

	be := binary.BigEndian

	ret := new(Data)
	ret.Temperature = float32(int16(be.Uint16(data[v6TempOffset:]))) * 0.005
	ret.Humidity = float32(be.Uint16(data[v6HumidityOffset:])) * 0.0025
	ret.Pressure = int(be.Uint16(data[v6PressureOffset:])) + 50000
	ret.PM25 = float32(be.Uint16(data[v6PM25Offset:])) * 0.1
	ret.CO2 = int(be.Uint16(data[v6CO2Offset:]))

	flags := data[v6FlagsOffset]
	ret.VOCIndex = decode9bit(data[v6VOCOffset], flags, flagVOCBit9)
	ret.NOXIndex = decode9bit(data[v6NOXOffset], flags, flagNOXBit9)
	ret.Calibrating = flags&flagCalibrating != 0

	// Luminosity is transmitted as a logarithmically coded 8 bit value
	if data[v6LuminosityOffset] == 0xff {
		ret.Luminosity = LuminosityNA
	} else {
		ratio := v6LuminosityValues / math.Log(v6LuminosityMax+1)
		ret.Luminosity = float32(math.Exp(float64(data[v6LuminosityOffset])/ratio) - 1)
	}

	ret.SoundAvg = decodeSound(decode9bit(data[v6SoundAvgOffset], flags, flagSoundAvgBit9))

	ret.Seqno = int(data[v6SeqnoOffset])

	// Measurements not available in data format 6 are marked not
	// available.
	ret.Voltage = VoltageNA
	ret.AccelerationX = AccelerationNA
	ret.AccelerationY = AccelerationNA
	ret.AccelerationZ = AccelerationNA
	ret.TxPower = TxPowerNA
	ret.MoveCount = MoveCountNA

	ret.DataFormat = FormatV6
	return ret, nil
}

// Unmarshall parses Ruuvi Data from given byte array and returns Data struct
// containing the read values
// Renamed to Decode(), this alias is here just for backwards compatability
func Unmarshall(data []byte) (*Data, error) {
	return Decode(data)
}

// Decode decodes Ruuvi data from given advertising data. Returned Data
// structrure contains the decoded values.
func Decode(data []byte) (*Data, error) {

	// Check if the data contains the Ruuvi Manufacturer ID, strip it
	// Otherwise we assume the Manufacturer ID is already stripped
	if len(data) >= 2 && binary.LittleEndian.Uint16(data) == 0x0499 {
		data = data[2:]
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("not enough data for Ruuvi data")
	}
	format := data[formatOffset]
	switch int(format) {
	case FormatV3:
		if len(data) < v3DataLength {
			return nil, fmt.Errorf("expected at least %d bytes of data, got %d", v3DataLength, len(data))
		}
		return decodeV3Data(data)
	case FormatV5:
		if len(data) < v5DataLength {
			return nil, fmt.Errorf("expected at least %d bytes of data, got %d", v5DataLength, len(data))
		}
		return decodeV5Data(data)
	case FormatV6:
		if len(data) < v6DataLength {
			return nil, fmt.Errorf("expected at least %d bytes of data, got %d", v6DataLength, len(data))
		}
		return decodeV6Data(data)
	default:
		return nil, fmt.Errorf("ruuvi Data format %d not supported", data[formatOffset])
	}
}
