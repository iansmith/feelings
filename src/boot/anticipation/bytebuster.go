package anticipation

import (
	"fmt"
	"testing"
	"unsafe"
)

//
// byteBuster is the interface the bootloader's code calls to actually
// write bytes into memory or set key values.
//
type byteBuster interface {
	Write(addr uint64, value uint8) bool
	SetBaseAddr(addr uint32)
	SetEntryPoint(addr uint32)
	SetUnixTime(addr uint32)
	BaseAddress() uint64
	EntryPoint() uint64
	EntryPointIsSet() bool

	SetBigEntryPoint(addr uint32)
	SetBigBaseAddr(addr uint32)
	UnixTime() uint32
}

const entryPointSentinal = 0x22222

/////////////////////////////////////////////////////////////////////////
// fakeByteBuster
////////////////////////////////////////////////////////////////////////

//
// fakeByteBuster is used for testing because testing on baremetal is hard.
//
type fakeByteBuster struct {
	written    int
	baseAdd    uint64
	lineOffset uint64
	values     []byte
	entryPoint uint64
	unixtime   uint32
	t          *testing.T
}

func (f *fakeByteBuster) SetEntryPoint(addr uint32) {
	prev := f.entryPoint & 0xffff_ffff_0000_0000
	f.entryPoint = prev | uint64(addr)
}

func (f *fakeByteBuster) SetUnixTime(t uint32) {
	f.unixtime = t
}

func (f *fakeByteBuster) SetBaseAddr(addr uint32) {
	prev := f.baseAdd & 0xffff_ffff_0000_0000
	f.baseAdd = prev | uint64(addr)
}

func (f *fakeByteBuster) SetBigEntryPoint(addr uint32) {
	prev := f.entryPoint & 0xffff_ffff
	f.entryPoint = prev | (uint64(addr) << 32)
}
func (f *fakeByteBuster) SetBigBaseAddr(addr uint32) {
	prev := f.baseAdd & 0xffff_ffff
	f.baseAdd = prev | (uint64(addr) << 32)
}

func (f *fakeByteBuster) FinishedOk() bool {
	return f.written == len(f.values)
}

// the fake one can only process one line of data
func newFakeByteBuster(data []byte, onlyLineOffset uint64) *fakeByteBuster {
	bb := &fakeByteBuster{lineOffset: onlyLineOffset, values: data}
	bb.entryPoint = entryPointSentinal
	return bb
}

func (f *fakeByteBuster) EntryPointIsSet() bool {
	return f.entryPoint != entryPointSentinal
}

func (f *fakeByteBuster) BaseAddress() uint64 {
	return f.baseAdd
}

func (f *fakeByteBuster) EntryPoint() uint64 {
	return f.entryPoint
}

func (f *fakeByteBuster) UnixTime() uint32 {
	return f.unixtime
}

func (f *fakeByteBuster) Write(addr uint64, value uint8) bool {
	//f.t.Logf("addr %x value %x, addrExpected %x valueExpected %x",addr,value,baseAddr,f.values[f.written])
	if addr != f.BaseAddress()+f.lineOffset+uint64(f.written) {
		return false
	}
	if f.written >= len(f.values) {
		return false
	}
	if f.values[f.written] != value {
		return false
	}
	f.written++
	return true
}

/////////////////////////////////////////////////////////////////////////
// MetalByteBuster
////////////////////////////////////////////////////////////////////////
//
// metalByBuster is the byteBuster that is actually used on the real hardware.
//
type MetalByteBuster struct {
	baseAdd    uint64
	lineOffset uint32
	entryPoint uint64
	unixTime   uint32
	written    uint32
}

//set entry point affects the LOWER 32 bits of the entry point
func (m *MetalByteBuster) SetEntryPoint(addr uint32) {
	prev := m.entryPoint & 0xffff_ffff_0000_0000
	m.entryPoint = prev | uint64(addr)
}

//set big entry point affects the UPPER 32 bits of the entry point
func (m *MetalByteBuster) SetBigEntryPoint(addr uint32) {
	prev := m.entryPoint & 0xffff_ffff
	m.entryPoint = prev | (uint64(addr) << 32)
}
func (m *MetalByteBuster) SetUnixTime(t uint32) {
	m.unixTime = t
}

// SetBigBaseAddr sets the HIGH order 32 bits of the base address
func (m *MetalByteBuster) SetBigBaseAddr(addr uint32) {
	prev := m.baseAdd & 0xffff_ffff
	m.baseAdd = prev | (uint64(addr) << 32)
	fmt.Printf("set BIG base address 0x%x\n", m.baseAdd)

}

// SetBaseAddr sets the LOW order 32 bits of the base address
func (m *MetalByteBuster) SetBaseAddr(addr uint32) {
	prev := m.baseAdd & 0xffff_ffff_0000_0000
	m.baseAdd = prev | uint64(addr)
	fmt.Printf("set base address 0x%x\n", m.baseAdd)

}

// the fake one can only process one line of data
func NewMetalByteBuster() *MetalByteBuster {
	bb := &MetalByteBuster{}
	bb.entryPoint = entryPointSentinal
	return bb
}

func (m *MetalByteBuster) BaseAddress() uint64 {
	return m.baseAdd
}

func (m *MetalByteBuster) EntryPoint() uint64 {
	return m.entryPoint
}

func (m *MetalByteBuster) UnixTime() uint32 {
	return m.unixTime
}

func (m *MetalByteBuster) Write(addr uint64, value uint8) bool {
	crap := uint64(0xff)
	crap = ^crap
	if addr == 0xfffffc0030001548 {
		fmt.Printf("at the entry point 0x%x, the byte is 0x%x\n", addr, value)
	}
	if addr&crap == addr {
		fmt.Printf("reached 0x%x\n", addr)
	}
	a := (*uint8)(unsafe.Pointer(uintptr(addr)))
	*a = value
	m.written++
	return true
}
func (m *MetalByteBuster) EntryPointIsSet() bool {
	return m.entryPoint != entryPointSentinal
}

/////////////////////////////////////////////////////////////////////////
// NullByteBuster
////////////////////////////////////////////////////////////////////////
type nullByteBuster struct {
	unixTime   uint32
	addr       uint64
	entryPoint uint64
}

func (n *nullByteBuster) SetEntryPoint(addr uint32) {
	n.entryPoint = uint64(addr)
}

func (n *nullByteBuster) SetUnixTime(t uint32) {
	n.unixTime = t
}

func (n *nullByteBuster) SetBaseAddr(addr uint32) {
	n.addr = uint64(addr)
}

func NewNullByteBuster() *nullByteBuster {
	bb := &nullByteBuster{
		entryPoint: entryPointSentinal,
	}
	return bb
}

func (n *nullByteBuster) EntryPointIsSet() bool {
	return n.entryPoint != entryPointSentinal
}

func (n *nullByteBuster) BaseAddress() uint64 {
	return n.addr
}

func (n *nullByteBuster) EntryPoint() uint64 {
	return n.entryPoint
}

func (n *nullByteBuster) UnixTime() uint32 {
	return n.unixTime
}

func (n *nullByteBuster) Write(addr uint64, value uint8) bool {
	return true
}

func (n *nullByteBuster) SetBigEntryPoint(addr uint32) {
}

func (n *nullByteBuster) SetBigBaseAddr(addr uint32) {
}
