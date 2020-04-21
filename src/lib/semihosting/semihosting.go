package semihosting

import (
	"unsafe"
)

type SemiHostingOp uint64

const (
	SemiHostOpExit  SemiHostingOp = 0x18
	SemiHostOpClock SemiHostingOp = 0x10
)

type SemihostingStopCode int

const (
	SemihostingStopBreakpoint          SemihostingStopCode = 0x20020
	SemihostingStopWatchpoint          SemihostingStopCode = 0x20021
	SemihostingStopStepComplete        SemihostingStopCode = 0x20022
	SemihostingStopRuntimeErrorUnknown SemihostingStopCode = 0x20023
	SemihostingStopInternalError       SemihostingStopCode = 0x20024
	SemihostingStopUserInterruption    SemihostingStopCode = 0x20025
	SemihostingStopApplicationExit     SemihostingStopCode = 0x20026
	SemihostingStopStackOverflow       SemihostingStopCode = 0x20027
	SemihostingStopDivisionByZero      SemihostingStopCode = 0x20028
	SemihostingStopOSSpecific          SemihostingStopCode = 0x20029
)

//go:linkname semihosting_param_block semihosting.semihosting_param_block
var semihosting_param_block uint64

//semihosting_call (the second param may be a value or a pointer)
//if it is a pointer, it will point to semihosting_param_block
//go:linkname semihosting_call semihosting.semihosting_call
func semihosting_call(op uint64, param uint64) uint64

//
// Exit and pass this code as result to OS.
//
func Exit(code uint64) {
	//arm.Asm("str     x30, [sp, #-32]!")
	ptr := unsafe.Pointer(&semihosting_param_block)
	*((*uint64)(ptr)) = uint64(SemihostingStopApplicationExit)
	ptr = unsafe.Pointer(uintptr(ptr) + 0x8)
	*((*uint64)(ptr)) = code
	ptr = unsafe.Pointer(uintptr(ptr) - 0x8) //point to start of block
	semihosting_call(uint64(SemiHostOpExit), uint64(uintptr(ptr)))
}

// Returns number of centiseconds since program started
func Clock() uint64 {
	x := semihosting_call(uint64(SemiHostOpClock), 0)
	return x
}
