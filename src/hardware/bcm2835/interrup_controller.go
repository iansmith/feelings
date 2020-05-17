package bcm2835

import "runtime/volatile"

type IRQRegisterMap struct {
	IRQBasicPending  volatile.Register32 //0x00
	IRQPending1      volatile.Register32 //0x04
	IRQPending2      volatile.Register32 //0x08
	FIQControl       volatile.Register32 //0x0C
	EnableIRQs1      volatile.Register32 //0x10
	EnableIRQs2      volatile.Register32 //0x14
	EnableBasicIRQs  volatile.Register32 //0x18
	DisableIRQs1     volatile.Register32 //0x1C
	DisableIRQs2     volatile.Register32 //0x20
	DisableBasicIRQs volatile.Register32 //0x24
}

// for the interrupt numbers for use with interrupt controller
const AuxInterrupt = 1 << 29

// for the 4 clocks
const systemTimerIRQReserved0 = 1 << 0
const systemTimerIRQReserved2 = 1 << 2

const SystemTimerIRQ1 = 1 << 1
const SystemTimerIRQ3 = 1 << 3

const BasicArmTimerIRQ = 1 << 0
