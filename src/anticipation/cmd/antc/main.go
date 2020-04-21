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

var metal *anticipation.MetalByteBuster

const interval = 0x100000

func wait() {
	amount := 1500000000
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
	_ = rt.MiniUART.Configure(rt.UARTConfig{RXInterrupt: true})

	//interrupts start as off
	arm64.InitExceptionVector()
	//all interrupts are "unexpected" until we set this
	arm64.SetExceptionHandlerEl1hInterrupts(interruptReceive)
	arm64.SetExceptionHandlerEl1hSynchronous(interruptReceive)

	currentValue := bcm2835.SysTimer.FreeRunningLower32.Get() //only using 32 bits
	currentValue += 5 * interval
	bcm2835.SysTimer.Compare1.Set(currentValue)

	//do we need to clear it this early?
	//bcm2835.SysTimer.ControlStatus.SetBits(bcm2835.SystemTimerMatch1)

	//tell the interrupt controller what we want and then unmask interrupts
	//bcm2835.InterruptController.EnableIRQs1.SetBits(bcm2835.AuxInterrupt | bcm2835.SystemTimerIRQ1)
	//bcm2835.InterruptController.EnableIRQs1.SetBits(bcm2835.SystemTimerIRQ1)

	//init timer interrupt via controller
	bcm2835.InterruptController.EnableIRQs1.SetBits(bcm2835.SystemTimerIRQ1 | bcm2835.AuxInterrupt)

	arm64.UnmaskDAIF()
	for {
		arm64.MaskDAIF()
		if started {
			//we leave this loop with interrupts OFF
			break
		}
		arm64.UnmaskDAIF()
		wait()
		arm64.MaskDAIF()
	}

	// we ignore errors because we are running on baremetal and there is literally
	// nothing we can do ... the error was in the part that lets us do "print"
	_ = rt.MiniUART.WriteString("#\n")
	_ = rt.MiniUART.WriteString("# Stage 0 kernel running: Anticipation bootloader\n")
	_ = rt.MiniUART.WriteString("# Running from physical address 0x")
	rt.MiniUART.Hex64string((uint64)(uintptr(unsafe.Pointer(&anticipation_addr))))
	_ = rt.MiniUART.WriteString("\n")
	_ = rt.MiniUART.WriteString("#\n")

	//nothing to do but wait for interrupts, we use lr.next() to block
	//until we get a line, and lr.next implies interrupts are off
	for {
		s := lr.next(buffer)
		if len(s) == 0 {
			continue
		}
		done, err := processLine(s)
		if err != nil {
			_ = rt.MiniUART.WriteString("!" + err.Error())
		} else {
			_ = rt.MiniUART.WriteString(".")
		}
		_ = rt.MiniUART.WriteByte('\n')
		if done {
			break
		}
	}
}

//go:noinline
func interruptReceive(_ uint64, _ uint64, _ uint64) {
	_ = rt.MiniUART.WriteString("interruptReceive\n")
	atLeastOne := true
	for atLeastOne {
		atLeastOne = false
		switch {
		//this ignores the possibility that HasBits(6) because docs (!)
		//say that bits 2 and 1 cannot both be set, so we just check bit 2
		case bcm2835.Aux.MiniUARTInterruptIdentify.HasBits(4):
			atLeastOne = true
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
		case bcm2835.InterruptController.IRQPending1.HasBits(bcm2835.SystemTimerIRQ1):
			atLeastOne = true
			if !started {
				started = true
			}
			bcm2835.SysTimer.ControlStatus.SetBits(bcm2835.SystemTimerMatch1)
			_ = rt.MiniUART.WriteString("# Timer interrupt received\n")

		default:
			_ = rt.MiniUART.WriteString("# Expected RX or Timer interrupt, but none found!\n")
		}
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
	converted, lt, _, err := anticipation.DecodeAndCheckStringToBytes(line[:end])
	if err != nil {
		return false, err
	}
	wasError, done := anticipation.ProcessLine(lt, converted, metal)
	if wasError {
		return false, errors.New("unable to excute line " + line)
	}
	if done {
		if !metal.EntryPointIsSet() {
			return false, errors.New("no entry point has been set")
		}
		_ = rt.MiniUART.WriteString(".")
		_ = rt.MiniUART.WriteCR() //signal the sender everything is ok
		_ = rt.MiniUART.WriteString("@ jumping to address ")
		rt.MiniUART.Hex32string(metal.EntryPoint())
		_ = rt.MiniUART.WriteCR()
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
