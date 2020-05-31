package main

import (
	"boot/anticipation"
	"device/arm"
	"errors"
	"fmt"
	"lib/trust"
	"lib/upbeat"
	"log"
	"machine"
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

	displayInfo()

	upbeat.UnmaskDAIF()
	for {
		upbeat.MaskDAIF()
		if started {
			//we leave this loop with interrupts OFF
			break
		}
		upbeat.UnmaskDAIF()
		wait()
		upbeat.MaskDAIF()
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
			machine.MiniUART.WriteString("! processing error:" + err.Error() + "\n")
		} else {
			machine.MiniUART.WriteString(".\n")
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
				//this is slightly dodgy, but since interrupts are off, it's ok
				if !started {
					started = true
				} else {
					//reset watchdog timer
					machine.QA7.LocalTimerClearReload.SetReload()
					machine.QA7.LocalTimerClearReload.SetClear() //sad nomenclature
				}
				//pull the character into the internal buffer
				ch := byte(machine.Aux.MUData.Receive())
				switch {
				case ch == 10:
					machine.MiniUART.LoadRx(10)
					moved := machine.MiniUART.CopyRxBuffer(buffer)
					lr.addLineToRing(string(buffer[:moved]))
				case ch < 32 || ch > 127:
					//nothing
				default:
					machine.MiniUART.LoadRx(ch) //put it in the receive buffer
				}
			}
		case machine.QA7.LocalTimerControl.InterruptPendingIsSet():
			//we really should do a lock here but becase we are running
			//on bare metal, we'll get away with this read of a shared
			//variable
			waitCount++
			atLeastOne = true
			if started {
				log.Printf("___________WATCHDOG! __________\n")
				machine.MiniUART.WriteString("! watchdog timeout during transfer\n")
			} else {
				log.Printf("local timer interrupt: #%03d", waitCount)
				machine.MiniUART.WriteString(fmt.Sprintf(". local timer interrupt: #%03d\n", waitCount))
			}
			machine.QA7.LocalTimerClearReload.SetClear() //ugh, nomenclature
			machine.QA7.LocalTimerClearReload.SetReload()
		}
	}
}

var waitCount = 0

func processLine(line string) (bool, error) {
	//really should do a lock here, but on baremetal will be ok
	waitCount = 0

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
		return false, errors.New("unable to execute line " + line)
	}
	if done {
		if !metal.EntryPointIsSet() {
			return false, errors.New("no entry point has been set")
		}
		// normally our CALLER does the confirm, but we are never going to
		// reach there
		machine.MiniUART.WriteString(".\n") //signal the sender everything is ok
		fmt.Printf("@ jumping to address %x\n", metal.EntryPoint())
		upbeat.MaskDAIF() //turn off interrupts while we boot up the kernel
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

func displayInfo() {
	var size, base uint32

	// info := videocore.SetFramebufferRes1920x1200()
	// if info == nil {
	//      rt.Abort("giving up")
	// }
	info := upbeat.SetFramebufferRes1024x768()
	if info == nil {
		fmt.Printf("can't set the framebuffer, aborting\n")
		machine.Abort()
	}

	logger = trust.NewLogger(trust.LogSink())

	id, ok := upbeat.BoardID()
	if ok == false {
		fmt.Printf("can't get board id, aborting\n")
		machine.Abort()
	}
	logger.Infof("board id         : %016x\n", id)

	v, ok := upbeat.FirmwareVersion()
	if ok == false {
		fmt.Printf("can't get firmware version id, aborting\n")
		machine.Abort()
	}
	logger.Infof("firmware version : %08x\n", v)

	rev, ok := upbeat.BoardRevision()
	if ok == false {
		fmt.Printf("can't get board revision id, aborting\n")
		return
	}
	logger.Infof("board revision   : %08x %s\n", rev, upbeat.BoardRevisionDecode(fmt.Sprintf("%x", rev)))

	cr, ok := upbeat.GetClockRate()
	if ok == false {
		fmt.Printf("can't get clock rate, aborting\n")
		machine.Abort()

	}
	logger.Infof("clock rate       : %d hz\n", cr)

	base, size, ok = upbeat.GetARMMemoryAndBase()
	if ok == false {
		fmt.Printf("can't get arm memory, aborting\n")
		machine.Abort()
	}
	logger.Infof("ARM Memory       : 0x%x bytes @ 0x%x\n", size, base)

	base, size, ok = upbeat.GetVCMemoryAndBase()
	if ok == false {
		fmt.Printf("can't get vc memory, aborting\n")
		machine.Abort()
	}
	logger.Infof("VidCore IV Memory: 0x%x bytes @ 0x%x\n", size, base)
}
