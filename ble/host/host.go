package host

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"

	"gitlab.com/jtaimisto/bluewalker/filter"
	"gitlab.com/jtaimisto/bluewalker/hci"
	"gitlab.com/jtaimisto/bluewalker/logging"
)

// how long to wait for command to execute
var cmdExecutionTimeout = time.Duration(30 * time.Second)

// Error for command execution timeout
var errExecutionTimeout = fmt.Errorf("command execution timed out")

// exec is used when HCI commands need to be sent to controller
type exec struct {
	// command to execute
	cmd *hci.CommandPacket
	// this function is called when execution has completed
	complete func(*hci.CommandCompleteEvent)
	// this is called if command can not be sent
	fail func(error)
}

//ScanReport contains information about a device found on scanning
//See Bluetooth 5.0, vol 2, part E, ch 7.7.65.2
type ScanReport struct {
	Type    hci.AdvType
	Address hci.BtAddress
	Rssi    int8
	Data    []*hci.AdStructure
}

// Host implements the host side of Bluetooth Host - Controller interface
type Host struct {
	// Transport we are using for Host - Controller communication
	tr hci.Transport
	// WaitGroup to signal when host event receiver has stopped
	wg sync.WaitGroup
	// Mutex synchronizing access to host internals
	mux sync.Mutex
	// Event channel
	// Event receiver pushes received events to this channel
	// Event handler will read events from this channel
	evt chan []byte
	// Commands for executor
	// Command executor will read commands to execute from this channel
	cmd chan *exec
	// CommandComplete events to executor
	// Command executor will read this channel when it is waiting command to complete
	cc chan *hci.CommandCompleteEvent
	// Channel used to inform about received scanning data
	// StartScanning() will return this channel and user will receive ScanReports
	// through it.
	ad chan *ScanReport
	// Filters for incoming advertising reports
	filters filter.AdFilter
	// flag indicating that host is closing.
	// access needs to be protected using mux as event receiving goroutine
	// is using this to indicate it should stop.
	closing bool
}

// New returns new host which uses given transport for communicating
// with controller
func New(tr hci.Transport) *Host {

	host := new(Host)
	host.tr = tr
	host.filters = nil
	host.evt = make(chan []byte, 2)
	host.cmd = make(chan *exec)
	host.cc = make(chan *hci.CommandCompleteEvent)
	host.ad = make(chan *ScanReport, 5)
	host.closing = false

	return host
}

// isClosing returns true if closing flag is set.
// This is safe way to check the status of closing flag
func (h *Host) isClosing() bool {
	var ret bool
	h.mux.Lock()
	ret = h.closing
	h.mux.Unlock()
	return ret
}

// eventReceiver is run on its own goroutine and it uses the transport to
// receive events from Controller. No other goroutine should read from
// transport. The received events are written to 'evt' channel in host
// this method will return when isClosing() returns true
func (h *Host) eventReceiver() {

	defer h.wg.Done()
	for !h.isClosing() {
		buf, err := h.tr.Read()
		if err != nil {
			var again hci.ErrReadAgain
			if !errors.As(err, &again) {
				logging.Warning.Printf("Error while reading: %v", err)
			}
			continue
		}
		if len(buf) == 0 {
			continue
		}
		if buf[0] != hci.HciEventPacket {
			logging.Debug.Printf("Received unexpected packet from controller")
			continue
		}
		logging.Debug.Printf("Received %d bytes of event", len(buf)-1)
		h.evt <- buf[1:]
	}
	logging.Trace.Printf("EventReceiver closing")
}

// eventHandler is run on its own goroutine and it reads the events
// from 'evt' channel. Event handler is responsible for routing the events
// to correct channel or calling proper handlers for the events.
// This method returns when 'evt' channel is closed
func (h *Host) eventHandler() {

	for buf := range h.evt {
		evt, err := hci.DecodeEvent(buf)
		if err != nil {
			logging.Warning.Printf("Received invalid event: %s", err.Error())
			continue
		}
		logging.Debug.Printf("Received %s event", evt.Code.String())
		switch evt.Code {
		case hci.EventCodeCommandComplete:
			cc, err := hci.DecodeCommandComplete(evt)
			if err != nil {
				logging.Warning.Printf("Received invalid Command Complete event: %s", err.Error())
				continue
			}
			h.cc <- cc
		case hci.EventCodeLeMeta:
			meta, err := hci.DecodeLeMeta(evt)
			if err != nil {
				logging.Warning.Printf("Received invalid LE Meta event: %s", err.Error())
				continue
			}
			if meta.GetSubeventCode() == hci.SubeventAdvertisingReport {
				if err := handleAdvertisingReport(h.ad, h.filters, meta.GetParameters()); err != nil {
					logging.Warning.Printf("Error while parsing Advertising report: %s", err.Error())
				}
			}
		default:
			logging.Debug.Printf("Received unexpected event %s", evt.Code.String())
		}
	}
	logging.Trace.Printf("Event handler stopping")
}

