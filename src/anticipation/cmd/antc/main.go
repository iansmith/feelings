package main

import (
	"errors"
	"feelings/src/anticipation"

	"github.com/tinygo-org/tinygo/src/device/arm"
	"github.com/tinygo-org/tinygo/src/machine"
)

var buffer oneLine
var lr *lineRing
var started = false

const EOF = ":00000001FF\n"
const signalValue = 0x1234

var entryPoint uint32 = signalValue

var metal *anticipation.MetalByteBuster

func wait() {
	amount := 150000000
	if started {
		amount = 150000
	}
	for i := 0; i < amount; i++ {
		arm.Asm("nop") //wait
	}
}

func main() {
	buffer = make([]uint8, anticipation.FileXFerDataLineSize)
	lr = newLineRing() //probably overkill since never need more than 1 line
	metal = anticipation.NewMetalByteBuster()

	machine.MiniUART = machine.NewUART()
	machine.MiniUART.Configure(machine.UARTConfig{RXInterrupt: true})
	//interrupts start as off
	machine.InitInterrupts()
	//all interrupts are "unexpected" until we set this
	machine.SetExceptionHandlerEl1hInterrupts(miniUARTReceive)

	//tell the interrupt controller what we want and then unmask interrupts
	machine.InterruptController.EnableIRQs1.SetBits(machine.AuxInterrupt)
	machine.UnmaskDAIF()

	for {
		machine.MaskDAIF()
		if started {
			//we leave this loop with interrupts OFF
			break
		}
		machine.UnmaskDAIF()
		machine.MiniUART.WriteString(".")
		machine.MiniUART.WriteByte('\n')
		wait()
		machine.MaskDAIF()

	}

	//nothing to do but wait for interrupts, we use lr.next() to block
	//until we get a line, and lr.next implies interrupts are off
	for {
		s := lr.next(buffer)
		if len(s) == 0 {
			continue
		}
		done, err := processLine(s)
		if err != nil {
			machine.MiniUART.WriteString("!" + err.Error())
		} else {
			machine.MiniUART.WriteString(".")
		}
		machine.MiniUART.WriteByte('\n')
		if done {
			break
		}
	}
}

//go:noinline
func miniUARTReceive(t uint64, esr uint64, addr uint64) {
	//this ignores the possibility that HasBits(6) because docs (!)
	//say that bits 2 and 1 cannot both be set, so we just check bit 2
	if machine.Aux.MiniUARTInterruptIdentify.HasBits(4) {
		for {
			if !machine.Aux.MiniUARTLineStatus.HasBits(machine.ReceiveFIFOReady) {
				break
			}
			//this is slightly dodgy, but since interrupts are off, it's ok
			if !started {
				started = true
			}
			//pull the character into the internal buffer
			ch := machine.MiniUART.ReadByte()
			switch {
			case ch == 10:
				machine.MiniUART.LoadRx(10)
				moved := machine.MiniUART.CopyRxBuffer(buffer)
				lr.addLineToRing(string(buffer[:moved]))
			case ch < 32:
				//nothing
			default:
				machine.MiniUART.LoadRx(ch) //put it in the receive buffer
			}
		}
	} else {
		machine.MiniUART.WriteString("#Expected RX interrupt, but none found!")
		machine.MiniUART.WriteByte('\n')
	}
}

func processLine(line string) (bool, error) {
	//clip off the LF that came from server
	end := len(line)
	if end > 0 && line[end-1] == 10 {
		end--
	}
	//just do what the line says
	converted := anticipation.ConvertBuffer(end, []byte(line))
	if len(converted) == 0 {
		return false, errors.New("no converted results from line:" + line)
	}
	t, ok := anticipation.ExtractLineType(converted[:end])
	if !ok {
		return false, errors.New("unable to extract line type from line:" + line)
	}
	if ok := anticipation.ValidBufferLength(end, converted); ok == false {
		return false, errors.New("expected buffer length to be ok, but wasn't for line: " + line)
	}
	if ok := anticipation.CheckChecksum(end, converted); ok == false {
		return false, errors.New("expected checksum to be ok, but wasn't, line was: " + line)
	}
	wasError, done := anticipation.ProcessLine(t, converted, metal)
	if wasError {
		return false, errors.New("unable to excute line " + line)
	}
	if done {
		if !metal.EntryPointIsSet() {
			return false, errors.New("no entry point has been set")
		}
		machine.MiniUART.WriteString("# jumping to address ")
		machine.MiniUART.Hex32string(metal.EntryPoint())
		machine.MiniUART.WriteCR()
		arm.AsmFull("mov x0, {t}", map[string]interface{}{"t": metal.UnixTime()})
		arm.AsmFull("mov x8, {e}", map[string]interface{}{"e": metal.EntryPoint()})
		arm.Asm("br x8")
	}
	//keep going
	return false, nil
}
