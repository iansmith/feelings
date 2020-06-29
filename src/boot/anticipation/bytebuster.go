package anticipation

import (
	"log"
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
	BaseAddress() uint64
	EntryPoint() uint64
	EntryPointIsSet() bool
	SetBigEntryPoint(addr uint32)
	SetBigBaseAddr(addr uint32)
	SetParameter(i int, value uint64)
	GetParameter(i int) uint64
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
	t          *testing.T
	param      [4]uint64
}

func (f *fakeByteBuster) SetEntryPoint(addr uint32) {
	prev := f.entryPoint & 0xffff_ffff_0000_0000
	f.entryPoint = prev | uint64(addr)
}

func (f *fakeByteBuster) SetParameter(i int, v uint64) {
	f.param[i] = v
}
func (f *fakeByteBuster) GetParameter(i int) uint64 {
	return f.param[i]
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
	written    uint32
	param      [4]uint64
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

// SetBigBaseAddr sets the HIGH order 32 bits of the base address
func (m *MetalByteBuster) SetBigBaseAddr(addr uint32) {
	prev := m.baseAdd & 0xffff_ffff
	m.baseAdd = prev | (uint64(addr) << 32)

}

// SetBaseAddr sets the LOW order 32 bits of the base address
func (m *MetalByteBuster) SetBaseAddr(addr uint32) {
	prev := m.baseAdd & 0xffff_ffff_0000_0000
	m.baseAdd = prev | uint64(addr)

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

var tmp = uint64(0xf)

func (m *MetalByteBuster) Write(addr uint64, value uint8) bool {
	a := (*uint8)(unsafe.Pointer(uintptr(addr)))
	*a = value

	if (addr & ^(tmp)) == 0xfffffc0030000000 {
		log.Printf("xxx at place: %x, value %x", addr, value)
	}
	m.written++
	return true
}
func (m *MetalByteBuster) EntryPointIsSet() bool {
	return m.entryPoint != entryPointSentinal
}
func (m *MetalByteBuster) SetParameter(i int, v uint64) {
	m.param[i] = v
}
func (m *MetalByteBuster) GetParameter(i int) uint64 {
	return m.param[i]
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
func (n *nullByteBuster) SetParameter(i int, v uint64) {
}
func (n *nullByteBuster) GetParameter(i int) uint64 {
	return 0
}
