package semihosting

import (
	"unsafe"

	"github.com/tinygo-org/tinygo/src/device/arm"
)

type SemiHostingOp uint64

const (
	SemiHostOpExit SemiHostingOp = 0x18
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

func semihostCommand(op uint64, param uint64) {
	arm.AsmFull("mov x0,{op}\n"+
		"mov x1,{param}", map[string]interface{}{"op": op, "param": param})
	arm.Asm("hlt 0xF000")
}

func semihostCommandWithParams(op uint64, param uintptr) {
	arm.AsmFull("mov x0,{op}\n"+
		"mov x1,{param}", map[string]interface{}{"op": op, "param": param})
	arm.Asm("hlt 0xF000")
}

func Exit(code uint64) {
	ptr := unsafe.Pointer(&semihosting_param_block)
	*((*uint64)(ptr)) = uint64(SemihostingStopApplicationExit)
	ptr = unsafe.Pointer(uintptr(ptr) + 0x8)
	*((*uint64)(ptr)) = code
	semihostCommandWithParams(uint64(SemiHostOpExit), uintptr(unsafe.Pointer(uintptr(ptr)-0x8)))
}
