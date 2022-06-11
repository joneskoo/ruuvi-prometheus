package host

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"
	"time"

	"gitlab.com/jtaimisto/bluewalker/hci"
)

type testTransport struct {
	received [][]byte
	rfunc    func() ([]byte, error)
	wfunc    func([]byte) error
}

func (tt *testTransport) Close() {

}

func (tt *testTransport) Read() ([]byte, error) {
	if tt.rfunc == nil {
		return make([]byte, 0), nil
	}
	return tt.rfunc()
}

func (tt *testTransport) Write(data []byte) error {
	if tt.wfunc == nil {
		tt.received = append(tt.received, data)
		return nil
	}
	return tt.wfunc(data)
}

func mkCommandCompleteEvent(status hci.ErrorCode, op hci.CommandOpCode, nrCompleted int, t *testing.T) *hci.CommandCompleteEvent {

	buf := make([]byte, 6)
	buf[0] = byte(hci.EventCodeCommandComplete)
	buf[1] = 4 // length
	buf[2] = byte(nrCompleted)
	binary.LittleEndian.PutUint16(buf[3:], uint16(op))
	buf[5] = byte(status)

	evt, err := hci.DecodeEvent(buf)
	if err != nil {
		t.Fatalf("Can not create event: %s", err.Error())
	}
	cc, err := hci.DecodeCommandComplete(evt)
	if err != nil {
		t.Fatalf("Can not create event: %s", err.Error())
	}
	return cc
}

func TestCommandExecSuccess(t *testing.T) {

	h := New(nil)
	cmd := hci.CommandPacket{OpCode: hci.CommandReset}

	cc := mkCommandCompleteEvent(0, hci.CommandReset, 1, t)

	ch := make(chan error)
	go func(errChan chan error, t *testing.T) {
		err := h.executeStatusCommand(&cmd)
		errChan <- err
	}(ch, t)
	exec := <-h.cmd
	exec.complete(cc)
	err := <-ch
	if err != nil {
		t.Errorf("Command execution failed unexpectedly")
	}
}

func TestCommandExecFail(t *testing.T) {

	h := New(nil)
	cmd := hci.CommandPacket{OpCode: hci.CommandReset}

	cc := mkCommandCompleteEvent(hci.StatusInvalidParams, hci.CommandReset, 1, t)

	ch := make(chan error)

	go func(errChan chan error, t *testing.T) {
		err := h.executeStatusCommand(&cmd)
		errChan <- err
	}(ch, t)
	exec := <-h.cmd
	exec.complete(cc)
	err := <-ch
	if err == nil {
		t.Errorf("Expected the command execution to fail")
	}
	var execFailErr *CommandExecutionError
	if errors.As(err, &execFailErr) {
		if execFailErr.ErrorCode() != hci.StatusInvalidParams {
			t.Errorf("Unexpected status code in returned error: %s", execFailErr.ErrorCode())
		}
		if execFailErr.op != hci.CommandReset {
			t.Errorf("Unexpected OP code in returned error: %s", execFailErr.op)
		}
	} else {
		t.Errorf("Expected CommandExecutionError to be returned")
	}
}

func TestCommandExecFail2(t *testing.T) {

	h := New(nil)
	cmd := hci.CommandPacket{OpCode: hci.CommandReset}

	ch := make(chan error)

	go func(errChan chan error, t *testing.T) {
		err := h.executeStatusCommand(&cmd)
		errChan <- err
	}(ch, t)
	exec := <-h.cmd
	exec.fail(fmt.Errorf("Execution failure"))
	err := <-ch
	if err == nil {
		t.Errorf("Expected the command execution to fail")
	}
}

func TestCommandExecWithCCWithoutStatus(t *testing.T) {
	h := New(nil)
	cmd := hci.CommandPacket{OpCode: hci.CommandReset}
	ch := make(chan error)

	buf := make([]byte, 5)
	buf[0] = byte(hci.EventCodeCommandComplete)
	buf[1] = 3 // length
	buf[2] = 0x01
	binary.LittleEndian.PutUint16(buf[3:], uint16(hci.CommandReset))

	evt, _ := hci.DecodeEvent(buf)
	cc, _ := hci.DecodeCommandComplete(evt)

	go func(errChan chan error) {
		errChan <- h.executeStatusCommand(&cmd)
	}(ch)
	exec := <-h.cmd
	exec.complete(cc)

	err := <-ch
	if err == nil {
		t.Errorf("Expected execution to fail")
	}
}

