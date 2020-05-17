package arm_cortex_a53

import (
	"hardware/rpi"
	"unsafe"

	"runtime/volatile"
)

type ARMTimerRegisterMap struct {
	Load        volatile.Register32 //0x00
	Value       volatile.Register32 //0x04
	Control     volatile.Register32 //0x08
	IRQClearACK volatile.Register32 //0x0C
	RawIRQ      volatile.Register32 //0x10
	MaskedIRQ   volatile.Register32 //0x14
	Reload      volatile.Register32 //0x18
}

var ARMTimer *ARMTimerRegisterMap = (*ARMTimerRegisterMap)(unsafe.Pointer(rpi.MemoryMappedIO + 0xB000))

const ARMTimerControlEnable = 1 << 7
const ARMTimerControl23Bit = 1 << 1
const ARMTimerControlPrescale256 = 0b10 << 2
const ARMTimerControlPrescale16 = 0b01 << 2
const ARMTimerControlPrescaleMask = 0xC
const ARMTimerControlIRQEnable = 1 << 5

// QA7_rev3.4.pdf
type QuadA7RegisterMap struct {
	Control                        volatile.Register32 // 0x00
	unused                         volatile.Register32 //0x04
	Prescaler                      volatile.Register32 //0x08
	GPUInterruptsRouting           volatile.Register32 //0xC
	PerfMonInterruptsSet           volatile.Register32 //0x10
	PerfMonInterruptsClear         volatile.Register32 //0x14
	unused0                        uint32              //0x18
	CoreTimerLower32               volatile.Register32 //0x1C
	CoreTimerUpper32               volatile.Register32 //0x20
	LocalInterruptRouting          volatile.Register32 //0x24, interrupts 1-7?
	unknown0                       uint32              //0x28
	AxiOutstandingCounters         volatile.Register32 //0x2C
	AxiOutstandingInterrupts       volatile.Register32 //0x30
	LocalTimerControlStatus        volatile.Register32 //0x34
	LocalTimerWriteFlags           volatile.Register32 //0x38
	unused1                        uint32              //0x3C
	Core0TimerInterruptControl     volatile.Register32 //0x40
	Core1TimerInterruptControl     volatile.Register32 //0x44
	Core2TimerInterruptControl     volatile.Register32 //0x48
	Core3TimerInterruptControl     volatile.Register32 //0x4C
	Core0MailboxesInterruptControl volatile.Register32 //0x50
	Core1MailboxesInterruptControl volatile.Register32 //0x54
	Core2MailboxesInterruptControl volatile.Register32 //0x58
	Core3MailboxesInterruptControl volatile.Register32 //0x5C
	Core0IRQSource                 volatile.Register32 //0x60
	Core1IRQSource                 volatile.Register32 //0x64
	Core2IRQSource                 volatile.Register32 //0x68
	Core3IRQSource                 volatile.Register32 //0x6C
	Core0FIQSource                 volatile.Register32 //0x70
	Core1FIQSource                 volatile.Register32 //0x74
	Core2FIQSource                 volatile.Register32 //0x78
	Core3FIQSource                 volatile.Register32 //0x7C
}

var QuadA7 *QuadA7RegisterMap = (*QuadA7RegisterMap)(unsafe.Pointer(uintptr(0x40000000)))

const QuadA7LocalTimerControlInterruptEnable = 1 << 29
const QuadA7LocalTimerControlTimerEnable = 1 << 28

const QuadA7TimerInterruptFlagClear = 1 << 31
const QuadA7TimerReload = 1 << 30

const QuadA7NonSecurePhysicalTimer = 1 << 1
const QuadA7GPUFast = 1 << 8
