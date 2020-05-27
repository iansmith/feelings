package upbeat

import (
	"machine"
	"unsafe"
)

// U
func (uart *machine.UART) Hex32string(d uint32) {
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

func (uart *machine.UART) Hex64string(d uint64) {
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

/**
 * Dump memory
 */
func (u *machine.UART) Dump(ptr unsafe.Pointer) {
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