func TestExecutorHappy(t *testing.T) {

	trp := new(testTransport)
	trp.received = make([][]byte, 0)
	h := New(trp)

	cmd := hci.CommandPacket{OpCode: hci.CommandReset}

	ch := make(chan int)

	ex := exec{cmd: &cmd,
		complete: func(cc *hci.CommandCompleteEvent) {
			ch <- 1
		}, fail: func(er error) {

		}}
	cc := mkCommandCompleteEvent(hci.StatusSuccess, hci.CommandReset, 1, t)

	go h.executor()
	h.cmd <- &ex
	h.cc <- cc

	// Wait for ex.complete() to be called
	<-ch

	// Check that the command was written
	if len(trp.received) != 1 {
		t.Errorf("Command was not written to transport")
	}
	close(h.cmd)
	close(h.cc)
}

func TestExecutorWriteFail(t *testing.T) {

	trp := new(testTransport)
	trp.wfunc = func(buf []byte) error {
		return fmt.Errorf("Test write failed")
	}
	h := New(trp)

	cmd := hci.CommandPacket{OpCode: hci.CommandReset}

	ch := make(chan int)

	ex := exec{cmd: &cmd,
		complete: func(cc *hci.CommandCompleteEvent) {
		}, fail: func(er error) {
			ch <- 1
		}}

	go h.executor()
	h.cmd <- &ex

	// Wait for ex.fail() to be called
	<-ch
	close(h.cmd)
	close(h.cc)
}

func TestExecutorTimeout(t *testing.T) {
	h := New(new(testTransport))
	// tweak the timeout value, we don't want to wait for 30 seconds
	cmdExecutionTimeout = time.Duration(5 * time.Second)
	cmd := hci.CommandPacket{OpCode: hci.CommandReset}

	ch := make(chan int)
	ex := exec{cmd: &cmd,
		complete: func(cc *hci.CommandCompleteEvent) {
			t.FailNow()
		},
		fail: func(er error) {
			ch <- 1
		},
	}
	go h.executor()
	h.cmd <- &ex
	// wait for executor to timeout
	<-ch
	close(h.cmd)
	close(h.cc)
}

func TestExecutorMultipleCC(t *testing.T) {

	h := New(new(testTransport))
	cmd := hci.CommandPacket{OpCode: hci.CommandReset}

	ch := make(chan int)
	ex := exec{cmd: &cmd,
		complete: func(cc *hci.CommandCompleteEvent) {
			ch <- 1
		},
		fail: func(er error) {
			t.FailNow()
		},
	}

	ccExpected := mkCommandCompleteEvent(hci.StatusSuccess, hci.CommandReset, 1, t)
	ccUnexpected := mkCommandCompleteEvent(hci.StatusSuccess, hci.CommandLeSetEventMask, 1, t)

	go h.executor()
	h.cmd <- &ex
	h.cc <- ccUnexpected
	h.cc <- ccExpected

	// wait for the completion to be called
	<-ch

	close(h.cmd)
	close(h.cc)
}

func TestZeroHCICommands(t *testing.T) {

	h := New(new(testTransport))

	cmd := hci.CommandPacket{OpCode: hci.CommandReset}
	cmd2 := hci.CommandPacket{OpCode: hci.CommandLeSetScanEnable}

	ch := make(chan int, 3)

	ex1 := exec{
		cmd: &cmd,
		complete: func(cc *hci.CommandCompleteEvent) {
			ch <- 1
		},
		fail: func(error) {
			t.FailNow()
		},
	}
	ex2 := exec{
		cmd: &cmd2,
		complete: func(cc *hci.CommandCompleteEvent) {
			ch <- 2
		},
		fail: func(error) {
			t.FailNow()
		},
	}

	cc := mkCommandCompleteEvent(hci.StatusSuccess, hci.CommandReset, 0, t)
	cc2 := mkCommandCompleteEvent(hci.StatusSuccess, 0, 1, t)
	cc3 := mkCommandCompleteEvent(hci.StatusSuccess, hci.CommandLeSetScanEnable, 1, t)

	go h.executor()

	h.cmd <- &ex1

	h.cc <- cc
	h.cc <- cc2

	h.cmd <- &ex2

	ret := <-ch
	if ret != 1 {
		t.Fatalf("Unexpected completion")
	}
	h.cc <- cc3

	ret = <-ch
	if ret != 2 {
		t.Fatalf("unexpected completion")
	}
}

