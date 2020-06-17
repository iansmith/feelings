package upbeat

import (
	"unsafe"

	"machine"
)

// Hex32string is for dumping a 32bit int without using fmt
func Hex32string(uart *machine.UART, d uint32) {
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

// Hex64string is for dumping a 64bit int without using fmt
func Hex64string(uart *machine.UART, d uint64) {
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

// Dump sends 512 bytes of ram to UART without using fmt
func DumpMemory(u *machine.UART, ptr unsafe.Pointer) {
	var a, b uint64
	var d byte
	var c byte

	for a = uint64(uintptr(ptr)); a < uint64(uintptr(ptr))+512; a += 16 {
		Hex32string(u, uint32(a))
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
		u.WriteByte(10)
	}
}
