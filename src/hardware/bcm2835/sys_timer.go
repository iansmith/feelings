package bcm2835

import "github.com/tinygo-org/tinygo/src/runtime/volatile"

type SysTimerRegisterMap struct {
	ControlStatus   volatile.Register32 //0x00
	CounterLower32  volatile.Register32 //0x04
	CounterHigher32 volatile.Register32 //0x08
	reservedGPU0    volatile.Register32 //0x0C
	Compare1        volatile.Register32 //0x10
	reservedGPU2    volatile.Register32 //0x14
	Compare3        volatile.Register32 //0x18
}
