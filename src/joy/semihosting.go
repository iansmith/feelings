package main

import "github.com/tinygo-org/tinygo/src/device/arm"

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

var twoWordParamBlock [2]uint64

func KExit(code uint64) {
	twoWordParamBlock[0] = uint64(SemiHostOpExit)
	twoWordParamBlock[1] = code
	arm.Asm("HLT #0xF000")
}
