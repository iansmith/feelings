package main

import (
	"feelings/src/anticipation"

	"github.com/tinygo-org/tinygo/src/machine"
)

type oneLine []uint8

const ringMax = 0xf //all 1s at the end

type lineRing struct {
	allLines []oneLine
	lineHead int
	lineTail int
}

func newLineRing() *lineRing {
	result := &lineRing{
		allLines: make([]oneLine, ringMax+1),
	}
	for i := 0; i < len(result.allLines); i++ {
		result.allLines[i] = make([]uint8, anticipation.FileXFerDataLineSize)
	}
	return result
}

// should only be called with interrupts masked!
func (l *lineRing) addLineToRing(s string) {
	l.allLines[l.lineHead] = []byte(s)
	l.lineHead++
	l.lineHead &= ringMax
}

// should only be called with interrupts masked!
func (l *lineRing) empty() bool {
	return l.lineHead == l.lineTail
}

// should only be called with interrupts masked!
func (l *lineRing) next(buffer []uint8) string {
	for {
		if l.empty() {
			machine.UnmaskDAIF()
			wait()
			machine.MaskDAIF()
		} else {
			break // can only get here with interrupts masked
		}
	}
	//byte by byte looking for the LF
	line := l.allLines[l.lineTail]
	l.lineTail++
	l.lineTail &= ringMax
	return string(line)
}
