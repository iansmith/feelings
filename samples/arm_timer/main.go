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

	/*
		// Make sure clock is stopped, illegal to change anything while running
		arm64.ARMTimer.Control.ClearBits(arm64.ARMTimerControlEnable)
		// Get GPU clock (it varies between 200-450Mhz)
		rate, ok := videocore.GetClockRate()
		if !ok {
			panic("can't read the clock rate")
		}
		rt.MiniUART.WriteString("CPU Clock Rate: ")
		rt.MiniUART.Hex32string(rate)
		rt.MiniUART.WriteCR()

		//enable "basic" IRQs
		bcm2835.InterruptController.EnableBasicIRQs.SetBits(bcm2835.BasicArmTimerIRQ)

		// The prescaler divider is set to 256
		rate /= 256
		// Divisor we would need at current clock speed
		div := uint32((periodInMuSecs * rate) / 1000000)
		//set divisor for desired interval
		arm64.ARMTimer.Load.Set(div)
		rt.MiniUART.WriteString("interval set: ")
		rt.MiniUART.Hex32string(div)
		rt.MiniUART.WriteCR()

		//tell the timer what to do
		arm64.ARMTimer.Control.SetBits(arm64.ARMTimerControl23Bit |
			arm64.ARMTimerControlIRQEnable | arm64.ARMTimerControlEnable)
		arm64.ARMTimer.Control.ReplaceBits(arm64.ARMTimerControlPrescale256, arm64.ARMTimerControlPrescaleMask, 0)

		rt.MiniUART.WriteString("irq_setup finished. Control: ")
		rt.MiniUART.Hex32string(arm64.ARMTimer.Control.Get())
		rt.MiniUART.WriteCR()

		arm64.UnmaskDAIF()
	*/

	rt.MiniUART.WriteString("local control/status ")
	rt.MiniUART.Hex32string(arm64.QuadA7.LocalTimerControlStatus.Get())
	rt.MiniUART.WriteCR()

	var freq uint32
	arm.AsmFull("mrs {freq},cntfrq_el0", map[string]interface{}{"freq": freq})
	rt.MiniUART.WriteString("freq ")
	rt.MiniUART.Hex32string(freq)
	rt.MiniUART.WriteCR()

	//compute correct value for timer to be 1 millis
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

	arm.Asm("wfi")

	for {
		arm.Asm("wfi")
		rt.MiniUART.WriteString("x")
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
	//if previousClockValue == 0 {
	//	previousClockValue = float32(semihosting.Clock())
	//	return
	//}
	//curr := float32(semihosting.Clock())
	//diff := previousClockValue - curr
	//previousClockValue = curr
	//print("diff is", diff, "\n")

	arm64.QuadA7.LocalTimerWriteFlags.SetBits(arm64.QuadA7TimerInterruptFlagClear |
		arm64.QuadA7TimerReload)
}
