package main

import (
	arm64 "feelings/src/hardware/arm-cortex-a53"
	"feelings/src/hardware/bcm2835"
	rt "feelings/src/tinygo_runtime"

	"github.com/tinygo-org/tinygo/src/device/arm"
)

const periodInMuSecs = 1 * 1000 /*millis*/ * 1000 /*micros*/

//go:noinline
func main() {
	rt.MiniUART = rt.NewUART()
	rt.MiniUART.Configure(rt.UARTConfig{ /*no interrupt*/ })

	//interrupts start as off
	arm64.InitExceptionVector()
	//all interrupts are "unexpected" until we set this
	arm64.SetExceptionHandlerEl1hInterrupts(serviceRoutine)
	arm64.SetExceptionHandlerEl1hSynchronous(badexc)

	//arm64.QuadA7.LocalInterruptRouting.Set(0)
	//arm64.QuadA7.LocalTimerWriteFlags.SetBits(arm64.QuadA7TimerInterruptFlagClear |
	//	arm64.QuadA7TimerReload)
	//
	////this is set to about 100ms
	//arm64.QuadA7.LocalTimerControlStatus.SetBits(arm64.QuadA7LocalTimerControlInterruptEnable |
	//	arm64.QuadA7LocalTimerControlTimerEnable | 3750000)
	//
	////we use SET here because we want to zero everything else
	//arm64.QuadA7.Core0TimerInterruptControl.Set(arm64.QuadA7NonSecurePhysicalTimer)
	arm64.UnmaskDAIF()

	bcm2835.SysTimer.Compare1.Set(bcm2835.SysTimer.FreeRunningLower32.Get() + 1000000)
	bcm2835.SysTimer.ControlStatus.SetBits(0x2)
	bcm2835.InterruptController.EnableIRQs1.Set(0x2)

	rt.MiniUART.Hex32string(bcm2835.InterruptController.IRQPending1.Get())
	for {
		for i := 0; i < 1000000; i++ {
			arm.Asm("nop")
		}
		rt.MiniUART.Hex32string(bcm2835.InterruptController.IRQPending1.Get())
	}
}

var previousClockValue uint64

//go:noinline
func badexc(t uint64, esr uint64, addr uint64) {
	rt.MiniUART.WriteString("bad\n")
	rt.MiniUART.Hex64string(t)
	rt.MiniUART.Hex64string(esr)
	rt.MiniUART.Hex64string(addr)
	rt.MiniUART.WriteCR()
	for i := 0; i < 1000000000; i++ {
		arm.Asm("nop")
	}
}

//go:noinline
func serviceRoutine(t uint64, esr uint64, addr uint64) {
	rt.MiniUART.WriteString("intr ")

	var high, low uint32
	for {
		low = bcm2835.SysTimer.FreeRunningLower32.Get()
		high = bcm2835.SysTimer.FreeRunningHigher32.Get()
		test := bcm2835.SysTimer.FreeRunningHigher32.Get()
		if test == high {
			break
		}
	}

	x := uint64(high)<<32 | uint64(low)
	if previousClockValue != 0 {
		rt.MiniUART.Hex64string(x)
		rt.MiniUART.Hex64string(previousClockValue)
		rt.MiniUART.Hex64string(x - previousClockValue)
		rt.MiniUART.WriteByte(',')
		rt.MiniUART.Hex32string(uint32(esr))
		rt.MiniUART.Hex32string(bcm2835.SysTimer.ControlStatus.Get())
		rt.MiniUART.WriteCR()
	}
	previousClockValue = x

	arm64.QuadA7.LocalTimerWriteFlags.SetBits(arm64.QuadA7TimerInterruptFlagClear |
		arm64.QuadA7TimerReload)
	bcm2835.SysTimer.ControlStatus.SetBits(0x2)
}
