package main

import (
	"boot/anticipation"
	"log"
	"strings"
)

///////////////////////////////////////////////////////////////////////
// transmitLooper
///////////////////////////////////////////////////////////////////////
const (
	tsData   transmitState = 0
	tsParams transmitState = 1
	tsEnd    transmitState = 2
)
const (
	kernelParamAddressBlockAddr = 0 // points to BootloaderParamsDef *inside* the kernel
)

// transmitLooper knows how to speak the line oriented protocol with the device
// and handle successful lines and failed lines, doing retransmits when lines
// fail.  the transmit looper uses a sequence of emmiters to do the work of
// figuring out WHAT line to send, the transmit looper is only concerned with
// the responses from the device.
//
// So the layers are: transmitLooper <---  emitter  <--- ioProto
// ioProto does the work of actually sending things through an io interface
//     and receiving things from it.  It only knows the handful of commands that
//     are the ones defined in Intel Hex format.
// emitter figures out what lines (the actual binary content) and addresses to send
//     emitter knows things about the intel hex encoding like where the address
//     part of a data line is, putting on checksums, etc
// transmitLooper works with each line and handles the actual line oriented protocol
//     at the top level.  it is primarily concerned with confirming each line was
//     received ok and if it wasn't, sending it again.
//
type transmitLooper struct {
	state        transmitState
	emitterIndex int
	current      emitter
	emitters     []emitter
	inBuffer     []uint8
	param        [4]uint64
	in           ioProto
	errorCount   int //in a row
	successCount int //overall
}

func newTransmitLooper(all []emitter, oh ioProto) *transmitLooper {
	return &transmitLooper{
		in:           oh,
		state:        tsData,
		emitterIndex: 1,
		current:      all[0],
		emitters:     all,
		inBuffer:     make([]uint8, anticipation.FileXFerDataLineSize),
	}
}

//this returns false when we transition the tsEnd state, even though that
//is a valid state... allows differentiation between moreLines() to moreLines emitter
//and moreLines() to end state.
func (t *transmitLooper) next() bool {
	if t.state == tsEnd { // are we done done?
		log.Fatalf("bad state, transmitLooper should know its done!")
	}
	if t.state == tsParams {
		t.state = tsEnd
		return true
	}
	if t.emitterIndex == len(t.emitters) { //sections done?
		t.state = tsParams
		return false
	}
	t.current = t.emitters[t.emitterIndex]
	t.current.moreLines()
	t.emitterIndex++
	return true
}

func (t *transmitLooper) read() (string, error) {
	l, err := t.current.read(t.inBuffer)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(l), nil
}

const EOFLine = ":00000000000000000001FF"

func (t *transmitLooper) line() (string, error) {
	switch t.state {
	case tsEnd:
		_, err := t.in.EOF()
		if err != nil {
			return "", err
		}
		return EOFLine, nil
	case tsData:
		return t.current.line()
	case tsParams:
		l := anticipation.EncodeExtensionSetParameters(t.param)
		return l, t.in.ExtensionSetParams(l, t.param)
	}
	panic("unexpected state for transmitLooper")
}
