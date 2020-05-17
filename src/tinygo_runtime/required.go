package tinygo_runtime

import (
)

//go:extern _sbss
var _sbss [0]byte

//go:extern _ebss
var _ebss [0]byte

//export runtime.external_putchar
// func putchar(c uint8) {
// 	MiniUART.WriteByte(c)
// }

//export runtime.export_preinit
func preinit() {
	// our bootloader will initialize the bss segment BUT this is used
	// by the bootloader itself
	//
	// Initialize .bss: zero-initialized global variables.
	// ptr := unsafe.Pointer(&_sbss)
	// for ptr != unsafe.Pointer(&_ebss) {
	// 	*(*uint8)(ptr) = 0
	// 	ptr = unsafe.Pointer(uintptr(ptr) + 1)
	// }
}

// var BootArg0, BootArg1, BootArg2, BootArg3, BootArg4 uint64
//
// //go:export main
// func main(a0 uint64, a1 uint64, a2 uint64, a3 uint64, a4 uint64) {
// 	//the args are only usable if you get called from the bootloader
// 	//for a bare metal program, they are garbage
// 	BootArg0 = a0
// 	BootArg1 = a1
// 	BootArg2 = a2
// 	BootArg3 = a3
// 	BootArg4 = a4
//
// 	runtime.Run()
// }
//
// //export runtime.external_postinit
// func postinit() {
// }
//
// //export runtime.external_abort
// func abort() {
// 	MiniUART.WriteString("# executable aborting...\n")
// 	for {
// 		arm.Asm("nop")
// 	}
// }
//
// //export runtime.external_ticks
// func external_ticks() uint64 {
// 	return uint64(0)
// }
//
// //export runtime.external_sleep_ticks
// func external_sleep_ticks(d uint64) {
// 	return
// }

//export:extalloc
//func extalloc(size uintptr) unsafe.Pointer {
//	last16 := size & 0xf
//	if last16 != 0 {
//		size += 16 - (last16)
//	}
//	addr := heapptr
//	heapptr += size
//	for i := uintptr(0); i < uintptr(size); i += 4 {
//		ptr := (*uint32)(unsafe.Pointer(addr + i))
//		*ptr = 0
//	}
//	return addr
//}
