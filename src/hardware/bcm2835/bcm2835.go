// +build rpi3

package bcm2835

import (
	"feelings/src/hardware/rpi"
	"feelings/src/tinygo_runtime"
	"unsafe"
)

var Aux *AuxPeripheralsRegisterMap = (*AuxPeripheralsRegisterMap)(unsafe.Pointer(rpi.MemoryMappedIO + 0x00215000))
var GPIO *GPIORegisterMap = (*GPIORegisterMap)(unsafe.Pointer(rpi.MemoryMappedIO + 0x00200000))
var SysTimer *SysTimerRegisterMap = (*SysTimerRegisterMap)(unsafe.Pointer(rpi.MemoryMappedIO + 0x3000))
var InterruptController *IRQRegisterMap = (*IRQRegisterMap)(unsafe.Pointer(rpi.MemoryMappedIO + 0xB200))

//decls
var MiniUART *tinygo_runtime.UART
