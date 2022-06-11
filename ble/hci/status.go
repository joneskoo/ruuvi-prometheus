package hci

import "fmt"

// ErrorCode defines the error codes returned by controller
// See Bluetooth 5.0, vol 2, part D
type ErrorCode byte

//Defined Error codes
const (
	StatusSuccess           ErrorCode = 0x00
	StatusUnknownCommand    ErrorCode = 0x01
	StatusCommandDisallowed ErrorCode = 0x0c
	StatusInvalidParams     ErrorCode = 0x12
)

func (e ErrorCode) String() string {
	switch e {
	case StatusSuccess:
		return "Success"
	case StatusUnknownCommand:
		return "Unknown HCI Command"
	case StatusCommandDisallowed:
		return "Command Disallowed"
	case StatusInvalidParams:
		return "Invalid command parameters"
	default:
		return fmt.Sprintf("Unknown error: 0x%.2x", byte(e))
	}
}