// executor is run on its own goroutine and it is responsible for for
// executing HCI commands. The commands to execute are read from 'cmd'
// channel and written to controller. Then Command Complete event is waited
// and the status of command execution is communicated back to one requesting
// the command to be sent.
// This method returns once the 'cmd' channel is closed.
func (h *Host) executor() {

	// number of commands we can execute
	numCommands := 1
	execTimer := time.NewTimer(cmdExecutionTimeout)

	for e := range h.cmd {
		logging.Debug.Printf("Executing command %s", e.cmd.OpCode.String())
		if numCommands == 0 {
			e.fail(fmt.Errorf("flow control error"))
			continue
		}
		if err := h.tr.Write(e.cmd.Encode()); err != nil {
			e.fail(fmt.Errorf("can not write: %s", err.Error()))
			continue
		}

		if !execTimer.Stop() {
			<-execTimer.C
		}
		execTimer.Reset(cmdExecutionTimeout)

		numCommands--
		completed := false
		// we need to wait until the command has completed before starting
		// with new command.
		for !completed || numCommands == 0 {
			select {
			case cc := <-h.cc:
				if cc == nil {
					// channel is closed and we should thus be breaking out
					break
				}
				numCommands = int(cc.GetNumHciCommandPackets())
				logging.Debug.Printf("Number of HCI packets increased to %d", numCommands)
				if !completed && cc.GetCommandOpcode() == e.cmd.OpCode {
					completed = true
					e.complete(cc)
				} else if !completed {
					logging.Warning.Printf("Received unexepcted cc for %s ", cc.GetCommandOpcode().String())
				}
			case <-execTimer.C:
				if !completed {
					logging.Warning.Printf("Command %s execution timed out", e.cmd.OpCode.String())
					completed = true
					e.fail(errExecutionTimeout)
					// The command complete has likely been lost somewhere
					// increase the numCommands to allow sending new command
					// otherwise we'll be stuck here looping and never
					numCommands = 1
				}
			}
		}
	}
}

// CommandExecutionError is returned when HCI command could not be
// executed.
type CommandExecutionError struct {
	status hci.ErrorCode     // Status code for the failure
	op     hci.CommandOpCode // command that failed to execute
}

func (e *CommandExecutionError) Error() string {
	return fmt.Sprintf("Command %s execution failed: %s ", e.op.String(), e.status.String())
}

// ErrorCode returns the error code indicating reason for command
// execution failure
func (e *CommandExecutionError) ErrorCode() hci.ErrorCode {
	return e.status
}

// executeStatusCommand executes single HCI command which expects to have
// 'status' parameter in the following CommandComplete event. This status
// is checked and error is returned command execution failed.
func (h *Host) executeStatusCommand(cmd *hci.CommandPacket) error {

	var wg sync.WaitGroup

	var err error

	e := new(exec)
	e.cmd = cmd
	e.complete = func(cc *hci.CommandCompleteEvent) {
		if cc.HasReturnParameters() {
			if cc.GetStatusParameter() != hci.StatusSuccess {
				err = &CommandExecutionError{status: cc.GetStatusParameter(), op: cmd.OpCode}
			}
		} else {
			err = fmt.Errorf("received unexpected Command Complete with no status")
		}
		wg.Done()
	}
	e.fail = func(er error) {
		err = fmt.Errorf("command execution failed: %s", er.Error())
		wg.Done()
	}
	wg.Add(1)
	h.cmd <- e
	wg.Wait()
	return err
}

