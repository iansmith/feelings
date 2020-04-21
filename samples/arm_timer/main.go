package main

import (
	arm64 "feelings/src/hardware/arm-cortex-a53"
	"feelings/src/lib/semihosting"
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

	var freq uint32
	arm.AsmFull("mrs {freq},cntfrq_el0", map[string]interface{}{"freq": freq})
	desired := uint32(100)
	prescalar := (1 << 31) * freq / desired
	rt.MiniUART.WriteString("prescaler ")
	rt.MiniUART.Hex32string(prescalar)
	rt.MiniUART.WriteCR()

	arm64.QuadA7.Prescaler.Set(prescalar)
	arm64.QuadA7.LocalInterruptRouting.Set(0)
	arm64.QuadA7.LocalTimerWriteFlags.SetBits(arm64.QuadA7TimerInterruptFlagClear |
		arm64.QuadA7TimerReload)

	arm64.QuadA7.LocalTimerControlStatus.SetBits(arm64.QuadA7LocalTimerControlInterruptEnable |
		arm64.QuadA7LocalTimerControlTimerEnable | 5000000)

	//we use SET here because we want to zero everything else
	arm64.QuadA7.Core0TimerInterruptControl.Set(arm64.QuadA7NonSecurePhysicalTimer)
	arm64.UnmaskDAIF()

	for {
		arm.Asm("wfi")
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
}

//go:noinline
func serviceRoutine(t uint64, esr uint64, addr uint64) {
	rt.MiniUART.WriteString("intr \n")
	x := semihosting.Clock()
	if previousClockValue != 0 {
		rt.MiniUART.Hex64string(uint64(x - previousClockValue))
	}
	previousClockValue = x

	arm64.QuadA7.LocalTimerWriteFlags.SetBits(arm64.QuadA7TimerInterruptFlagClear |
		arm64.QuadA7TimerReload)
}
