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

//go:extern semihosting_param_block
var semihosting_param_block uint64

//semihosting_call (the second param may be a value or a pointer)
//if it is a pointer, it will point to semihosting_param_block
//go:linkname semihosting_call semihosting.semihosting_call
func semihosting_call(op uint64, param uint64) uint64

//
////go:noinline
//func semihostCommand(op uint64, param uint64) uint64 {
//	arm.AsmFull("mov x0,{op}\n"+
//		"mov x1,{param}", map[string]interface{}{"op": op, "param": param})
//	arm.Asm("hlt 0xF000")
//	var r uint64
//	arm.AsmFull("mov {r},x0", map[string]interface{}{"r": r})
//	return r
//}
//
////go:noinline
//func semihostCommandWithParams(op uint64, param uintptr) uint64 {
//	arm.AsmFull("mov x0,{op}\n"+
//		"mov x1,{param}", map[string]interface{}{"op": op, "param": param})
//	arm.Asm("hlt 0xF000")
//	var r uint64
//	arm.AsmFull("mov {r},x0", map[string]interface{}{"r": r})
//
//	return r
//}
//
func Exit(code uint64) {
	//arm.Asm("str     x30, [sp, #-32]!")
	ptr := unsafe.Pointer(&semihosting_param_block)
	*((*uint64)(ptr)) = uint64(SemihostingStopApplicationExit)
	ptr = unsafe.Pointer(uintptr(ptr) + 0x8)
	*((*uint64)(ptr)) = code
	semihosting_call(uint64(SemiHostOpExit), uint64(uintptr(ptr)))
	//semihostCommandWithParams(uint64(SemiHostOpExit), uintptr(unsafe.Pointer(uintptr(ptr)-0x8)))
	//arm.Asm("ldr     x30, [sp, #32]")
}

//go:noinline
func Clock() uint64 {
	//arm.Asm("str     x30, [sp, #-32]!")
	//semihostCommand(uint64(SemiHostOpClock), 0)
	x := semihosting_call(uint64(SemiHostOpClock), 0)
	return x
	//var r int64
	//arm.AsmFull("mov {r},x0", map[string]interface{}{"r": r})
	//arm.Asm("ldr     x30, [sp, #32]")
	//return r
}
