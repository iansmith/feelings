package anticipation

import (
	"testing"
	"unsafe"
)

//
// byteBuster is the interface the bootloader's code calls to actually
// write bytes into memory or set key values.
//
type byteBuster interface {
	Write(addr uint32, value uint8) bool
	SetBaseAddr(addr uint32)
	SetEntryPoint(addr uint32)
	SetUnixTime(addr uint32)
	BaseAddress() uint32
	EntryPoint() uint32
	UnixTime() uint32
	EntryPointIsSet() bool
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
	baseAdd    uint32
	lineOffset uint32
	values     []byte
	entryPoint uint32
	unixtime   uint32
	t          *testing.T
}

func (f *fakeByteBuster) SetEntryPoint(addr uint32) {
	f.entryPoint = addr
}

func (f *fakeByteBuster) SetUnixTime(t uint32) {
	f.unixtime = t
}

func (f *fakeByteBuster) SetBaseAddr(addr uint32) {
	f.baseAdd = addr
}

func (f *fakeByteBuster) FinishedOk() bool {
	return f.written == len(f.values)
}

// the fake one can only process one line of data
func newFakeByteBuster(data []byte, onlyLineOffset uint32) *fakeByteBuster {
	bb := &fakeByteBuster{lineOffset: onlyLineOffset, values: data}
	bb.entryPoint = entryPointSentinal
	return bb
}

func (f *fakeByteBuster) EntryPointIsSet() bool {
	return f.entryPoint != entryPointSentinal
}

func (f *fakeByteBuster) BaseAddress() uint32 {
	return f.baseAdd
}

func (f *fakeByteBuster) EntryPoint() uint32 {
	return f.entryPoint
}

func (f *fakeByteBuster) UnixTime() uint32 {
	return f.unixtime
}

func (f *fakeByteBuster) Write(addr uint32, value uint8) bool {
	//f.t.Logf("addr %x value %x, addrExpected %x valueExpected %x",addr,value,baseAddr,f.values[f.written])
	if addr != f.BaseAddress()+f.lineOffset+uint32(f.written) {
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
	baseAdd    uint32
	lineOffset uint32
	entryPoint uint32
	unixTime   uint32
	written    uint32
}

func (m *MetalByteBuster) SetEntryPoint(addr uint32) {
	print("@ setting entry point of stage 1 to ", addr, "\n")
	m.entryPoint = addr
}

func (m *MetalByteBuster) SetUnixTime(t uint32) {
	print("@setting current time to unix ", t, "\n")
	m.unixTime = t
}

func (m *MetalByteBuster) SetBaseAddr(addr uint32) {
	print("@ setting base address for download to ", addr, "\n")
	m.baseAdd = addr
}

// the fake one can only process one line of data
func NewMetalByteBuster() *MetalByteBuster {
	bb := &MetalByteBuster{}
	bb.entryPoint = entryPointSentinal
	return bb
}

func (m *MetalByteBuster) BaseAddress() uint32 {
	return m.baseAdd
}

func (m *MetalByteBuster) EntryPoint() uint32 {
	return m.entryPoint
}

func (m *MetalByteBuster) UnixTime() uint32 {
	return m.unixTime
}

func (m *MetalByteBuster) Write(addr uint32, value uint8) bool {
	a := (*uint8)(unsafe.Pointer(uintptr(addr)))
	*a = value
	m.written++
	return true
}
func (m *MetalByteBuster) EntryPointIsSet() bool {
	return m.entryPoint != entryPointSentinal
}
