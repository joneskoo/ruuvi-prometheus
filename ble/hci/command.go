package hci

import (
	"encoding/binary"
	"fmt"
)

// CommandOpCode represents the Operation code for HCI command
// See Bluetooth 5.0 vol 2, part E, ch 5.4.1
type CommandOpCode uint16

// Opcodes for HCI Commands
const (
	CommandReset                CommandOpCode = 0x0c03
	CommandSetEventMask         CommandOpCode = 0x0c01
	CommandWriteLeHostSupported CommandOpCode = 0x0c6d
	CommandLeSetEventMask       CommandOpCode = 0x2001
	CommandLeSetRandomAddress   CommandOpCode = 0x2005
	CommandLeSetAdvParameters   CommandOpCode = 0x2006
	CommandLeSetAdvData         CommandOpCode = 0x2008
	CommandLeSetScanResponse    CommandOpCode = 0x2009
	CommandLeSetAdvEnable       CommandOpCode = 0x200a
	CommandLeSetScanParameters  CommandOpCode = 0x200b
	CommandLeSetScanEnable      CommandOpCode = 0x200c
)

func (op CommandOpCode) String() string {
	switch op {
	case CommandReset:
		return "Reset"
	case CommandLeSetEventMask:
		return "Set LE Event Mask"
	case CommandSetEventMask:
		return "Set Event Mask"
	case CommandLeSetRandomAddress:
		return "LE Set Random Address"
	case CommandLeSetAdvParameters:
		return "LE Set Advertising Parameters"
	case CommandLeSetAdvData:
		return "LE Set Advertising Data"
	case CommandLeSetScanResponse:
		return "LE Set Scan Response"
	case CommandLeSetAdvEnable:
		return "LE Set Advertising Enable"
	case CommandWriteLeHostSupported:
		return "Write LE Host Supported"
	case CommandLeSetScanParameters:
		return "LE Set Scan Parameters"
	case CommandLeSetScanEnable:
		return "LE Set Scan Enable"
	default:
		return fmt.Sprintf("Unknown command 0x%.2x", int(op))
	}
}

const (
	headerLength int = 3
)

// the byte order we are using for encoding data
var le binary.ByteOrder = binary.LittleEndian

// CommandPacket defines HCI Command packet
type CommandPacket struct {
	// OpCode for this command
	OpCode     CommandOpCode
	parameters []byte
}

// Parameters add parameters for command packets
// the given byte slice is not copied, do not modify it
func (pkt *CommandPacket) Parameters(params []byte) {
	pkt.parameters = params
}

// Encode encodes the command into properly formed byte array
// See Bluetooth 5.0 vol 2, part E, ch 5.4.1
func (pkt *CommandPacket) Encode() []byte {

	paramlen := 0
	if pkt.parameters != nil {
		paramlen = len(pkt.parameters)
	}
	len := headerLength + paramlen + 1
	ret := make([]byte, len)
	// The RAW transport expects first byte to be the type of packet
	ret[0] = hciCommandPacket
	le.PutUint16(ret[1:], uint16(pkt.OpCode))
	ret[3] = byte(paramlen)
	copy(ret[4:], pkt.parameters)
	return ret
}
