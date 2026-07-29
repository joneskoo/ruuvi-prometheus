package hci

import (
	"encoding/hex"
	"testing"
)

func TestDecodeValidEvent(t *testing.T) {

	buf := make([]byte, 3)
	buf[0] = byte(EventCodeCommandComplete)
	buf[1] = 1
	buf[2] = 0x12

	ev, err := DecodeEvent(buf)
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}
	if ev.Code != EventCodeCommandComplete {
		t.Errorf("Deoced invalid event code")
	}
	param := ev.parameters
	if len(param) != 1 || param[0] != 0x12 {
		t.Errorf("Decoded invalid parameters")
	}
}

func TestDecodeShortEvent(t *testing.T) {

	buf := make([]byte, 2)
	buf[0] = byte(EventCodeCommandComplete)
	buf[1] = 0

	ev, err := DecodeEvent(buf)
	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}
	if ev.Code != EventCodeCommandComplete {
		t.Errorf("Decoded invalid event code")
	}
	if len(ev.parameters) != 0 {
		t.Errorf("Decoded parameters even there were none")
	}
}

func TestDecodeTooShortEvent(t *testing.T) {

	buf := make([]byte, 2)
	buf[0] = byte(EventCodeCommandComplete)
	buf[1] = 1

	_, err := DecodeEvent(buf)
	if err == nil {
		t.Errorf("Decoded invalid event!")
	}
}

func TestDecodeCommandComplete(t *testing.T) {

	buf, err := hex.DecodeString("0e0401030c00")
	if err != nil {
		t.Fatalf("Invalid hex string for test") // should not happen
	}
	ev, err := DecodeEvent(buf)
	if err != nil {
		t.Fatalf("Invalid event in test") // should not happen
	}
	cc, err := DecodeCommandComplete(ev)
	if err != nil {
		t.Fatalf("Unable to decode command complete")
	}
	if cc.GetCommandOpcode() != CommandReset {
		t.Errorf("Decoded invalid opcode")
	}
	if cc.GetNumHciCommandPackets() != 1 {
		t.Errorf("Decoded invalid num HCI packets")
	}
	if !cc.HasReturnParameters() {
		t.Errorf("Expected CC to have return parameters")
	}
	params := cc.GetReturnParameters()
	if len(params) != 1 || params[0] != 0x00 {
		t.Errorf("Decoded invalid return parameters")
	}
	if cc.GetStatusParameter() != StatusSuccess {
		t.Errorf("Decoded invalid status")
	}
}

func TestCommandCompleteHasNoParamaters(t *testing.T) {
	buf, _ := hex.DecodeString("0e0301030c")
	ev, err := DecodeEvent(buf)
	if err != nil {
		t.Errorf("Unexpected error while parsing event: %v", err)
	}
	cc, err := DecodeCommandComplete(ev)
	if err != nil {
		t.Errorf("Was not able to decode CC with no return parameters")
	}
	if cc.GetCommandOpcode() != CommandReset {
		t.Errorf("Decoded invalid opcode")
	}
	if cc.GetNumHciCommandPackets() != 1 {
		t.Errorf("Decoded invalid num HCI packets")
	}
	if cc.HasReturnParameters() {
		t.Errorf("Expected CC to have no return parameters")
	}
}

func TestDecodeCommandCompleteInvalidOp(t *testing.T) {
	buf, _ := hex.DecodeString("0f0401030c00")
	ev, _ := DecodeEvent(buf)
	_, err := DecodeCommandComplete(ev)
	if err == nil {
		t.Errorf("Decoded command complete with invalid event code")
	}
}
func TestDecodeTooShortCommandComplete(t *testing.T) {

	buf, _ := hex.DecodeString("0e020103")
	ev, _ := DecodeEvent(buf)
	_, err := DecodeCommandComplete(ev)
	if err == nil {
		t.Errorf("Decoded command complete with invalid length")
	}
}

func TestDecodeLeMeta(t *testing.T) {

	buf, _ := hex.DecodeString("3e020201")
	ev, _ := DecodeEvent(buf)

	meta, err := DecodeLeMeta(ev)
	if err != nil {
		t.Fatalf("Unable to decode valid LE Meta event")
	}
	if meta.GetSubeventCode() != SubeventAdvertisingReport {
		t.Errorf("Decoded invalid subevent code")
	}
	param := meta.GetParameters()
	if len(param) != 1 || param[0] != 0x01 {
		t.Errorf("Decoded invalid parameters for LE Meta")
	}
}

func TestDecodeLeMetaInvalidCode(t *testing.T) {
	buf, _ := hex.DecodeString("3f020201")
	ev, _ := DecodeEvent(buf)

	_, err := DecodeLeMeta(ev)
	if err == nil {
		t.Errorf("Decoded LE Meta with invalid opcode")
	}
}

func TestDecodeLeMetaInvalidLength(t *testing.T) {
	buf, _ := hex.DecodeString("3e00")
	ev, _ := DecodeEvent(buf)

	_, err := DecodeLeMeta(ev)
	if err == nil {
		t.Errorf("Decoded LE Meta with invalid length")
	}
}
