package joy

import (
	"unsafe"

	"machine"

	"lib/trust"
	"lib/upbeat"
)

const quanta = 4000000

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
	if CurrentDomain == nil {
		trust.Fatalf(1, "got a timer tick with no current domain!")
	}
	if CurrentDomain.Counter > 0 {
		CurrentDomain.Counter--
	}
	trust.Debugf("timerTick with current counter: %d (premempt count %d)", CurrentDomain.Counter,
		CurrentDomain.PreemptCount)
	if CurrentDomain.Counter > 0 || CurrentDomain.PreemptCount > 0 {
		return
	}
	CurrentDomain.Counter = 0
	EnableIRQAndFIQ()
	scheduleInternal()
	DisableIRQAndFIQ()
}

func schedule() {
	CurrentDomain.Counter = 0
	scheduleInternal()
}

func scheduleInternal() {
	DisallowPreemption()
	trust.Debugf("schedule internal reached!")
	var p *DomainControlBlock
	next := uint16(0)
	for {
		c := int64(-1)
		for i := uint16(0); i < uint16(len(Domain)); i++ {
			p = Domain[i]
			if p != nil && p.State == DomainStateRunning && p.Counter > c {
				c = p.Counter
				next = i
			}
		}
		if c > 0 {
			break
		}
		for i := uint16(0); i < uint16(len(Domain)); i++ {
			p = Domain[i]
			if p != nil {
				p.Counter = (p.Counter >> 1) + p.Priority
				trust.Debugf("updated counter on %d: %d (from prio %d)", i, p.Counter, p.Priority)
			}
		}
	}
	trust.Debugf("switching to domain: %d", next)
	switchToDomain(Domain[next])
	PermitPreemption()
}

func switchToDomain(next *DomainControlBlock) {
	if CurrentDomain == next {
		return //safety
	}
	prev := CurrentDomain
	CurrentDomain = next
	trust.Debugf("----- cpuSwitchFrom DCB=%p cpuSwitchTo DCB=%p", prev, next)
	cpuSwitchTo(unsafe.Pointer(prev), unsafe.Pointer(next), 0)
}

//go:external
func cpuSwitchTo(unsafe.Pointer, unsafe.Pointer, uintptr)
