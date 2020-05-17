package bcm2835

import "runtime/volatile"

type GPIORegisterMap struct {
	FuncSelect               [6]volatile.Register32 //0x00,04,08,0C,10, and 14
	reserved00               volatile.Register32    //0x18
	OutputSet0               volatile.Register32    //0x1C
	OutputSet1               volatile.Register32    //0x20
	reserved01               volatile.Register32    //0x24
	OutputClear0             volatile.Register32    //0x28
	OutputClear1             volatile.Register32    //0x2C
	reserved03               volatile.Register32    //0x30
	Level0                   volatile.Register32    //0x34
	Level1                   volatile.Register32    //0x38
	reserved04               volatile.Register32    //0x3C
	EventDetectStatus0       volatile.Register32    //0x40
	EventDetectStatus1       volatile.Register32    //0x44
	reserved05               volatile.Register32    //0x48
	RisingEdgeDetectEnable0  volatile.Register32    //0x4C
	RisingEdgeDetectEnable1  volatile.Register32    //0x50
	reserved06               volatile.Register32    //0x54
	FallingEdgeDetectEnable0 volatile.Register32    //0x58
	FallingEdgeDetectEnable1 volatile.Register32    //0x5C
	reserved07               volatile.Register32    //0x60
	HighDetectEnable0        volatile.Register32    //0x64
	HighDetectEnable1        volatile.Register32    //0x68
	reserved08               volatile.Register32    //0x6C
	LowDetectEnable0         volatile.Register32    //0x70
	LowDetectEnable1         volatile.Register32    //0x74
	reserved09               volatile.Register32    //0x78
	AsyncRisingEdgeDetect0   volatile.Register32    //0x7C
	AsyncRisingEdgeDetect1   volatile.Register32    //0x80
	reserved0A               volatile.Register32    //0x84
	AsyncFallingEdgeDetect0  volatile.Register32    //0x88
	AsyncFallingEdgeDetect1  volatile.Register32    //0x8C
	reserved0B               volatile.Register32    //0x90
	PullUpDownEnable         volatile.Register32    // 0x94
	PullUpDownEnableClock0   volatile.Register32    //0x98
	PullUpDownEnableClock1   volatile.Register32    //0x9C
	reserved0C               volatile.Register32    //0xA0
	test                     volatile.Register32    //0xA4
}

type GPIOMode uint32 //3 bits wide
const GPIOInput GPIOMode = 0
const GPIOOutput GPIOMode = 1
const GPIOAltFunc5 GPIOMode = 2
const GPIOAltFunc4 GPIOMode = 3
const GPIOAltFunc0 GPIOMode = 4
const GPIOAltFunc1 GPIOMode = 5
const GPIOAltFunc2 GPIOMode = 6
const GPIOAltFunc3 GPIOMode = 7

//func GPIOSetup(pinNumber uint8, mode GPIOMode) bool {
//	if pinNumber > 54 { //54 pins on RPI
//		return false
//	} // Check GPIO pin number valid, return false if invalid
//	var shift uint8
//	shift = ((pinNumber % 10) * 3) // Create shift amount
//
//	value := uint32(7)
//	mask := uint32(7)
//
//	GPIO.FuncSelect[int(mode)].ReplaceBits(value, mask, shift)
//	return true // Return true
//}