func TestEventHandlerCC(t *testing.T) {

	h := New(nil)
	buf := make([]byte, 6)
	buf[0] = byte(hci.EventCodeCommandComplete)
	buf[1] = 4 // length
	buf[2] = 1 // num HCI packets
	binary.LittleEndian.PutUint16(buf[3:], uint16(hci.CommandReset))
	buf[5] = byte(hci.StatusSuccess)

	go h.eventHandler()
	h.evt <- buf
	cc := <-h.cc
	close(h.evt)
	if cc.GetCommandOpcode() != hci.CommandReset || cc.GetStatusParameter() != hci.StatusSuccess {
		t.Errorf("Received unexpected Command Complete event")
	}
}
func TestEventHandlerWithInvalidEvents(t *testing.T) {

	h := New(nil)
	invalid := make([]byte, 6)
	invalid[0] = byte(hci.EventCodeCommandComplete)
	invalid[1] = 8 // length .. invalid
	invalid[2] = 1 // num HCI packets
	binary.LittleEndian.PutUint16(invalid[3:], uint16(hci.CommandReset))
	invalid[5] = byte(hci.StatusSuccess)

	// invald LE Meta event
	invalid2 := make([]byte, 2)
	invalid2[0] = byte(hci.EventCodeLeMeta)
	invalid2[1] = 0

	invalid3 := make([]byte, 2)
	invalid3[0] = 0xff // invalid op code
	invalid3[1] = 0

	buf := make([]byte, 6)
	buf[0] = byte(hci.EventCodeCommandComplete)
	buf[1] = 4 // length
	buf[2] = 1 // num HCI packets
	binary.LittleEndian.PutUint16(buf[3:], uint16(hci.CommandReset))
	buf[5] = byte(hci.StatusSuccess)

	go h.eventHandler()
	h.evt <- invalid
	h.evt <- invalid2
	h.evt <- invalid3
	h.evt <- buf
	cc := <-h.cc
	close(h.evt)
	if cc.GetCommandOpcode() != hci.CommandReset || cc.GetStatusParameter() != hci.StatusSuccess {
		t.Errorf("Received unexpected Command Complete event")
	}
}
func TestEventHandlerWithInvalidCC(t *testing.T) {

	h := New(nil)
	invalid := make([]byte, 6)
	invalid[0] = byte(hci.EventCodeCommandComplete)
	invalid[1] = 2 // length
	binary.LittleEndian.PutUint16(invalid[2:], uint16(hci.CommandReset))

	buf := make([]byte, 6)
	buf[0] = byte(hci.EventCodeCommandComplete)
	buf[1] = 4 // length
	buf[2] = 1 // num HCI packets
	binary.LittleEndian.PutUint16(buf[3:], uint16(hci.CommandReset))
	buf[5] = byte(hci.StatusSuccess)

	go h.eventHandler()
	h.evt <- invalid
	h.evt <- buf
	cc := <-h.cc
	close(h.evt)
	if cc.GetCommandOpcode() != hci.CommandReset || cc.GetStatusParameter() != hci.StatusSuccess {
		t.Errorf("Received unexpected Command Complete event")
	}
}

func TestEventHandlerMetaInvalidAd(t *testing.T) {

	h := New(nil)
	buf, _ := hex.DecodeString("3e1402010000010203040506080309410003084100ff")
	invalid, _ := hex.DecodeString("3e1302010000010203040506080309410003084100")

	go h.eventHandler()
	h.evt <- invalid
	h.evt <- buf
	ad := <-h.ad
	close(h.evt)
	bdaddr := hci.ToBtAddress([]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06})
	if ad.Address != bdaddr {
		// the Advertising report parsing is tested elsewhere
		t.Errorf("Unexpected address in edvertising report")
	}
}

