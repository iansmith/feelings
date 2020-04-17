package tinygo_runtime

import (
	"unsafe"

	"github.com/tinygo-org/tinygo/src/device/arm"
)

//go:extern _sbss
var _sbss [0]byte

//go:extern _ebss
var _ebss [0]byte

type BaremetalRT struct {
}

func (b *BaremetalRT) Putchar(c byte) {
	MiniUART.WriteByte(c)
}

func (b *BaremetalRT) Abort() {
	MiniUART.WriteString("Aborting...")
	for {
		arm.Asm("nop")
	}
}
func (b *BaremetalRT) PostInit() {
}

func (b *BaremetalRT) Ticks() int64 {
	return 0
}

func (b *BaremetalRT) SleepTicks(_ int64) {
	return
}

//export preinit
func preinit() {
	// our bootloader will initialize the bss segment BUT this is used
	// by the bootloader itself
	//
	// Initialize .bss: zero-initialized global variables.
	ptr := unsafe.Pointer(&_sbss)
	for ptr != unsafe.Pointer(&_ebss) {
		*(*uint8)(ptr) = 0
		ptr = unsafe.Pointer(uintptr(ptr) + 1)
	}
}
