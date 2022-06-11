package hci

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"

	"gitlab.com/jtaimisto/bluewalker/logging"
)

// Transport allows sending and receiving raw HCI packets
// Use hci.Raw() to create transport
type Transport interface {
	// Close closes the transport
	Close()
	Read() ([]byte, error)
	Write(buffer []byte) error
}

// ErrReadAgain is returned by Transport.Read() if the read was
// interrupted (by EINTR or by timeout) and should be tried again.
// ErrReadAgain wraps the original error.
type ErrReadAgain struct {
	orig error
}

func (err ErrReadAgain) Error() string {
	return fmt.Sprintf("Try again (%v)", err.orig)
}

func (err ErrReadAgain) Unwrap() error {
	return err.orig
}

const (
	hciCommandPacket byte = 0x01
	hciACLPacket     byte = 0x02
	// HciEventPacket indicates that data from transport contains HCI Event
	HciEventPacket byte = 0x04
)

//AdvChannelMap defines the advertising channel(s) to use
// See Bluetooth 5.0, vol 2, parth E, ch 7.8.5
type AdvChannelMap int

//Values for AdvChannelMap
const (
	AdvChannel37  AdvChannelMap = 0x01
	AdvChannel38  AdvChannelMap = 0x01 << 1
	AdvChannel39  AdvChannelMap = 0x01 << 2
	AdvChannelAll AdvChannelMap = (AdvChannel37 | AdvChannel38 | AdvChannel39)
)

//AdvFilterPolicy defines what scan and connection requests are accepted
type AdvFilterPolicy byte

//Values for AdvFilterPolicy
const (
	//Scan and connection requests from all
	ScanConnAll AdvFilterPolicy = 0x00
	//Scan requests from all connection requests from white list
	ScanAllConnWhite AdvFilterPolicy = 0x01
	//Scan requests from white list and connect requests from all
	ScanWhiteConnAll AdvFilterPolicy = 0x02
	// Scan and connection requests from white list
	ScanConnWhite = 0x03
)

//AdvAddressType defines values for 'Own Address Type' advertising parameter.
// See Bluetooth 5.0, vol 2, part E, ch 7.8.5
type AdvAddressType byte

// Allowed values for AdvAddressType
const (
	AdvAddressPublic           AdvAddressType = 0x00
	AdvAddressRandom           AdvAddressType = 0x01
	AdvAddressGenerateOrPublic AdvAddressType = 0x02
	AdvAddressGenerateOrRandom AdvAddressType = 0x03
)

// AdvType defines the Advertising Event Type
// See Bluetooth 5.0, vol 2, part E, ch 7.7.65.2
type AdvType byte

// Advertising type values
const (
	AdvInd          AdvType = 0x00
	AdvDirectInd    AdvType = 0x01
	AdvScanInd      AdvType = 0x02
	AdvNonconnInd   AdvType = 0x03
	ScanRsp         AdvType = 0x04
	AdvDirectIndLow AdvType = 0x04 // DIRECT_IND, low duty cycle, when advertising
)

const (
	connUndirectedStr = "Connectable undirected"
	connDirectedStr   = "Connectable directed"
	scanUndirectedStr = "Scannable undirected"
	nonConnUndirStr   = "Non connectable undirected"
	scanRspStr        = "Scan response"
)

func (adv AdvType) String() string {
	switch adv {
	case AdvInd:
		return connUndirectedStr
	case AdvDirectInd:
		return connDirectedStr
	case AdvScanInd:
		return scanUndirectedStr
	case AdvNonconnInd:
		return nonConnUndirStr
	case ScanRsp:
		return scanRspStr
	default:
		return fmt.Sprintf("Unknown (%.2x)", int(adv))
	}
}

//MarshalJSON marshals AdvType into JSON
func (adv AdvType) MarshalJSON() ([]byte, error) {
	return json.Marshal(adv.String())
}

