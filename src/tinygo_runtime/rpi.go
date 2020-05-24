package tinygo_runtime

import (
	p "hardware/bcm2835"

	"unsafe"

	"device/arm"
)

//
// RPI has many uarts, this is the "miniuart" which is the simplest to configure.
//
const RxBufMax = 0xfff

type UART struct {
	rxhead   int
	rxtail   int
	rxbuffer []uint8
}

func NewUART() *UART {
	return &UART{
		rxhead:   0,
		rxtail:   0,
		rxbuffer: make([]uint8, RxBufMax+1),
	}
}

//
// For now, zero value is a non-interrupt UART at 8bits, bidirectional.
// If you set EnableRXInterrupt, you'll need to actually turn on the
// interrupts when you are ready.
type UARTConfig struct {
	RXInterrupt bool
	Data7Bits   bool
	DisableTx   bool
	DisableRx   bool
}

// Configure accepts a config object to set some simple
// properties of the UART.  It is not a fully featured 16550 UART,
// rather it is the "mini" UART.   The zero value of conf
// gives you 8 bits, no interrupts, and both tx and rx enabled.
func (uart *UART) Configure(conf UARTConfig) error {
	var r uint32

	p.Aux.Enables.SetBits(p.PeripheralMiniUART) //enable AUX Mini uart

	//turn off the transmitter and receiver
	p.Aux.MiniUARTExtraControl.ClearBits(p.ReceiveEnable | p.TransmitEnable)
	p.Aux.MiniUARTExtraControl.Set(0)

	//configure data bits
	if conf.Data7Bits {
		p.Aux.MiniUARTLineControl.ClearBits(p.DataLength8Bits) //7 bits
	} else {
		//see errata for why (bad docs!) uses excuse of compat with 16550
		// https://elinux.org/BCM2835_datasheet_errata#p14
		p.Aux.MiniUARTLineControl.SetBits(p.DataLength8Bits)
	}

	p.Aux.MiniUARTModemControl.ClearBits(p.ReadyToSend) // this asserts the line
	p.Aux.MiniUARTInterruptIdentify.ReplaceBits(p.ClearTransmitFIFO|p.ClearReceiveFIFO, p.ClearFIFOsMask, 0 /*no shift*/)

	// derived from clock speed: BCM2835 ARM Peripheral manual page 11
	p.Aux.MiniUARTBAUD.Set(270) // 115200 baud

	//set the bits
	if conf.RXInterrupt {
		p.Aux.MiniUARTInterruptEnable.SetBits(p.ReceiveFIFOReady)
	} else {
		p.Aux.MiniUARTInterruptEnable.ClearBits(p.ReceiveFIFOReady | p.TransmitFIFOEmpty | p.LineStatusError | p.ModemStatusChange)
	}

	// map UART1 to GPIO pins
	p.GPIOSetup(14, p.GPIOAltFunc5)
	p.GPIOSetup(15, p.GPIOAltFunc5)

	//sleep 150 cycles
	r = 150
	for r > 0 {
		r--
		arm.Asm("nop")
	}

	p.GPIO.PullUpDownEnableClock0.SetBits((1 << 14) | (1 << 15))

	//sleep 150 cycles
	r = 150
	for r > 0 {
		r--
		arm.Asm("nop")
	}

	p.GPIO.PullUpDownEnableClock0.Set(0) //flush gpio setup

	if !conf.DisableRx {
		p.Aux.MiniUARTExtraControl.SetBits(p.ReceiveEnable)
	} else {
		p.Aux.MiniUARTExtraControl.ClearBits(p.ReceiveEnable)
	}
	if !conf.DisableTx {
		p.Aux.MiniUARTExtraControl.SetBits(p.TransmitEnable)
	} else {
		p.Aux.MiniUARTExtraControl.ClearBits(p.TransmitEnable)
	}

	return nil
}

//
// Writing a byte over serial.  Blocking.
//
//go:noinline
func (uart UART) WriteByte(c byte) error {
	// wait until we can send
	for {
		if p.Aux.MiniUARTLineStatus.HasBits(p.TransmitFIFOSpaceAvailable) {
			break
		}
		arm.Asm("nop")
	}

	// write the character to the buffer
	c32 := uint32(c)
	p.Aux.MiniUARTData.Set(c32) //really 8 bit write
	return nil
}

//
// Write a CR (and secretly an LF) to serial.
//
func (uart *UART) WriteCR() error {
	if err := uart.WriteByte(13); err != nil {
		return err
	}
	if err := uart.WriteByte(10); err != nil {
		return err
	}
	return nil
}

