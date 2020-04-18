package tinygo_runtime

import (
	"unsafe"

	"github.com/tinygo-org/tinygo/src/runtime"

	"github.com/tinygo-org/tinygo/src/device/arm"
)

//go:extern _sbss
var _sbss [0]byte

//go:extern _ebss
var _ebss [0]byte

//export runtime.external_putchar
func putchar(c uint8) {
	MiniUART.WriteByte(c)
}

//export runtime.export_preinit
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

var BootArg0, BootArg1, BootArg2, BootArg3, BootArg4 uint64

//export main
func main(a0 uint64, a1 uint64, a2 uint64, a3 uint64, a4 uint64) {
	BootArg0 = a0
	BootArg1 = a1
	BootArg2 = a2
	BootArg3 = a3
	BootArg4 = a4
	runtime.Run()
}

//export runtime.external_postinit
func postinit() {
}

//export runtime.external_abort
func abort() {
	MiniUART.WriteString("# anticipation aborting...\n")
	for {
		arm.Asm("nop")
	}
}

//export runtime.external_ticks
func external_ticks() uint64 {
	return uint64(0)
}

//export runtime.external_sleep_ticks
func external_sleep_ticks(d uint64) {
	return
}