//UnmarshalJSON decodes the JSON encoded AdvertisingType
func (adv *AdvType) UnmarshalJSON(b []byte) error {

	var str string
	err := json.Unmarshal(b, &str)
	if err != nil {
		return err
	}

	switch str {
	case connUndirectedStr:
		*adv = AdvInd
	case connDirectedStr:
		*adv = AdvDirectInd
	case scanUndirectedStr:
		*adv = AdvScanInd
	case nonConnUndirStr:
		*adv = AdvNonconnInd
	case scanRspStr:
		*adv = ScanRsp
	default:
		return fmt.Errorf("invalid Advertising type value")
	}
	return nil
}

//AdvertisingParameters can be used to set advertising parameters for controller
// See Bluetooth 5.0 vol 2, part E, ch 7.8.5
type AdvertisingParameters struct {
	IntervalMin  uint16
	IntervalMax  uint16
	Type         AdvType
	OwnAddrType  AdvAddressType
	PeerAddress  BtAddress
	ChannelMap   AdvChannelMap
	FilterPolicy AdvFilterPolicy
}

//DefaultAdvParameters returns AdvertisingParameters struct with all values set to defaults.
func DefaultAdvParameters() AdvertisingParameters {

	return AdvertisingParameters{
		IntervalMin:  0x0800,
		IntervalMax:  0x0800,
		Type:         AdvNonconnInd,
		OwnAddrType:  AdvAddressPublic,
		ChannelMap:   AdvChannelAll,
		FilterPolicy: ScanConnAll,
	}
}

//AdvertisingReport represents data parsed from LE Advertising Report
//Event received from controller
// See Bluetooth 5.0, vol 2, part E, ch 7.7.65.2
type AdvertisingReport struct {
	EventType AdvType
	Address   BtAddress
	Data      []*AdStructure
	Rssi      int8
}

func (r *AdvertisingReport) String() string {
	return fmt.Sprintf("%s from %s (%s) with %d bytes of data, RSSI %d", r.EventType.String(), r.Address.String(), r.Address.Atype.String(), len(r.Data), r.Rssi)
}

// AdType is the type for advertising data
// See Bluetooth 5.0, vol 3, part C, ch 11
type AdType byte

// AD type values
// See https://www.bluetooth.com/specifications/assigned-numbers/generic-access-profile
const (
	AdFlags                  AdType = 0x01
	AdMore16BitService       AdType = 0x02
	AdComplete16BitService   AdType = 0x03
	AdMore32BitService       AdType = 0x04
	AdComplete32BitService   AdType = 0x05
	AdMore128BitService      AdType = 0x06
	AdComplete128BitService  AdType = 0x07
	AdShortenedLocalName     AdType = 0x08
	AdCompleteLocalName      AdType = 0x09
	AdTxPower                AdType = 0x0a
	AdClassOfdevice          AdType = 0x0d
	AdPairingHash            AdType = 0x0e
	AdPairingRandomizer      AdType = 0x0f
	AdSmTk                   AdType = 0x10
	AdSmOobFlags             AdType = 0x11
	AdSlaveConnInterval      AdType = 0x12
	Ad16bitServiceSol        AdType = 0x14
	Ad128bitServiceSol       AdType = 0x15
	AdServiceData            AdType = 0x16
	AdPublicTargetAddr       AdType = 0x17
	AdRandomTargetAddr       AdType = 0x18
	AdAppearance             AdType = 0x19
	AdAdvInterval            AdType = 0x1a
	AdDeviceAddress          AdType = 0x1b
	AdLeRole                 AdType = 0x1c
	AdPairingHash256         AdType = 0x1d
	AdPairingRandomizer256   AdType = 0x1e
	Ad32BitServiceSol        AdType = 0x1f
	AdServiceData32          AdType = 0x20
	AdServiceData128         AdType = 0x21
	AdSecureConnConfirm      AdType = 0x22
	AdSecureConnRandom       AdType = 0x23
	AdURI                    AdType = 0x24
	AdIndoorPosit            AdType = 0x25
	AdTransportDiscoveryData AdType = 0x26
	AdLeSupportedFeatures    AdType = 0x27
	AdChannelMapUpdate       AdType = 0x28
	AdMeshPbAdv              AdType = 0x29
	AdMeshMessage            AdType = 0x2a
	AdMeshBeacon             AdType = 0x2b
	Ad3dData                 AdType = 0x3d
	AdManufacturerSpecific   AdType = 0xff
)

