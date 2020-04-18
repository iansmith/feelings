package main

import (
	"errors"
	"feelings/src/anticipation"
	"feelings/src/hardware/bcm2835"
	"unsafe"

	arm64 "feelings/src/hardware/arm-cortex-a53"
	rt "feelings/src/tinygo_runtime"

	"github.com/tinygo-org/tinygo/src/device/arm"
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

//go:extern anticipation_addr
var anticipation_addr uint64

func main() {
	buffer = make([]uint8, anticipation.FileXFerDataLineSize)
	lr = newLineRing() //probably overkill since never need more than 1 line
	metal = anticipation.NewMetalByteBuster()

	rt.MiniUART = rt.NewUART()
	rt.MiniUART.Configure(rt.UARTConfig{RXInterrupt: true})
	rt.MiniUART.WriteString("OK1\n")
	//interrupts start as off
	arm64.InitInterrupts()
	//all interrupts are "unexpected" until we set this
	arm64.SetExceptionHandlerEl1hInterrupts(miniUARTReceive)

	//tell the interrupt controller what we want and then unmask interrupts
	bcm2835.InterruptController.EnableIRQs1.SetBits(bcm2835.AuxInterrupt)
	arm64.UnmaskDAIF()
	for {
		arm64.MaskDAIF()
		if started {
			//we leave this loop with interrupts OFF
			break
		}
		arm64.UnmaskDAIF()
		rt.MiniUART.WriteString(".")
		rt.MiniUART.WriteByte('\n')
		wait()
		arm64.MaskDAIF()

	}

	rt.MiniUART.WriteString("#\n")
	rt.MiniUART.WriteString("# Stage 0 kernel running: Anticipation bootloader\n")
	rt.MiniUART.WriteString("# Running from physical address 0x")
	rt.MiniUART.Hex64string((uint64)(uintptr(unsafe.Pointer(&anticipation_addr))))
	rt.MiniUART.WriteString("\n")
	rt.MiniUART.WriteString("#\n")
	//nothing to do but wait for interrupts, we use lr.next() to block
	//until we get a line, and lr.next implies interrupts are off
	for {
		s := lr.next(buffer)
		if len(s) == 0 {
			continue
		}
		done, err := processLine(s)
		if err != nil {
			rt.MiniUART.WriteString("!" + err.Error())
		} else {
			rt.MiniUART.WriteString(".")
		}
		rt.MiniUART.WriteByte('\n')
		if done {
			break
		}
	}
}

func miniUARTReceive(t uint64, esr uint64, addr uint64) {
	//this ignores the possibility that HasBits(6) because docs (!)
	//say that bits 2 and 1 cannot both be set, so we just check bit 2
	if bcm2835.Aux.MiniUARTInterruptIdentify.HasBits(4) {
		for {
			if !bcm2835.Aux.MiniUARTLineStatus.HasBits(bcm2835.ReceiveFIFOReady) {
				break
			}
			//this is slightly dodgy, but since interrupts are off, it's ok
			if !started {
				started = true
			}
			//pull the character into the internal buffer
			ch := rt.MiniUART.ReadByte()
			switch {
			case ch == 10:
				rt.MiniUART.LoadRx(10)
				moved := rt.MiniUART.CopyRxBuffer(buffer)
				lr.addLineToRing(string(buffer[:moved]))
			case ch < 32:
				//nothing
			default:
				rt.MiniUART.LoadRx(ch) //put it in the receive buffer
			}
		}
	} else {
		rt.MiniUART.WriteString("#Expected RX interrupt, but none found!")
		rt.MiniUART.WriteByte('\n')
	}
}

//go:extern get_el
func getEl() uint64

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
		rt.MiniUART.WriteString(".")
		rt.MiniUART.WriteCR() //signal the sender everything is ok
		rt.MiniUART.WriteString("@ jumping to address ")
		rt.MiniUART.Hex32string(metal.EntryPoint())
		rt.MiniUART.WriteCR()
		arm64.MaskDAIF() //turn off interrupts while we boot up the kernel
		ut := metal.UnixTime()
		ep := metal.EntryPoint()
		el := getEl()
		jumpToNewKernel(ut, ep, uint32(el))

	}
	//keep going
	return false, nil
}

func jumpToNewKernel(ut uint32, ep uint32, el uint32) {
	arm.AsmFull("mov x19, {ut}", map[string]interface{}{"ut": ut})
	arm.AsmFull("mov x20, {ep}", map[string]interface{}{"ep": ep})
	arm.AsmFull("mov x21, {el}", map[string]interface{}{"el": el})
	arm.Asm("mov x22, #0")
	arm.Asm("mov x23, #0")
	arm.Asm("mov x8, x20")
	arm.Asm("br x8")

}
