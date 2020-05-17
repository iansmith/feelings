package main

import (
	arm64 "hardware/arm-cortex-a53"
	"io"
	"device/arm"
	"machine"
)

var writer io.Writer

var c = &ConsoleImpl{}
const oneSecond = 40000000 //measured by hand

//go:noinline
func main() {
	machine.MiniUART = machine.NewUART()
	machine.MiniUART.Configure(&machine.UARTConfig{ /*no interrupt*/ })
	writer=&machine.MiniUARTWriter{}
	arm64.QuadA7.LocalInterruptRouting.Set(0)

	arm64.QuadA7.LocalTimerControlStatus.Set(arm64.QuadA7LocalTimerControlInterruptEnable | arm64.QuadA7LocalTimerControlTimerEnable| oneSecond)
	arm64.QuadA7.LocalTimerWriteFlags.Set(arm64.QuadA7TimerInterruptFlagClear | arm64.QuadA7TimerReload)
	arm64.QuadA7.Core0TimerInterruptControl.Set( arm64.QuadA7NonSecurePhysicalTimer) //nCNTPNSIRQ_IRQ for SVC mode (EL1)
	arm.Asm("msr daifclr,#2")
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
	print("bad",t,",",esr,",",addr,"\n")
	for i := 0; i < 10000000000; i++ {
		arm.Asm("nop")
	}
}

var previous uint64
//export raw_exception_handler
func rawExceptionHandler(t uint64, esr uint64, addr uint64, el uint64, procId uint64) {
	c.Logf("raw exception handler:exception type %d and esr %x with addr %x and EL=%d, ProcID=%x\n",
		t,esr,addr,el,procId)
	if t!=5 {
		//this is in case we get some OTHER kind of exception
		printoutException(esr)
		c.Logf("DEADLOOP\n")
		for {
			arm.Asm("nop")
		}
		return
	}
	//current:=machine.SystemTime()
	//c.Logf("difference %d and %d\n",current,current/420000)
	//previous=current
	arm64.QuadA7.LocalTimerWriteFlags.Set(arm64.QuadA7TimerInterruptFlagClear| arm64.QuadA7TimerReload)

}


//go:noinline
func printoutException(esr uint64) {
	exceptionClass:=esr>>26
	switch exceptionClass{
	case 0:
		c.Logf("unknown exception")
	case 1:
		c.Logf("trapped WFE or WFI instruction")
	case 2,8,9,10,11,15,16,18,19,20,22,23,26,27,28,29,30,31,35,38,39,41,42,43,45,46,54,55,57,58,59,61,62,63:
		c.Logf("unused code!!")
	case 3:
		c.Logf("trapped MRRC or MCRR access")
	case 4:
		c.Logf("trapped MRRC or MCRR access")
	case 5:
		c.Logf("trapped MRC or MCR access")
	case 6:
		c.Logf("trapped LDC or STC access")
	case 7:
		c.Logf("access to SVE, advanced SIMD or FP functionality")
	case 12:
		c.Logf("trapped to MRRC access")
	case 13:
		c.Logf("branch target exception")
	case 14:
		c.Logf("illegal execution state")
	case 17:
		c.Logf("SVC instruction in AARCH32")
		c.Logf("[",esr&0xffff,"]")
	case 21:
		c.Logf("SVC instruction in AARCH64")
		c.Logf("[",esr&0xffff,"]")
	case 24:
		c.Logf("trapped MRS, MSR or System instruction in AARCH64")
	case 25:
		c.Logf("access to SVE functionality")
	case 32:
		c.Logf("instruction abort from lower exception level")
	case 33:
		c.Logf("instruction abort from same exception level")
	case 34:
		c.Logf("PC alignment fault")
	case 36:
		c.Logf("data abort from lower exception level")
	case 37:
		c.Logf("data abort from same exception level")
	case 40:
		c.Logf("trapped floating point exception from AARCH32")
	case 44:
		c.Logf("trapped floating point exception from AARCH64")
	case 47:
		c.Logf("SError exception")
	case 48:
		c.Logf("Breakpoint from lower exception level")
	case 49:
		c.Logf("Breakpoint from same exception level")
	case 50:
		c.Logf("Software step from lower exception level")
	case 51:
		c.Logf("Software step from same exception level")
	case 52:
		c.Logf("Watchpoint from lower exception level")
	case 53:
		c.Logf("Watchpoint from same exception level")
	case 56:
		c.Logf("BKPT from AARCH32")
	case 60:
		c.Logf("BRK from AARCH64")
	}
	c.Logf("\n")

}