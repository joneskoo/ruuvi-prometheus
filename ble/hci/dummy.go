//+build darwin

package hci

import "fmt"

// ErrNotImplemented is returned for functions which are not
// implemented by this transport
var ErrNotImplemented = fmt.Errorf("HCI transport not implemented on Darwin")

type dummy struct {
}

// Raw returns RAW HCI transport
func Raw(device string) (Transport, error) {
	return &dummy{}, nil
}

// Close closes the dummy transport
func (d *dummy) Close() {

}

func (d *dummy) Read() ([]byte, error) {
	return nil, ErrNotImplemented
}

func (d *dummy) Write(buffer []byte) error {
	return ErrNotImplemented
}
