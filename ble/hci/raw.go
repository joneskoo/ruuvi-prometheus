//+build linux

package hci

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"gitlab.com/jtaimisto/bluewalker/logging"
	"golang.org/x/sys/unix"
)

type hciSocket struct {
	fd int
}

func createSocket(devindex int) (*hciSocket, error) {

	fd, err := unix.Socket(unix.AF_BLUETOOTH, unix.SOCK_RAW, unix.BTPROTO_HCI)
	if err != nil {
		return nil, err
	}

	sa := unix.SockaddrHCI{
		Dev: uint16(devindex), Channel: unix.HCI_CHANNEL_USER,
	}

	err = unix.Bind(fd, &sa)
	if err != nil {
		unix.Close(fd)
		return nil, err
	}
	logging.Debug.Printf("Bound fd %d to HCI device %d ", fd, sa.Dev)

	tv := &unix.Timeval{Sec: 1}
	if err = unix.SetsockoptTimeval(fd, unix.SOL_SOCKET, unix.SO_RCVTIMEO, tv); err != nil {
		logging.Warning.Printf("Unable to set RCVTO: %s", err.Error())
	}

	return &hciSocket{fd: fd}, nil
}

// Raw returns new raw HCI transport
func Raw(device string) (Transport, error) {

	if device == "" || !strings.HasPrefix(device, "hci") {
		return nil, fmt.Errorf("Invalid device name %s", device)
	}
	index, err := strconv.Atoi(device[3:])
	if err != nil {
		return nil, fmt.Errorf("Invalid device name %s", err.Error())
	}

	s, err := createSocket(index)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (hci *hciSocket) Close() {
	unix.Close(hci.fd)
}

func (hci *hciSocket) Write(buf []byte) error {

	logging.IfTracing(func(l *log.Logger) {
		l.Printf("Writing:\n%s", hex.Dump(buf))
	})
	n, err := unix.Write(hci.fd, buf)
	if err != nil {
		return err
	}
	if n != len(buf) {
		return fmt.Errorf("Wrote only %d of %d bytes", n, len(buf))
	}
	return nil
}

func (hci *hciSocket) Read() ([]byte, error) {

	buf := make([]byte, 512)
	n, err := unix.Read(hci.fd, buf)
	if err != nil {
		if errors.Is(err, unix.EINTR) || os.IsTimeout(err) {
			return nil, ErrReadAgain{orig: err}
		}
		return nil, err
	}
	return buf[0:n], nil
}
