package main

import (
	"anticipation"
	"device/arm"
	arm64 "hardware/arm-cortex-a53"
	"machine"

	"errors"
	"fmt"
	"unsafe"
)

var buffer oneLine
var lr *lineRing
var started = false

var metal *anticipation.MetalByteBuster

const interval = 0x4000000

func wait() {
	amount := 1500000000
	if started {
		amount = 1500000
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

	//setup the mini uart so you can see output over serial
	machine.MiniUART = machine.NewUART()
	machine.MiniUART.Configure(&machine.UARTConfig{RXInterrupt: true})
	machine.MiniUART.WriteByte('X')
	//tell the Interrupt controlller what's going on
	machine.IC.Enable1.SetAux()

	machine.QA7.LocalTimerControl.SetInterruptEnable()
	machine.QA7.LocalTimerControl.SetReloadValue(interval)
	machine.QA7.LocalTimerControl.SetTimerEnable()

	//route BOTH local timer and GPU to core 0 on IRQ
	machine.QA7.GPUInterruptRouting.IRQToCore0()
	machine.QA7.LocalInterrupt.SetCore0IRQ()

	//Tell Core0 which interrupts to consume
	machine.QA7.TimerInterruptControl[0].SetPhysicalNonSecureTimerIRQ()
	machine.QA7.IRQSource[0].SetGPU()

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
	fmt.Printf("#\n")
	fmt.Printf("# Stage 0 kernel running: Anticipation bootloader\n")
	fmt.Printf("# Running from physical address 0x%x\n",
		(uint64)(uintptr(unsafe.Pointer(&anticipation_addr))))
	fmt.Printf("#\n")

	//nothing to do but wait for interrupts, we use lr.next() to block
	//until we get a line, and lr.next implies interrupts are off
	for {
		s := lr.next(buffer)
		if len(s) == 0 {
			continue
		}
		done, err := processLine(s)
		if err != nil {
			fmt.Printf("!" + err.Error())
		} else {
			fmt.Printf(".\n")
		}
		if done {
			break
		}
	}
}

//export raw_exception_handler
func rawExceptionHandler(t uint64, esr uint64, addr uint64, el uint64, procId uint64) {
	if t != 5 {
		// this is in case we get some OTHER kind of exception
		fmt.Printf("raw exception handler:exception type %d and "+
			"esr %x with addr %x and EL=%d, ProcID=%x\n",
			t, esr, addr, el, procId)
		fmt.Printf("DEADLOOP\n")
		for {
			arm.Asm("nop")
		}
	}
	interruptReceive()
}

//go:noinline
func interruptReceive() {
	//fmt.Printf("interruptReceive\n")
	atLeastOne := true
	for atLeastOne {
		atLeastOne = false
		switch {
		//note that we don't jump into this case if there is a transmit intr
		case !machine.Aux.MUIIR.InterruptPendingIsSet():
			atLeastOne = true
			for {
				if !machine.Aux.MULSR.DataReadyIsSet() {
					break
				}
				fmt.Printf("got a miniuart interrupt\n")
				//this is slightly dodgy, but since interrupts are off, it's ok
				if !started {
					started = true
				}
				//pull the character into the internal buffer
				ch := byte(machine.Aux.MUData.Receive())
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
		case machine.QA7.LocalTimerControl.InterruptPendingIsSet():
			fmt.Printf("got a timer interrupt\n")
			atLeastOne = true
			machine.QA7.LocalTimerClearReload.SetClear()
			machine.QA7.LocalTimerClearReload.SetReload()
		}
	}
}

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
		fmt.Sprintf((".\n")) //signal the sender everything is ok
		fmt.Printf("@ jumping to address %x\n", metal.EntryPoint())
		arm64.MaskDAIF() //turn off interrupts while we boot up the kernel
		ut := metal.UnixTime()
		ep := metal.EntryPoint()
		jumpToNewKernel(ut, ep)
	}
	//keep going
	return false, nil
}

func jumpToNewKernel(ut uint32, ep uint32) {
	arm.AsmFull("mov x19, {ut}", map[string]interface{}{"ut": ut})
	arm.AsmFull("mov x20, {ep}", map[string]interface{}{"ep": ep})
	arm.Asm("mov x22, #0")
	arm.Asm("mov x23, #0")
	arm.Asm("mov x8, x20")
	arm.Asm("br x8")
}
