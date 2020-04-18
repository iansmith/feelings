package tinygo_runtime

import (
	p "feelings/src/hardware/bcm2835"

	"github.com/tinygo-org/tinygo/src/device/arm"
)

//decls
var MiniUART *UART

func Abort(s string) {
	MiniUART.WriteString("Aborting..." + s + "\n")
	for {
		arm.Asm("nop")
	}
}

//
// Wait MuSec waits for at least n musecs based on the system timer. This is a busy wait.
//
//go:export WaitMuSec
func WaitMuSec(n uint64) {
	var f, t, r uint64
	arm.AsmFull(`mrs x28, cntfrq_el0
		str x28,{f}
		mrs x27, cntpct_el0
		str x27,{t}`, map[string]interface{}{"f": &f, "t": &t})
	//expires at t
	t += ((f / 1000) * n) / 1000
	for r < t {
		arm.AsmFull(`mrs x27, cntpct_el0
			str x27,{r}`, map[string]interface{}{"r": &r})
	}
}

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