// AD Flags bitmap values.
// Specified in Supplement to the Bluetooth Core Specification (CSS version 7), Part A, ch 1.3.1
const (
	// LE Limited Discoverable Mode
	AdFlagLimitedDisc = 0x01
	// LE General Discoverable Mode
	AdFlagGeneralDisc = (0x01 << 1)
	// BR/EDR Not Supported.
	AdFlagNoBrEdr = (0x01 << 2)
	// Simultaneous LE and BR/EDR to Same Device Capable (Controller)
	AdFlagLeBrEdrController = (0x01 << 3)
	// Simultaneous LE and BR/EDR to Same Device Capable (Host)
	AdFlagLeBrEdrHost = (0x01 << 4)
)

func (ad AdType) String() string {
	switch ad {
	case AdFlags:
		return "Flags"
	case AdMore16BitService:
		return "16 Bit Service Class UUID"
	case AdComplete16BitService:
		return "Complete 16 Bit Service Class UUID"
	case AdMore32BitService:
		return "32 Bit Service Class UUID"
	case AdComplete32BitService:
		return "Complete 32 Bit Service Class UUID"
	case AdMore128BitService:
		return "128 Bit Service Class UUID"
	case AdComplete128BitService:
		return "Complete 128 Bit Service Class UUID"
	case AdShortenedLocalName:
		return "Shortened Local name"
	case AdCompleteLocalName:
		return "Complete local name"
	case AdTxPower:
		return "Tx Power"
	case AdClassOfdevice:
		return "Class of device"
	case AdManufacturerSpecific:
		return "Manufacturer Specific"
	case AdDeviceAddress:
		return "LE Bluetooth Device Address"
	case AdAppearance:
		return "Appearance"
	case AdPairingHash:
		return "Simple Pairing Hash"
	case AdPairingRandomizer:
		return "Simple Pairing Randomizer"
	case AdSmTk:
		return "Security Manager TK Value"
	case AdSmOobFlags:
		return "Security Manager OOB Flags"
	case AdSlaveConnInterval:
		return "Slave Connection Interval Range"
	case Ad16bitServiceSol:
		return "List of 16-bit Service Solicitation UUIDs"
	case Ad128bitServiceSol:
		return "List of 128-bit Service Solicitation UUIDs"
	case AdServiceData:
		return "Service Data"
	case AdPublicTargetAddr:
		return "Public Target Address"
	case AdRandomTargetAddr:
		return "Random Target Address"
	case AdAdvInterval:
		return "Advertising interval"
	case AdLeRole:
		return "LE Role"
	case AdPairingHash256:
		return "Simple Pairing Hash C-256"
	case AdPairingRandomizer256:
		return "Simple Pairing Randomizer R-256"
	case Ad32BitServiceSol:
		return "List of 32-bit Service Solicitation UUIDs"
	case AdServiceData32:
		return "Service Data - 32-bit UUID"
	case AdServiceData128:
		return "Service Data - 128-bit UUID"
	case AdSecureConnConfirm:
		return "LE Secure Connections Confirmation Value"
	case AdSecureConnRandom:
		return "LE Secure Connections Random Value"
	case AdURI:
		return "URI"
	case AdIndoorPosit:
		return "Indoor Positioning"
	case AdTransportDiscoveryData:
		return "Transport Discovery Data"
	case AdLeSupportedFeatures:
		return "LE Supported Features"
	case AdChannelMapUpdate:
		return "Channel Map Update Indication"
	case AdMeshPbAdv:
		return "PB-ADV"
	case AdMeshMessage:
		return "Mesh Message"
	case AdMeshBeacon:
		return "Mesh Beacon"
	case Ad3dData:
		return "3D Data"
	default:
		return fmt.Sprintf("Unknown (%.2x)", int(ad))
	}
}

