package main

import (
	"boot/anticipation"
	"log"
	"strings"
	"time"
)

///////////////////////////////////////////////////////////////////////
// transmitter
///////////////////////////////////////////////////////////////////////
const (
	tsData transmitState = 0
	tsTime transmitState = 1
	tsEnd  transmitState = 2
)

type transmitter struct {
	state        transmitState
	emitterIndex int
	current      sectionWriter
	emitters     []sectionWriter
	inBuffer     []uint8
	in           protoReceiver
	errorCount   int //in a row
	successCount int //overall
}

func newTransmitter(all []sectionWriter, oh protoReceiver) *transmitter {

	return &transmitter{
		in:           oh,
		state:        tsData,
		emitterIndex: 1,
		current:      all[0],
		emitters:     all,
		inBuffer:     make([]uint8, anticipation.FileXFerDataLineSize),
	}
}

//this returns false when we transition the tsEnd state, even though that
//is a valid state... allows differentiation between next() to next emitter
//and next() to end state.
func (t *transmitter) next() bool {
	if t.state == tsEnd { // are we done done?
		log.Fatalf("bad state, transmitter should know its done!")
	}
	if t.state == tsTime {
		t.state = tsEnd
		return true
	}
	if t.emitterIndex == len(t.emitters) { //sections done?
		t.state = tsTime
		return false
	}
	t.current = t.emitters[t.emitterIndex]
	t.current.next()
	t.emitterIndex++
	return true
}

func (t *transmitter) read() (string, error) {
	l, err := t.current.read(t.inBuffer)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(l), nil
}

const EOFLine = ":00000001FF"

func (t *transmitter) line() (string, error) {
	switch t.state {
	case tsEnd:
		_, err := t.in.EOF()
		if err != nil {
			return "", err
		}
		return EOFLine, nil
	case tsData:
		return t.current.line()
	case tsTime:
		now := uint32(time.Now().Unix())
		l := anticipation.EncodeExtensionUnixTime(now)
		return l, t.in.ExtensionUnixTime(l, now)
	}
	panic("unexpected state for transmitter")
}