func TestEventHandlerMeta(t *testing.T) {

	h := New(nil)
	buf, err := hex.DecodeString("3e1402010000010203040506080309410003084100ff")
	if err != nil {
		t.Fatalf("Invalid test data (%s)", err.Error())
	}

	go h.eventHandler()
	h.evt <- buf
	ad := <-h.ad
	close(h.evt)
	bdaddr := hci.ToBtAddress([]byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06})
	if ad.Address != bdaddr {
		// the Advertising report parsing is tested elsewhere
		t.Errorf("Unexpected address in edvertising report")
	}
}

func TestEventReceiver(t *testing.T) {

	trp := new(testTransport)
	count := 0
	trp.rfunc = func() ([]byte, error) {
		if count > 1 {
			// we need to return error to trigger closing
			// of the event loop. Ugly, but works
			return nil, fmt.Errorf("Intentional error")
		}
		count++
		return hex.DecodeString("040e0401030c00")
	}
	h := New(trp)
	h.wg.Add(1)
	go h.eventReceiver()

	buf := <-h.evt
	h.mux.Lock()
	h.closing = true
	h.mux.Unlock()

	rcv, err := hci.DecodeEvent(buf)
	if err != nil {
		t.Errorf("Received malformed event: %s", err.Error())
	}
	if rcv.Code != hci.EventCodeCommandComplete {
		t.Errorf("Received unexpected event")
	}
	h.wg.Wait()
}

const (
	paramStartOffset int = 4
)

type checkfn func(encoded []byte, t *testing.T)