// AdStructure defines advertising data
// See Bluetooth 5.0, vol 3, part C, ch 11
type AdStructure struct {
	Typ  AdType `json:"type"`
	Data []byte `json:"data"`
}

//EncodeTo encodes AdStructure into given byte buffer. The buffer should
//be big enough to hold the data or error is returned. On success, returns
//the number of bytes written.
// See Bluetooth v5.0 vol 3, part C, ch 11
func (ad *AdStructure) EncodeTo(buf []byte) (int, error) {
	length := len(ad.Data) + 1 // (type + data)
	if len(buf) < length+1 {
		return 0, fmt.Errorf("buffer too small to hold AD structure data")
	}
	buf[0] = byte(length)
	buf[1] = byte(ad.Typ)
	copy(buf[2:], ad.Data)
	return length + 1, nil
}

func (ad *AdStructure) String() string {
	return fmt.Sprintf("%s : 0x%x", ad.Typ.String(), ad.Data)
}

func decodeAdStructure(buf []byte) (*AdStructure, error) {
	length := int(buf[0])
	// sometimes zero -length AD structres are used for padding
	if length == 0 {
		return nil, nil
	}
	if length+1 > len(buf) {
		return nil, fmt.Errorf("invalid length for AD Structure")
	}
	t := AdType(buf[1])
	dat := buf[2 : 2+length-1]
	return &AdStructure{Typ: t, Data: dat}, nil
}

// ParseAdData parses the advertising data to ad structres
func parseAdData(buf []byte) ([]*AdStructure, error) {

	offset := 0
	structures := make([]*AdStructure, 0)
	for offset < len(buf) {
		ad, err := decodeAdStructure(buf[offset:])
		if err != nil {
			return nil, err
		}
		if ad == nil {
			// in theory, there could be another structure after
			// 0 -length block.
			offset++
			continue
		}
		structures = append(structures, ad)
		offset += (len(ad.Data) + 2)
	}
	return structures, nil
}

//DecodeAdvertisingReport can be used to decode data received in
//Advertising Report Event. Returns all reports contained in event.
func DecodeAdvertisingReport(buf []byte) ([]*AdvertisingReport, error) {

	eMalformed := fmt.Errorf("malformed data for advertising report")
	rd := bytes.NewReader(buf)
	b, err := rd.ReadByte()
	if err != nil {
		return nil, eMalformed
	}
	numReports := int(b)

	ret := make([]*AdvertisingReport, numReports)
	for i := 0; i < numReports; i++ {
		ret[i] = new(AdvertisingReport)
		b, err := rd.ReadByte()
		if err != nil {
			return nil, eMalformed
		}
		ret[i].EventType = AdvType(b)
		// Read the address, first byte is address type
		b, err = rd.ReadByte()
		if err != nil {
			return nil, eMalformed
		}
		addrBytes := make([]byte, 6)
		n, err := rd.Read(addrBytes)
		if n != len(addrBytes) || err != nil {
			return nil, eMalformed
		}
		ret[i].Address = ToBtAddress(addrBytes)
		if b == 0 {
			ret[i].Address.Atype = LePublicAddress
		} else {
			ret[i].Address.Atype = LeRandomAddress
		}
		// Read the AD Structure data, length first
		b, err = rd.ReadByte()
		if err != nil {
			return nil, eMalformed
		}
		if b > 0 {
			advData := make([]byte, int(b))
			n, err = rd.Read(advData)
			if n != len(advData) || err != nil {
				return nil, eMalformed
			}
			ret[i].Data, err = parseAdData(advData)
			if err != nil {
				return nil, fmt.Errorf("malformed data in AD Structures: %s", err.Error())
			}
		} else {
			// initialize to empty slice, not nil
			ret[i].Data = []*AdStructure{}
		}
		b, err = rd.ReadByte()
		if err != nil {
			return nil, eMalformed
		}
		ret[i].Rssi = int8(b)
		logging.IfTracing(func(l *log.Logger) {
			l.Printf("Advertising report: %s", ret[i].String())
		})
	}
	return ret, nil
}