// initializeController sends the necessary commands to initialize
// communication with Controller. The controller is reset first.
func (h *Host) initializeController() error {

	commands := make([]*hci.CommandPacket, 4)

	// Reset
	commands[0] = &hci.CommandPacket{OpCode: hci.CommandReset}

	// LE Host Supported command
	commands[1] = &hci.CommandPacket{OpCode: hci.CommandWriteLeHostSupported}
	params := make([]byte, 2)
	params[0] = 0x01 // LE Supported Host enabeled
	params[1] = 0x00 // Simultaneous LE Host parameter
	commands[1].Parameters(params)

	// Set Event mask
	commands[2] = &hci.CommandPacket{OpCode: hci.CommandSetEventMask}
	params = make([]byte, 8)
	// Default mask, all events, we might want to optimize this
	binary.LittleEndian.PutUint64(params, 0x3fffffffffffffff)
	commands[2].Parameters(params)

	// Set LE event mask
	commands[3] = &hci.CommandPacket{OpCode: hci.CommandLeSetEventMask}
	params = make([]byte, 8)
	binary.LittleEndian.PutUint64(params, 0x000000000000001f)
	commands[3].Parameters(params)

	for _, cmd := range commands {
		if err := h.executeStatusCommand(cmd); err != nil {
			return err
		}
	}
	return nil
}

// Init initializes the host
func (h *Host) Init() error {

	logging.Debug.Printf("Initializing Host")

	h.wg.Add(1)
	// Start the event and command handlng goroutines
	go h.eventReceiver()
	go h.eventHandler()
	go h.executor()

	logging.Debug.Printf("Resetting...")

	if err := h.initializeController(); err != nil {
		// XXX: Deinitialize
		var execErr *CommandExecutionError
		if errors.As(err, &execErr) {
			return fmt.Errorf("unable to initialize controller: %w", execErr)
		}
		return fmt.Errorf("unable to initialize controller: %s", err.Error())
	}

	return nil
}

//StartScanning will start scanning for Bluetooth LE Advertisements
//Active defines if active or passive scanning should be done
//The returned channel can be used to receive all scan reports matching _any_
//of the filters on list. The returned cannel should _not_ be closed.
func (h *Host) StartScanning(active bool, filters []filter.AdFilter) (chan *ScanReport, error) {

	if len(filters) > 0 {
		h.filters = filter.All(filters)
	}

	cmd := hci.CommandPacket{OpCode: hci.CommandLeSetScanParameters}
	// See Bluetooth v5.0, vol 2, part E, ch 7.8.10
	parameters := make([]byte, 7)
	if active {
		// active scanning
		parameters[0] = 0x01
	}
	// Scan interval
	binary.LittleEndian.PutUint16(parameters[1:], 0x0010)
	// Scan window
	binary.LittleEndian.PutUint16(parameters[3:], 0x0010)
	// Own address type, public
	parameters[5] = 0x00
	// Filter policy
	parameters[6] = 0x00
	cmd.Parameters(parameters)

	logging.Debug.Printf("Setting scan parameters")
	if err := h.executeStatusCommand(&cmd); err != nil {
		return nil, fmt.Errorf("unable to set Scan Parameters: %s", err.Error())
	}

	cmd = hci.CommandPacket{OpCode: hci.CommandLeSetScanEnable}
	// See Bluetooth v5.0, vol 2, part E, ch 7.8.11
	parameters = make([]byte, 2)
	// Scan enable
	parameters[0] = 0x01
	// Filter duplicates
	parameters[1] = 0x00
	cmd.Parameters(parameters)

	logging.Debug.Printf("Starting scan")
	if err := h.executeStatusCommand(&cmd); err != nil {
		return nil, fmt.Errorf("unable to start scanning: %s", err.Error())
	}
	return h.ad, nil
}

//StopScanning stops scanning for advertising LE devices
func (h *Host) StopScanning() error {

	cmd := hci.CommandPacket{OpCode: hci.CommandLeSetScanEnable}
	parameters := make([]byte, 2)
	// Scan enable
	parameters[0] = 0x00
	// filter duplicates
	parameters[1] = 0x00
	cmd.Parameters(parameters)
	if err := h.executeStatusCommand(&cmd); err != nil {
		return fmt.Errorf("unable to stop scanning: %s", err.Error())
	}
	return nil
}