//
// Reading a byte from serial. Blocking.
//
func (uart *UART) ReadByte() uint8 {
	for {
		if p.Aux.MiniUARTLineStatus.HasBits(p.ReceiveFIFOReady) {
			break
		}
		arm.Asm("nop")
	}
	r := p.Aux.MiniUARTData.Get() //8 bit read
	return uint8(r)
}

//
// Put a whole string out to serial. Blocking.
//
func (uart *UART) WriteString(s string) error {
	for i := 0; i < len(s); i++ {
		uart.WriteByte(s[i])
	}
	return nil
}

func (uart *UART) Hex32string(d uint32) {
	var rb uint32
	var rc uint32

	rb = 32
	for {
		rb -= 4
		rc = (d >> rb) & 0xF
		if rc > 9 {
			rc += 0x37
		} else {
			rc += 0x30
		}
		uart.WriteByte(uint8(rc))
		if rb == 0 {
			break
		}
	}
	uart.WriteByte(0x20)
}

func (uart *UART) Hex64string(d uint64) {
	var rb uint64
	var rc uint64

	rb = 64
	for {
		rb -= 4
		rc = (d >> rb) & 0xF
		if rc > 9 {
			rc += 0x37
		} else {
			rc += 0x30
		}
		uart.WriteByte(uint8(rc))
		if rb == 0 {
			break
		}
	}
	uart.WriteByte(0x20)
}

// CopyRxBuffer copies entire RX buffer into target and leaves it empty
// Caller needs to insure that all of the buffer can fit into target
func (uart *UART) CopyRxBuffer(target []byte) uint32 {
	moved := uint32(0)
	found := false
	for !uart.EmptyRx() {
		index := uart.rxtail
		target[moved] = uart.rxbuffer[index]
		targ := target[moved]
		if !found {
			moved++
		}
		if targ == 10 {
			found = true
		}
		tail := index + 1
		tail &= RxBufMax
		uart.rxtail = tail
	}
	return moved
}

// DumpRxBuffer pushes the entire RX buffer out to serial and leaves the
// buffer empty.
func (uart *UART) DumpRxBuffer() uint32 {
	moved := uint32(0)
	for !uart.EmptyRx() {
		index := uart.rxtail
		uart.WriteByte(uart.rxbuffer[index])
		tail := index + 1
		tail &= RxBufMax
		uart.rxtail = tail
		moved++
	}
	return moved
}

// LoadRx puts a byte in the RxBuffer as if it came in from
// the other side.  Probably should be called from an exception
// hadler.
func (uart *UART) LoadRx(b uint8) {
	//receiver holds a valid byte
	index := uart.rxhead
	uart.rxbuffer[index] = b
	head := index + 1
	head &= RxBufMax
	uart.rxhead = head
}

// EmptyRx is true if the receiver ring buffer is empty.
func (uart *UART) EmptyRx() bool {
	return uart.rxtail == uart.rxhead
}

// Returns the next element from the read queue.  Note that
// this busy waits on EmptyRx() so you should be sure
// there is data there before you call this or it will block
// and only an interrupt can save that...
func (uart *UART) NextRx() uint8 {
	for {
		if !uart.EmptyRx() {
			break
		}
	}
	result := uart.rxbuffer[uart.rxtail]
	tail := uart.rxtail + 1
	tail &= RxBufMax
	uart.rxtail = tail
	return result
}

/**
 * Dump memory
 */
func (u *UART) Dump(ptr unsafe.Pointer) {
	var a, b uint64
	var d byte
	var c byte

	for a = uint64(uintptr(ptr)); a < uint64(uintptr(ptr))+512; a += 16 {
		u.Hex32string(uint32(a))
		u.WriteString(": ")
		for b = 0; b < 16; b++ {
			c = *((*byte)(unsafe.Pointer(uintptr(a + b))))
			d = c
			d >>= 4
			d &= 0xF
			if d > 9 {
				d += 0x37
			} else {
				d += 0x30
			}
			u.WriteByte(byte(d))
			d = c
			d &= 0xF
			if d > 9 {
				d += 0x37
			} else {
				d += 0x30
			}
			u.WriteByte(byte(d))
			u.WriteByte(' ')
			if b%4 == 3 {
				u.WriteByte(' ')
			}
		}
		for b = 0; b < 16; b++ {
			c = *((*byte)(unsafe.Pointer(uintptr(a + b))))
			if c < 32 || c > 127 {
				u.WriteByte('.')
			} else {
				u.WriteByte(c)
			}
		}
		u.WriteCR()
	}
}

