package joy

import (
	"unsafe"

	"machine"

	"lib/trust"
	"lib/upbeat"
)

const quanta = 500000

func EnableIRQAndFIQ() {
	upbeat.UnmaskDAIF()
}

func DisableIRQAndFIQ() {
	upbeat.MaskDAIF()
}

// We use local arm timer for the ticks for the preemption counter.
func InitSchedulingTimer() {
	machine.QA7.LocalTimerControl.SetInterruptEnable()
	machine.QA7.LocalTimerControl.SetReloadValue(quanta)
	machine.QA7.LocalTimerControl.SetTimerEnable()

}

// GIC is the generic interrupt controller.  It's actually be used by the
// bootloader, so there isn't much to do.
func InitGIC() {
	// this *should* already have been done by the bootloader

	//route BOTH local timer and GPU to core 0 on IRQ
	machine.QA7.GPUInterruptRouting.IRQToCore0()
	machine.QA7.LocalInterrupt.SetCore0IRQ()

	//Tell Core0 which interrupts to consume
	machine.QA7.TimerInterruptControl[0].SetPhysicalNonSecureTimerIRQ()
	machine.QA7.IRQSource[0].SetGPU()

}

func timerTick() {
	if currentFamily == nil {
		trust.Fatalf(1, "got a timer tick with no current domain!")
	}
	if currentFamily.counter > 0 {
		currentFamily.counter--
	}
	trust.Debugf("timerTick: current domain: %d, counter %d", currentFamily.Id, currentFamily.counter)
	if currentFamily.counter > 0 || currentFamily.preemptCount > 0 {
		return
	}
	currentFamily.counter = 0
	EnableIRQAndFIQ()
	scheduleInternal()
	DisableIRQAndFIQ()
}

func schedule() {
	currentFamily.counter = 0
	scheduleInternal()
}

func scheduleInternal() {
	prohibitPreemption()
	trust.Debugf("schedule internal reached!")
	var p *family
	next := uint16(0)
	for {
		c := int64(-1)
		for i := uint16(0); i < uint16(len(familyImpl)); i++ {
			p = familyImpl[i]
			if p != nil && p.state == fsRunning && p.counter > c {
				c = p.counter
				next = i
			}
		}
		if c > 0 {
			break
		}
		for i := uint16(0); i < uint16(len(familyImpl)); i++ {
			p = familyImpl[i]
			if p != nil {
				p.counter = (p.counter >> 1) + p.priority
				trust.Debugf("updated counter on %d: %d (from prio %d)", i, p.counter, p.priority)
			}
		}
	}
	switchToDomain(familyImpl[next])
	permitPreemption()
}

func switchToDomain(next *family) {
	if currentFamily == next {
		return //safety
	}
	prev := currentFamily
	currentFamily = next
	trust.Debugf("----- cpuSwitchFrom DCB=%p cpuSwitchTo DCB=%p (SP=%x, PC=%x)", prev, next, next.rss.SP, next.rss.PC)
	cpuSwitchTo(unsafe.Pointer(prev), unsafe.Pointer(next), 0)
}

//go:external
func cpuSwitchTo(unsafe.Pointer, unsafe.Pointer, uintptr)
