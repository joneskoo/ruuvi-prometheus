package hci

import (
	"encoding/binary"
	"testing"
)

func verifyUint16(array []byte, expected uint16, name string, t *testing.T) {

	val := binary.LittleEndian.Uint16(array)
	if val != expected {
		t.Errorf("Invalid value for %s (expected %.4x got %.4x)", name, expected, val)
	}
}

func TestCommandExport(t *testing.T) {

	cmd := &CommandPacket{OpCode: CommandReset}
	raw := cmd.Encode()

	if len(raw) != headerLength+1 {
		t.Errorf("Invalid length for encoded command %d ", len(raw))
	}
	verifyUint16(raw[1:], uint16(CommandReset), "OpCode", t)
	if raw[3] != 0 {
		t.Errorf("Invalid parameter length")
	}
}

func TestEncodeWithParameters(t *testing.T) {

	cmd := &CommandPacket{OpCode: CommandReset}
	param := make([]byte, 2)
	param[0] = 0x01
	param[1] = 0x02
	cmd.Parameters(param)

	raw := cmd.Encode()

	if len(raw) != headerLength+len(param)+1 {
		t.Errorf("Invalid total length for packet")
	}

	if raw[3] != byte(len(param)) {
		t.Errorf("Invalid parameter length")
	}
	for i := 0; i < len(param); i++ {
		if raw[4+i] != param[i] {
			t.Errorf("Parameter %d incorrectly encoded", i)
		}
	}
}
