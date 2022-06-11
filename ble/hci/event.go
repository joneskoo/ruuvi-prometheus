package hci

import (
	"encoding/hex"
	"fmt"
	"log"

	"gitlab.com/jtaimisto/bluewalker/logging"
)

// EventCode identifies the HCI Event received
type EventCode byte

// Event codes for incoming HCI events
const (
	EventCodeCommandComplete EventCode = 0x0e
	EventCodeCommandStatus   EventCode = 0x0f
	EventCodeLeMeta          EventCode = 0x3e
)

func (evt EventCode) String() string {
	switch evt {
	case EventCodeCommandComplete:
		return "Command Complete"
	case EventCodeCommandStatus:
		return "Command Status"
	case EventCodeLeMeta:
		return "LE Meta Event"
	default:
		return fmt.Sprintf("Unknown event 0x%.2x", int(evt))
	}
}

// Event structure represents received HCI Event
// See Bluetooth 5.0, vol 2, Part E, ch 5.4.4
type Event struct {
	Code       EventCode
	parameters []byte
}

// DecodeEvent decodes HCI Event from given buffer, error is received if
// event can not be decoded
func DecodeEvent(buf []byte) (*Event, error) {

	logging.IfTracing(func(l *log.Logger) {
		l.Printf("Event:\n%s", hex.Dump(buf))
	})

	op := buf[0]
	plen := int(buf[1])
	if int(plen) > len(buf)-2 {
		return nil, fmt.Errorf("too short event packet, expected at least %d bytes of parameters", plen)
	}
	return &Event{Code: EventCode(op), parameters: buf[2 : plen+2]}, nil
}

// CommandCompleteEvent represents Command Complete HCI Event
// see Bluetooth v5.0, vol2, part E, ch 7.7.14
type CommandCompleteEvent struct {
	Event
}

// LeMetaEvent represents LE Meta Event HCI Event
// See Bluetooth 5.0, vol 2, part E, ch 7.7.65
type LeMetaEvent struct {
	Event
}

const (
	// minimum length for command complete event
	ccMinParamLength  int = 3
	leMetaParamLength int = 1
)

// DecodeCommandComplete returns given event as CommandCompleteEvent
// caller should check that the event is CommandCompleteEvent
func DecodeCommandComplete(evt *Event) (*CommandCompleteEvent, error) {

	if evt.Code != EventCodeCommandComplete {
		return nil, fmt.Errorf("unexpected event code %.2x", evt.Code)
	}
	if len(evt.parameters) < ccMinParamLength {
		return nil, fmt.Errorf("not enough paramaters for Command Complete")
	}
	return &CommandCompleteEvent{Event: *evt}, nil
}

// GetNumHciCommandPackets returns the number of Hci commands parameter value
func (cc *CommandCompleteEvent) GetNumHciCommandPackets() byte {
	return cc.parameters[0]
}

// GetCommandOpcode returns the opcode this command complete event was for
func (cc *CommandCompleteEvent) GetCommandOpcode() CommandOpCode {
	return CommandOpCode(le.Uint16(cc.parameters[1:]))
}

//HasReturnParameters returns true if this command complete event contains
//return parameters
func (cc *CommandCompleteEvent) HasReturnParameters() bool {
	return len(cc.parameters) > 3
}

// GetReturnParameters returns the return parameters for this event
func (cc *CommandCompleteEvent) GetReturnParameters() []byte {
	return cc.parameters[3:]
}

//GetStatusParameter returns the first parameter as status code.
// XXX bounds check
func (cc *CommandCompleteEvent) GetStatusParameter() ErrorCode {
	return ErrorCode(cc.parameters[3])
}

// DecodeLeMeta returns given event as Le Meta Event
func DecodeLeMeta(evt *Event) (*LeMetaEvent, error) {
	if evt.Code != EventCodeLeMeta {
		return nil, fmt.Errorf("unexpected event code 0x%.2x", evt.Code)
	}
	if len(evt.parameters) < leMetaParamLength {
		return nil, fmt.Errorf("not enough parameters for Le Meta Event")
	}
	return &LeMetaEvent{Event: *evt}, nil
}

// SubeventCode for LE Meta Events
type SubeventCode byte

// Subevent types for LE Meta Event
const (
	SubeventAdvertisingReport SubeventCode = 0x02
)

//GetSubeventCode return subevent code parameter value
func (le *LeMetaEvent) GetSubeventCode() SubeventCode {
	return SubeventCode(le.parameters[0])
}

//GetParameters returns parameters in this event, subevent code is not included
func (le *LeMetaEvent) GetParameters() []byte {
	return le.parameters[1:]
}