//SetAdvertisingParams set advertising params to the controller.
// hci.DefaultAdvParameters() can be used to get default set of parameters.
func (h *Host) SetAdvertisingParams(advParams hci.AdvertisingParameters) error {

	cmd := hci.CommandPacket{OpCode: hci.CommandLeSetAdvParameters}
	params := make([]byte, 15)

	// Min advertising interval
	binary.LittleEndian.PutUint16(params[0:2], advParams.IntervalMin)
	// Max advertising interval
	binary.LittleEndian.PutUint16(params[2:4], advParams.IntervalMax)
	// advertising type
	params[4] = byte(advParams.Type)
	// Own Address Type
	params[5] = byte(advParams.OwnAddrType)
	// Peer address type
	if advParams.PeerAddress.Atype == hci.LePublicAddress {
		params[6] = 0x00
	} else {
		params[6] = 0x01
	}
	// peer address
	advParams.PeerAddress.Put(params[7:])
	// Channel Map
	params[13] = byte(advParams.ChannelMap)
	// Filter policy
	params[14] = byte(advParams.FilterPolicy)

	cmd.Parameters(params)
	if err := h.executeStatusCommand(&cmd); err != nil {
		return fmt.Errorf("unable to set advertising parameters: %s", err.Error())
	}
	return nil
}

func putAdvData(buf []byte, datas []*hci.AdStructure) (int, error) {
	offset := 0
	for i, ad := range datas {
		n, err := ad.EncodeTo(buf[offset:])
		if err != nil {
			return 0, fmt.Errorf("advertising Data %d could not be written (%s)", i, err.Error())
		}
		offset += n
	}
	return offset, nil
}

func (h *Host) setAdvData(data []*hci.AdStructure, scanResp bool) error {
	var opcode hci.CommandOpCode
	if scanResp {
		opcode = hci.CommandLeSetScanResponse
	} else {
		opcode = hci.CommandLeSetAdvData
	}
	cmd := hci.CommandPacket{OpCode: opcode}
	params := make([]byte, 32)
	len, err := putAdvData(params[1:], data)
	if err != nil {
		return err
	}
	params[0] = byte(len)
	cmd.Parameters(params)

	if err := h.executeStatusCommand(&cmd); err != nil {
		return fmt.Errorf("unable set advertising data: %s", err.Error())
	}

	return nil
}

//SetAdvertisingData sets the advertising data that will be sent
//when advertising is enabled
func (h *Host) SetAdvertisingData(data []*hci.AdStructure) error {
	return h.setAdvData(data, false)
}

//SetScanResponse sets the scan response data which will be sent
//when advertising and the mode allows it.
func (h *Host) SetScanResponse(data []*hci.AdStructure) error {
	return h.setAdvData(data, true)
}

//StartAdvertising directs the controller to start sending advertisments
func (h *Host) StartAdvertising() error {

	cmd := hci.CommandPacket{OpCode: hci.CommandLeSetAdvEnable}
	params := make([]byte, 1)
	// Enabled
	params[0] = 0x01
	cmd.Parameters(params)
	if err := h.executeStatusCommand(&cmd); err != nil {
		return fmt.Errorf("unable to start advertising: %s", err.Error())
	}
	return nil
}

//StopAdvertising directs the controller to stop sending advertisments
func (h *Host) StopAdvertising() error {

	cmd := hci.CommandPacket{OpCode: hci.CommandLeSetAdvEnable}
	params := make([]byte, 1)
	// Enabled
	params[0] = 0x00
	cmd.Parameters(params)
	if err := h.executeStatusCommand(&cmd); err != nil {
		return fmt.Errorf("unable to start advertising: %s", err.Error())
	}
	return nil
}

//SetRandomAddress sets LE Random Device Address to the Controller
func (h *Host) SetRandomAddress(addr hci.BtAddress) error {
	if addr.Atype != hci.LeRandomAddress {
		return fmt.Errorf("invalid address type %s, expected %s",
			addr.Atype.String(), hci.LeRandomAddress.String())
	}
	cmd := hci.CommandPacket{OpCode: hci.CommandLeSetRandomAddress}
	params := make([]byte, 6)
	addr.Put(params)
	cmd.Parameters(params)
	if err := h.executeStatusCommand(&cmd); err != nil {
		return fmt.Errorf("unable to set random address: %s", err.Error())
	}
	return nil
}

// Deinit will deinitialize Host
func (h *Host) Deinit() {
	logging.Debug.Printf("Deinitializing host")
	cmd := hci.CommandPacket{OpCode: hci.CommandReset}
	// not checking the return value since there is not much we can do on error
	h.executeStatusCommand(&cmd)
	h.mux.Lock()
	h.closing = true
	h.mux.Unlock()

	h.tr.Close()
	h.wg.Wait()
	// Now the eventReceiver has stopped. Rest should stop when we close
	// the channels
	close(h.evt)
	close(h.cc)
	close(h.ad)
	logging.Debug.Printf("Deinitialization done")
}
