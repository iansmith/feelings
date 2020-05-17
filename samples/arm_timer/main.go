package main

import (
	arm64 "hardware/arm-cortex-a53"
	"io"
	"device/arm"
	"machine"
)

var writer io.Writer

const periodInMuSecs = 1 * 1000 /*millis*/ * 1000 /*micros*/

var c = &ConsoleImpl{}
//go:noinline
func main() {
	machine.MiniUART = machine.NewUART()
	machine.MiniUART.Configure(&machine.UARTConfig{ /*no interrupt*/ })
	writer=&machine.MiniUARTWriter{}
	c.Logf("setting up timers...\n")

	//rate, ok:=machine.GetClockRate()
	//if !ok {
	//	panic("unable to get the GPU clock rate")
	//}

	arm64.QuadA7.LocalInterruptRouting.Set(0)
	//machine.MiniUART.WriteString("A1\n")
	//arm64.QuadA7.Prescaler.Set(0x06AA_AAAB)
	arm64.QuadA7.LocalTimerControlStatus.Set(arm64.QuadA7LocalTimerControlInterruptEnable | arm64.QuadA7LocalTimerControlTimerEnable| 5000000)
	//machine.MiniUART.WriteString("A2")
	//arm64.ARMTimer.Control.SetBits(1<<9) //turn on free runner
	//arm64.ARMTimer.Control.SetBits(0b101 << 5) //turn on the timer and its interrupt
	//desiredDivisor:=(rate/16)/periodInMuSecs
	//preDivide:=(arm64.ARMTimer.Control.Get()>>16) &(0xff)
	//if preDivide==0 {
	//	print("clock speed on qemu should probably be measured...\n")
	//}

	arm64.QuadA7.LocalTimerWriteFlags.Set(arm64.QuadA7TimerInterruptFlagClear | arm64.QuadA7TimerReload)
	//machine.MiniUART.WriteString("A3\n")
	arm64.QuadA7.Core0TimerInterruptControl.Set( arm64.QuadA7NonSecurePhysicalTimer) //nCNTPNSIRQ_IRQ for SVC mode (EL1)
	//machine.MiniUART.WriteString("A4\n")
	//arm64.ARMTimer.Control.SetBits(0b01<<2)
	//arm64.ARMTimer.Control.SetBits((desiredDivisor-1)<<16)
	//prescaleValue:="unknown"
	//switch (arm64.ARMTimer.Control.Get()&0xC)>>2 {
	//case 0:
	//	prescaleValue="1"
	//case 01:
	//	prescaleValue="16"
	//case 10:
	//	prescaleValue="256"
	//}
	//preDivide=(arm64.ARMTimer.Control.Get()>>16) &(0xff)
	//
	//print("clock rate is ", rate," divider ", preDivide," prescale ",prescaleValue, " 23bit?", arm64.QuadA7.LocalTimerControlStatus.Get()&0x2,"\n")
	////desired rate is tied to the choice of the interval
	//if desiredDivisor-1 != preDivide {
	//	panic("can't set the divisor on the arm timer?")
	//}
	//desiredRate:=rate/(desiredDivisor+1)
	//target=uint32( desiredRate & 0xffff_ffff) //32 bit value //

	//arm64.UnmaskDAIF()
	arm.Asm("msr daifclr,#2")
	//machine.MiniUART.WriteString("A5\n")

	//target=uint32( desiredRate & 0xffff_ffff) //32 bit value
	//print("target is ",target,"control is ",uintptr(arm64.ARMTimer.Control.Get()),"\n")
	//arm64.ARMTimer.Load.Set(target)
	//arm64.ARMTimer.Reload.Set(target)
	//arm64.ARMTimer.IRQClearACK.Set(0x1)
	//
	////this is set to about 100ms
	//arm64.QuadA7.LocalTimerControlStatus.SetBits(arm64.QuadA7LocalTimerControlInterruptEnable |
	//	arm64.QuadA7LocalTimerControlTimerEnable | (machine.ArmTimer.Counter.Get() + target))
	//
	////we use SET here because we want to zero everything else
	//arm64.QuadA7.Core0TimerInterruptControl.Set(arm64.QuadA7NonSecurePhysicalTimer)

	for {
		//machine.MiniUART.WriteString("A6\n")
		for i := 0; i < 100000000; i++ {
			arm.Asm("nop")
		}
		//machine.MiniUART.WriteString("A7\n")

		//if machine.InterruptController.EnableIRQs1.Get()!=0 {
		//}
		//machine.MiniUART.WriteString("here1\n")
		c.Logf("---- bottom of wait loop: IRQPending local time control/status=%x and value of counter=%x\n",
			arm64.QuadA7.LocalTimerControlStatus.Get() & 0x8000_0000,
		uint64(arm64.QuadA7.CoreTimerLower32.Get())|uint64(arm64.QuadA7.CoreTimerUpper32.Get()<<32))
		//	//uint64(arm64.QuadA7.CoreTimerLower32.Get())|uint64(arm64.QuadA7.CoreTimerUpper32.Get()<<32),
		//	" reg ",uintptr(arm64.QuadA7.LocalTimerWriteFlags.Get()),"\n")
		//print("armtimer counter ",arm64.QuadA7..Value.Get(), " pending ",arm64.ARMTimer.IRQClearACK.Get(), "\n")
	}
}

var target uint32
var previousClockValue uint64

//go:noinline
func badexc(t uint64, esr uint64, addr uint64) {
	print("bad",t,",",esr,",",addr,"\n")
	for i := 0; i < 10000000000; i++ {
		arm.Asm("nop")
	}
}


//export raw_exception_handler
func rawExceptionHandler(t uint64, esr uint64, addr uint64, el uint64, procId uint64) {
	c.Logf("raw exception handler:exception type %d and esr %x with addr %x and EL=%d, ProcID=%x\n",
		t,esr,addr,el,procId)
	if t!=5 {
		printoutException(esr)
		c.Logf("DEADLOOP\n")
		for {
			arm.Asm("nop")
		}
		return
	}
	lower:=uint64(arm64.QuadA7.CoreTimerLower32.Get())
	upper:=uint64(arm64.QuadA7.CoreTimerUpper32.Get())
	x:=(upper<<32)|lower
	//pending:="local timer not pending!"
	//if arm64.QuadA7.LocalTimerControlStatus.Get() & 0x8000_0000 !=0 {
	//	pending="local timer pending"
	//}
	//src:="unknown source"
	//if arm64.QuadA7.Core0IRQSource.Get() & (1<<11) !=0 {
	//	src="timer source"
	//}
	//c.Logf("type 5 '%s'\n",src)
	//c.Logf("rawExceptionHandler: %s, %s (%x)\n",
	//	pending,src, arm64.QuadA7.Core0IRQSource.Get())
	if previousClockValue!=0 {
		//print("ISR ",t,",",uintptr(esr),",",uintptr(addr),"\n")
		sub:=int64(x)-int64(previousClockValue)
		c.Logf("    time is now %d was %d, delta %d musecs?\n ",x,previousClockValue,sub)
	}
	previousClockValue = x
	arm64.QuadA7.LocalTimerWriteFlags.Set(arm64.QuadA7TimerInterruptFlagClear| arm64.QuadA7TimerReload)
	c.Logf("DONE LATE")

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