func TestCommand(t *testing.T) {

	tests := []struct {
		name        string
		doCommand   func(h *Host, ch chan error)
		expectedOps []hci.CommandOpCode
		check       []checkfn
		statuses    []hci.ErrorCode
		expectErr   bool
	}{
		{
			"Test init",
			func(h *Host, ch chan error) {
				ch <- h.initializeController()
			},
			[]hci.CommandOpCode{hci.CommandReset, hci.CommandWriteLeHostSupported, hci.CommandSetEventMask, hci.CommandLeSetEventMask},
			[]checkfn{nil, nil, nil, nil},
			[]hci.ErrorCode{hci.StatusSuccess, hci.StatusSuccess, hci.StatusSuccess, hci.StatusSuccess},
			false,
		},
		{
			"Test init Fail",
			func(h *Host, ch chan error) {
				ch <- h.initializeController()
			},
			[]hci.CommandOpCode{hci.CommandReset, hci.CommandWriteLeHostSupported},
			[]checkfn{nil, nil},
			[]hci.ErrorCode{hci.StatusSuccess, hci.StatusInvalidParams},
			true,
		},
		{
			"Start advertising",
			func(h *Host, ch chan error) {
				ch <- h.StartAdvertising()
			},
			[]hci.CommandOpCode{hci.CommandLeSetAdvEnable},
			[]checkfn{func(encoded []byte, t *testing.T) {
				if len(encoded) != paramStartOffset+1 {
					t.Errorf("Invalid lenght for parameters")
				}
				if encoded[paramStartOffset] != 0x01 {
					t.Errorf("Unexpected parameter")
				}

			}},
			[]hci.ErrorCode{hci.StatusSuccess},
			false,
		},
		{
			"Start advertising fail",
			func(h *Host, ch chan error) {
				ch <- h.StartAdvertising()
			},
			[]hci.CommandOpCode{hci.CommandLeSetAdvEnable},
			[]checkfn{nil},
			[]hci.ErrorCode{hci.StatusInvalidParams},
			true,
		},
		{
			"Stop advertising",
			func(h *Host, ch chan error) {
				ch <- h.StopAdvertising()
			},
			[]hci.CommandOpCode{hci.CommandLeSetAdvEnable},
			[]checkfn{func(encoded []byte, t *testing.T) {
				if len(encoded) != paramStartOffset+1 {
					t.Errorf("Invalid lenght for parameters")
				}
				if encoded[paramStartOffset] != 0x00 {
					t.Errorf("Unexpected parameter")
				}

			}},
			[]hci.ErrorCode{hci.StatusSuccess},
			false,
		},
		{
			"Stop advertising fail",
			func(h *Host, ch chan error) {
				ch <- h.StopAdvertising()
			},
			[]hci.CommandOpCode{hci.CommandLeSetAdvEnable},
			[]checkfn{nil},
			[]hci.ErrorCode{hci.StatusInvalidParams},
			true,
		},
		{
			"Start Scanning",
			func(h *Host, ch chan error) {
				_, err := h.StartScanning(false, nil)
				ch <- err
			},
			[]hci.CommandOpCode{hci.CommandLeSetScanParameters, hci.CommandLeSetScanEnable},
			[]checkfn{
				func(encoded []byte, t *testing.T) {
					if len(encoded) != paramStartOffset+7 {
						t.Errorf("Invalid length for set scan parameters")
					}
					if encoded[paramStartOffset] != 0x00 {
						t.Errorf("Expected passive scanning by default")
					}
				},
				func(encoded []byte, t *testing.T) {
					if len(encoded) != paramStartOffset+2 {
						t.Errorf("Invalid length for scan enable")
					}
					if encoded[paramStartOffset] != 0x01 {
						t.Errorf("Scan not enabled")
					}
				},
			},
			[]hci.ErrorCode{hci.StatusSuccess, hci.StatusSuccess},
			false,
		},
		{
			"Start Scanning passive",
			func(h *Host, ch chan error) {
				_, err := h.StartScanning(true, nil)
				ch <- err
			},
			[]hci.CommandOpCode{hci.CommandLeSetScanParameters, hci.CommandLeSetScanEnable},
			[]checkfn{
				func(encoded []byte, t *testing.T) {
					if len(encoded) != paramStartOffset+7 {
						t.Errorf("Invalid length for set scan parameters")
					}
					if encoded[paramStartOffset] != 0x01 {
						t.Errorf("Expected active scanning")
					}
				},
				func(encoded []byte, t *testing.T) {
					if len(encoded) != paramStartOffset+2 {
						t.Errorf("Invalid length for scan enable")
					}
					if encoded[paramStartOffset] != 0x01 {
						t.Errorf("Scan not enabled")
					}
				},
			},
			[]hci.ErrorCode{hci.StatusSuccess, hci.StatusSuccess},
			false,
		},
		{
			"Start Scanning Fail",
			func(h *Host, ch chan error) {
				_, err := h.StartScanning(false, nil)
				ch <- err
			},
			[]hci.CommandOpCode{hci.CommandLeSetScanParameters},
			[]checkfn{
				func(encoded []byte, t *testing.T) {

				},
			},
			[]hci.ErrorCode{hci.StatusInvalidParams},
			true,
		},
		{
			"Start Scanning 2nd message fail",
			func(h *Host, ch chan error) {
				_, err := h.StartScanning(false, nil)
				ch <- err
			},
			[]hci.CommandOpCode{hci.CommandLeSetScanParameters, hci.CommandLeSetScanEnable},
			[]checkfn{
				func(encoded []byte, t *testing.T) {

				},
				func(encoded []byte, t *testing.T) {

				},
			},
			[]hci.ErrorCode{hci.StatusSuccess, hci.StatusInvalidParams},
			true,
		},
		{
			"Stop scanning",
			func(h *Host, ch chan error) {
				ch <- h.StopScanning()
			},
			[]hci.CommandOpCode{hci.CommandLeSetScanEnable},
			[]checkfn{func(encoded []byte, t *testing.T) {
				if len(encoded) != paramStartOffset+2 {
					t.Errorf("invalid length for parameters")
				}
				if !bytes.Equal(encoded[paramStartOffset:], []byte{0x00, 0x00}) {
					t.Errorf("Unexpected parameters")
				}
			}},
			[]hci.ErrorCode{hci.StatusSuccess},
			false,
		},
		{
			"Stop scanning fail",
			func(h *Host, ch chan error) {
				ch <- h.StopScanning()
			},
			[]hci.CommandOpCode{hci.CommandLeSetScanEnable},
			[]checkfn{nil},
			[]hci.ErrorCode{hci.StatusInvalidParams},
			true,
		},
		{
			"Set Advertising Params",
			func(h *Host, ch chan error) {
				p := hci.DefaultAdvParameters()
				ch <- h.SetAdvertisingParams(p)
			},
			[]hci.CommandOpCode{hci.CommandLeSetAdvParameters},
			[]checkfn{func(encoded []byte, t *testing.T) {
				if len(encoded) != paramStartOffset+15 {
					t.Errorf("Invalid length for advertising parameters")
				}
				expected := make([]byte, 15)
				binary.LittleEndian.PutUint16(expected[0:], 0x0800)
				binary.LittleEndian.PutUint16(expected[2:], 0x0800)
				expected[4] = byte(hci.AdvNonconnInd)
				// own address type, peer address type and peer address
				// should be 0x00 by default
				expected[13] = 0x07 // channel map
				// policy should be 0x00
				if !bytes.Equal(expected, encoded[paramStartOffset:]) {
					t.Errorf("Unexpected parameters for command")
				}
			}},
			[]hci.ErrorCode{hci.StatusSuccess},
			false,
		},
		{
			"Set Advertising Params Fail",
			func(h *Host, ch chan error) {
				p := hci.DefaultAdvParameters()
				ch <- h.SetAdvertisingParams(p)
			},
			[]hci.CommandOpCode{hci.CommandLeSetAdvParameters},
			[]checkfn{nil},
			[]hci.ErrorCode{hci.StatusInvalidParams},
			true,
		},
		{
			"Set Advertising Data",
			func(h *Host, ch chan error) {
				s := []*hci.AdStructure{
					{
						Typ:  hci.AdCompleteLocalName,
						Data: []byte{0x20, 0x00},
					},
					{
						Typ:  hci.AdAppearance,
						Data: []byte{0x01, 0x02},
					},
				}
				ch <- h.SetAdvertisingData(s)
			},
			[]hci.CommandOpCode{hci.CommandLeSetAdvData},
			[]checkfn{func(encoded []byte, t *testing.T) {
				if len(encoded) != paramStartOffset+32 {
					t.Errorf("Expected command parameter length to be 32 bytes, was %d", len(encoded))
				}
				if encoded[paramStartOffset] != 8 {
					t.Errorf("Unexpected advertising data length, was %d", encoded[0])
				}
				expected := make([]byte, 31)
				copy(expected[0:8], []byte{0x03, 0x09, 0x20, 0x00, 0x03, 0x19, 0x01, 0x02})
				if !bytes.Equal(expected, encoded[paramStartOffset+1:]) {
					t.Errorf("invalid parameter contents not expected")
				}
			}},
			[]hci.ErrorCode{hci.StatusSuccess},
			false,
		},
		{
			"Set Scan response",
			func(h *Host, ch chan error) {
				s := []*hci.AdStructure{
					{
						Typ:  hci.AdCompleteLocalName,
						Data: []byte{0x20, 0x00},
					},
					{
						Typ:  hci.AdAppearance,
						Data: []byte{0x01, 0x02},
					},
				}
				ch <- h.SetScanResponse(s)
			},
			[]hci.CommandOpCode{hci.CommandLeSetScanResponse},
			[]checkfn{func(encoded []byte, t *testing.T) {
				if len(encoded) != paramStartOffset+32 {
					t.Errorf("Expected command parameter length to be 32 bytes, was %d", len(encoded))
				}
				if encoded[paramStartOffset] != 8 {
					t.Errorf("Unexpected advertising data length, was %d", encoded[0])
				}
				expected := make([]byte, 31)
				copy(expected[0:8], []byte{0x03, 0x09, 0x20, 0x00, 0x03, 0x19, 0x01, 0x02})
				if !bytes.Equal(expected, encoded[paramStartOffset+1:]) {
					t.Errorf("invalid parameter contents not expected")
				}
			}},
			[]hci.ErrorCode{hci.StatusSuccess},
			false,
		},
		{
			"Set Advertising Data too big",
			func(h *Host, ch chan error) {
				s := []*hci.AdStructure{
					{
						Typ: hci.AdCompleteLocalName,
						Data: []byte{0x20, 0x00, 0x0a, 0x0a, 0x0a, 0x0a, 0x0a, 0x0a, 0x0a, 0x0a,
							0x0a, 0x0a, 0x0a, 0x0a, 0x0a, 0x0a, 0x0a, 0x0a, 0x0a, 0x0a, 0x0a, 0x0a},
					},
					{
						Typ:  hci.AdAppearance,
						Data: []byte{0x01, 0x02, 0x0a, 0x0a, 0x0a, 0x0a, 0x0a, 0x0a, 0x0a, 0x0a, 0x0a, 0x0a},
					},
				}
				ch <- h.SetAdvertisingData(s)
			},
			[]hci.CommandOpCode{},
			[]checkfn{nil},
			[]hci.ErrorCode{},
			true,
		},
		{
			"Set Advertising Data Fail",
			func(h *Host, ch chan error) {
				s := []*hci.AdStructure{
					{
						Typ:  hci.AdCompleteLocalName,
						Data: []byte{0x20, 0x00},
					},
					{
						Typ:  hci.AdAppearance,
						Data: []byte{0x01, 0x02},
					},
				}
				ch <- h.SetAdvertisingData(s)
			},
			[]hci.CommandOpCode{hci.CommandLeSetAdvData},
			[]checkfn{nil},
			[]hci.ErrorCode{hci.StatusInvalidParams},
			true,
		},
		{
			"Set Scan response fail",
			func(h *Host, ch chan error) {
				s := []*hci.AdStructure{
					{
						Typ:  hci.AdCompleteLocalName,
						Data: []byte{0x20, 0x00},
					},
					{
						Typ:  hci.AdAppearance,
						Data: []byte{0x01, 0x02},
					},
				}
				ch <- h.SetScanResponse(s)
			},
			[]hci.CommandOpCode{hci.CommandLeSetScanResponse},
			[]checkfn{nil},
			[]hci.ErrorCode{hci.StatusInvalidParams},
			true,
		},
		{
			"Set Random address",
			func(h *Host, ch chan error) {
				ba, _ := hci.BtAddressFromString("11:22:33:44:55:66")
				ba.Atype = hci.LeRandomAddress

				ch <- h.SetRandomAddress(ba)
			},
			[]hci.CommandOpCode{hci.CommandLeSetRandomAddress},
			[]checkfn{func(encoded []byte, t *testing.T) {
				if len(encoded) != paramStartOffset+6 {
					t.Errorf("Invalid length for parameters")
				}
				if !bytes.Equal(encoded[paramStartOffset:], []byte{0x66, 0x55, 0x44, 0x33, 0x22, 0x11}) {
					t.Errorf("Unexpected payload for command")
				}
			}},
			[]hci.ErrorCode{hci.StatusSuccess},
			false,
		},
		{
			"Set Random address fail",
			func(h *Host, ch chan error) {
				ba, _ := hci.BtAddressFromString("11:22:33:44:55:66")
				ba.Atype = hci.LeRandomAddress

				ch <- h.SetRandomAddress(ba)
			},
			[]hci.CommandOpCode{hci.CommandLeSetRandomAddress},
			[]checkfn{nil},
			[]hci.ErrorCode{hci.StatusCommandDisallowed},
			true,
		},
		{
			"Set Random address, invalid address type",
			func(h *Host, ch chan error) {
				ba, _ := hci.BtAddressFromString("11:22:33:44:55:66")
				ba.Atype = hci.LePublicAddress

				ch <- h.SetRandomAddress(ba)
			},
			[]hci.CommandOpCode{},
			[]checkfn{},
			[]hci.ErrorCode{},
			true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			h := New(nil)
			ch := make(chan error)
			go test.doCommand(h, ch)

			for i, op := range test.expectedOps {
				cmd := <-h.cmd
				if cmd.cmd.OpCode != op {
					t.Errorf("Unexpected opcode for command %d, %v", i, cmd.cmd.OpCode)
				}
				if test.check[i] != nil {
					enc := cmd.cmd.Encode()
					test.check[i](enc, t)
				}
				cmd.complete(mkCommandCompleteEvent(test.statuses[i], op, 1, t))
			}
			err := <-ch
			if test.expectErr && err == nil {
				t.Errorf("Expected error, dit not get")
			} else if !test.expectErr && err != nil {
				t.Errorf("Unexpected error %v", err)
			}
		})
	}

}
