// +build rpi3_qemu

package runtime

import (
	"device/arm"
	"unsafe"
)


var asyncScheduler = false
type timeUnit int64

// this is needed because at boot time of the kernel we futz with
// the heapStart, heapEnd, and stackTop values.
// Cannot be inlined because of the optimizer.  It doesn't know we are
// doing and we can't mark heapStart volatile without changing the runtime.

//go:noinline 
func ReInit() {
	tmp:=((*uint64)(unsafe.Pointer(&heapStartSymbol)))
	heapStart = uintptr(*tmp)
	tmp=((*uint64)(unsafe.Pointer(&heapEndSymbol)))
	heapEnd = uintptr(*tmp)
	tmp=((*uint64)(unsafe.Pointer(&stackTopSymbol)))
	stackTop = uintptr(*tmp)

	initHeap()

//	heapptr = heapStart
	
}

//go:export sleepticks sleepticks
func sleepTicks(n timeUnit) {
	start := Semihostingv2Call(uint64(Semihostingv2OpClock), 0)
	centis := uint64(n)
	current := start
	for current-start < centis { //busy wait
		for i := 0; i < 20; i++ {
			arm.Asm("nop")
		}
		current = Semihostingv2Call(uint64(Semihostingv2OpClock), 0)
	}
}

func ticks() timeUnit {
	current := Semihostingv2Call(uint64(Semihostingv2OpClock), 0)
	return timeUnit(current)

}

func ticksToNanoseconds(t timeUnit) int64 {
	//first one here is 10 because t is in CENTIseconds
	return int64((10*1000*1000)*t)
}
func nanosecondsToTicks(t int64) timeUnit {
	//first one here is 10 because return is in CENTIseconds
	return timeUnit(t / (10 * 1000 * 1000))
}

//go:export main
func main() {
	run()
	Exit()
}

func putchar(c byte) {
	Semihostingv2Putchar(c)
}

//go:extern _sbss
var _sbss [0]byte

//go:extern _ebss
var _ebss [0]byte

func abort() {
	Semihostingv2Call(uint64(Semihostingv2OpExit), uintptr(Semihostingv2StopRuntimeErrorUnknown))
}

func Exit() {
	Semihostingv2Call(uint64(Semihostingv2OpExit), uintptr(Semihostingv2StopApplicationExit))
}

func postinit() {
	// Initialize .bss: zero-initialized global variables.
	ptr := unsafe.Pointer(&_sbss)
	for ptr != unsafe.Pointer(&_ebss) {
		*(*uint32)(ptr) = 0
		ptr = unsafe.Pointer(uintptr(ptr) + 4)
	}
}

type SemiHostingOp uint64

const (
	Semihostingv2OpExit   SemiHostingOp = 0x18
	Semihostingv2OpClock  SemiHostingOp = 0x10
	Semihostingv2OpWriteC SemiHostingOp = 0x03
)

type SemihostingStopCode int

const (
	Semihostingv2StopBreakpoint          SemihostingStopCode = 0x20020
	Semihostingv2StopWatchpoint          SemihostingStopCode = 0x20021
	Semihostingv2StopStepComplete        SemihostingStopCode = 0x20022
	Semihostingv2StopRuntimeErrorUnknown SemihostingStopCode = 0x20023
	Semihostingv2StopInternalError       SemihostingStopCode = 0x20024
	Semihostingv2StopUserInterruption    SemihostingStopCode = 0x20025
	Semihostingv2StopApplicationExit     SemihostingStopCode = 0x20026
	Semihostingv2StopStackOverflow       SemihostingStopCode = 0x20027
	Semihostingv2StopDivisionByZero      SemihostingStopCode = 0x20028
	Semihostingv2StopOSSpecific          SemihostingStopCode = 0x20029
)

//go:linkname semihosting_param_block semihosting.semiHostingParamBlock
var semihostingParamBlock uint64

//semihosting_call (the second param may be a value or a pointer)
//if it is a pointer, it will point to semihosting_param_block
//export semihosting_call
func Semihostingv2Call(op uint64, param uintptr) uint64

//go:export semihosting_putchar
func Semihostingv2Putchar(byte) uint64

//Semihostingv2ClockMicros is supposed to return micros since the program
//started. However, the underlying call is supposed to return centiseconds
//and it appears to be wildly inaccurate.
func Semihostingv2ClockMicros() uint64 {
	centis := Semihostingv2Call(uint64(Semihostingv2OpClock), 0)
	return centis * 100000
}
