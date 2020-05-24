package main

import (
	"device/arm"
	"golang/bytes"
	"io"
	"machine"
	"runtime"
)

// Globals
var writer io.Writer
var c = &ConsoleImpl{}
var buf bytes.Buffer

const period = 38400000 * 4 //about 10 secs

func printLine() {
	c.Logf("Line was: %s", buf.String())
	buf.Reset()
}

func bufferCharacter(ch uint8) {
	if ch == 10 {
		//do nothing
	} else {
		if ch == 13 {
			machine.MiniUART.WriteByte('\n')
			machine.MiniUART.WriteByte('\r')
			printLine()
		} else {
			machine.MiniUART.WriteByte(ch)
			buf.WriteByte(ch)
		}
	}

}

//go:noinline
func main() {
	machine.MiniUART = machine.NewUART()
	machine.MiniUART.Configure(&machine.UARTConfig{RXInterrupt: true})
	writer = &machine.MiniUARTWriter{}

	//tell the Interrupt controlller what's going on
	machine.IC.Enable1.SetAux()

	//Configure the local timer
	//On qemu, you have to be sure set the timer enable AFTER the
	//interrupt enable and reload value, as below. If not you get this:
	//Assertion failed: (LOCALTIMER_VALUE(s->local_timer_control) > 0),
	// function bcm2836_control_local_timer_set_next,
	// file /Users/iansmith/rpi3/src/qemu-5.0.0/hw/intc/bcm2836_control.c, line 201.

	machine.QA7.LocalTimerControl.SetInterruptEnable()
	machine.QA7.LocalTimerControl.SetReloadValue(period)
	machine.QA7.LocalTimerControl.SetTimerEnable()

	//route BOTH local timer and GPU to core 0 on IRQ
	machine.QA7.GPUInterruptRouting.IRQToCore0()
	machine.QA7.LocalInterrupt.SetCore0IRQ()

	//Tell Core0 which interrupts to consume
	machine.QA7.TimerInterruptControl[0].SetPhysicalNonSecureTimerIRQ()
	machine.QA7.IRQSource[0].SetGPU()

	//enable interrupts
	arm.Asm("msr daifclr,#2")

	c.Logf("you have about 10 secs to type a line or two (hit return to end line)\n")

	//spinloop, wait
	for {
		for i := 0; i < 100000000; i++ {
			arm.Asm("nop")
		}
	}
}

//var target uint32
//var previousClockValue uint64

//go:noinline
func badexc(t uint64, esr uint64, addr uint64) {
	print("bad", t, ",", esr, ",", addr, "\n")
	for i := 0; i < 10000000000; i++ {
		arm.Asm("nop")
	}
}

var previous uint64

// All exceptions, no matter their origin come here first.  We check
// to see if it's one we expect (type 5) and if it is not, we just
// print out info about it and lock up.
//export raw_exception_handler
func rawExceptionHandler(t uint64, esr uint64, addr uint64, el uint64, procId uint64) {
	if t != 5 {
		//this is in case we get some OTHER kind of exception
		c.Logf("raw exception handler:exception type %d and esr %x with addr %x and EL=%d, ProcID=%x\n",
			t, esr, addr, el, procId)
		c.Logf("DEADLOOP\n")
		for {
			arm.Asm("nop")
		}
	}
	if !machine.Aux.MUIIR.InterruptPendingIsSet() { //clear means interrupt
		value := machine.Aux.MUData.Receive()
		bufferCharacter(uint8(value))
	} else {
		if machine.QA7.LocalTimerControl.InterruptPendingIsSet() {
			c.Logf("\nsorry, you are out of time.")
			if buf.Len() > 0 {
				c.Logf("We had a partial line (%d characters):", buf.Len())
				printLine()
			}
			runtime.Exit()
		} else {
			c.Logf("!! ignoring spurious interrupt !!")
		}
	}
	//clear timer interrupt
	machine.QA7.LocalTimerClearReload.SetClear() //weird nomenclature, but correct
	machine.QA7.LocalTimerClearReload.SetReload()
}