func BoardRevisionDecode(s string) string {
	switch s {
	case "9020e0":
		return "3A+, Revision 1.0, 512MB, Sony UK"
	case "a02082":
		return "3B, Revision 1.2, 1GB, Sony UK"
	case "a020d3":
		return "3B+, Revision 1.3, 1GB, Sony UK"
	case "a22082":
		return "3B, Revision 1.2, 1GB, Embest"
	case "a220a0":
		return "CM3, Revision 1.0, 1GB, Embest"
	case "a32082":
		return "3B, Revision 1.2, 1GB, Sony Japan"
	case "a52082":
		return "3B, Revision 1.2, 1GB, Stadium"
	case "a22083":
		return "3B, Revision 1.3, 1GB, Embest"
	case "a02100":
		return "CM3+, Revision 1.0, 1GB, Sony UK"
	case "a03111":
		return "4B, Revision 1.1, 2GB, Sony UK"
	case "b03111":
		return "4B, Revision 1.2, 2GB, Sony UK"
	case "b03112":
		return "4B, Revision 1.2, 2GB, Sony UK"
	case "c03111":
		return "4B, Revision 1.1, 4GB, Sony UK"
	case "c03112":
		return "4B, Revision 1.2, 4GB, Sony UK"
	}
	return "unknown board"
}

//
// Wait MuSec waits for at least n musecs based on the system timer. This is a busy wait.
//
//func WaitMuSec(n uint64) {
//	start:=runtime.Semihostingv2Call(uint64(runtime.Semihostingv2OpClock), 0)
//	centis:=int64(n)/tickMicros
//	current:=start
//	for current-start<centis{ //busy wait
//		for i:=0; i<20; i++ {
//			arm.Asm("nop")
//		}
//		current=runtime.Semihostingv2Call(uint64(runtime.Semihostingv2OpClock), 0)
//	}
//}

//
// SysTimer gets the 64 bit timer's value.
//
//func SystemTime() uint64 {
//	current:=runtime.Semihostingv2Call(uint64(runtime.Semihostingv2OpClock), 0)
//	return current*machine.tickMicros
//}

//go:noinline
func printoutException(esr uint64) {
	exceptionClass := esr >> 26
	switch exceptionClass {
	case 0:
		c.Logf("unknown exception")
	case 1:
		c.Logf("trapped WFE or WFI instruction")
	case 2, 8, 9, 10, 11, 15, 16, 18, 19, 20, 22, 23, 26, 27, 28, 29, 30, 31, 35, 38, 39, 41, 42, 43, 45, 46, 54, 55, 57, 58, 59, 61, 62, 63:
		c.Logf("unused code!!")
	case 3:
		c.Logf("trapped MRRC or MCRR access")
	case 4:
		c.Logf("trapped MRRC or MCRR access")
	case 5:
		c.Logf("trapped MRC or MCR access")
	case 6:
		c.Logf("trapped LDC or STC access")
	case 7:
		c.Logf("access to SVE, advanced SIMD or FP functionality")
	case 12:
		c.Logf("trapped to MRRC access")
	case 13:
		c.Logf("branch target exception")
	case 14:
		c.Logf("illegal execution state")
	case 17:
		c.Logf("SVC instruction in AARCH32")
		c.Logf("[", esr&0xffff, "]")
	case 21:
		c.Logf("SVC instruction in AARCH64")
		c.Logf("[", esr&0xffff, "]")
	case 24:
		c.Logf("trapped MRS, MSR or System instruction in AARCH64")
	case 25:
		c.Logf("access to SVE functionality")
	case 32:
		c.Logf("instruction abort from lower exception level")
	case 33:
		c.Logf("instruction abort from same exception level")
	case 34:
		c.Logf("PC alignment fault")
	case 36:
		c.Logf("data abort from lower exception level")
	case 37:
		c.Logf("data abort from same exception level")
	case 40:
		c.Logf("trapped floating point exception from AARCH32")
	case 44:
		c.Logf("trapped floating point exception from AARCH64")
	case 47:
		c.Logf("SError exception")
	case 48:
		c.Logf("Breakpoint from lower exception level")
	case 49:
		c.Logf("Breakpoint from same exception level")
	case 50:
		c.Logf("Software step from lower exception level")
	case 51:
		c.Logf("Software step from same exception level")
	case 52:
		c.Logf("Watchpoint from lower exception level")
	case 53:
		c.Logf("Watchpoint from same exception level")
	case 56:
		c.Logf("BKPT from AARCH32")
	case 60:
		c.Logf("BRK from AARCH64")
	}
	c.Logf("\n")

}
