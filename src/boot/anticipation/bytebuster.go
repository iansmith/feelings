package anticipation

import (
	"unsafe"
)

//
// byteBuster is the interface the bootloader's code calls to actually
// write bytes into memory or set key values.
//
type byteBuster interface {
	Write(addr uint64, value uint8) bool
	SetEntryPoint(addr uint64)
	EntryPoint() uint64
	EntryPointIsSet() bool
	SetParameter(i int, value uint64)
	GetParameter(i int) uint64
}

const entryPointSentinal = 0x22222

/////////////////////////////////////////////////////////////////////////
// MetalByteBuster
////////////////////////////////////////////////////////////////////////
//
// metalByBuster is the byteBuster that is actually used on the real hardware.
//
type MetalByteBuster struct {
	entryPoint    uint64
	hasEntryPoint bool
	param         [4]uint64
}

//set entry point is a 64bit addr
func (m *MetalByteBuster) SetEntryPoint(addr uint64) {
	m.entryPoint = addr
	m.hasEntryPoint = true
}

// the fake one can only process one line of data
func NewMetalByteBuster() *MetalByteBuster {
	return &MetalByteBuster{}
}

func (m *MetalByteBuster) EntryPoint() uint64 {
	return m.entryPoint
}

func (m *MetalByteBuster) Write(addr uint64, value uint8) bool {
	//log.Printf("writing at %x: %v", addr, value)
	a := (*uint8)(unsafe.Pointer(uintptr(addr)))
	*a = value
	return true
}

func (m *MetalByteBuster) EntryPointIsSet() bool {
	return m.hasEntryPoint
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
	entryPoint    uint64
	hasEntryPoint bool
}

func (n *nullByteBuster) SetEntryPoint(addr uint64) {
	n.entryPoint = addr
	n.hasEntryPoint = true
}

func NewNullByteBuster() *nullByteBuster {
	bb := &nullByteBuster{
		entryPoint: entryPointSentinal,
	}
	return bb
}

func (n *nullByteBuster) EntryPointIsSet() bool {
	return n.hasEntryPoint
}

func (n *nullByteBuster) EntryPoint() uint64 {
	return n.entryPoint
}

func (n *nullByteBuster) Write(addr uint64, value uint8) bool {
	return true
}

func (n *nullByteBuster) SetParameter(i int, v uint64) {
}
func (n *nullByteBuster) GetParameter(i int) uint64 {
	return 0
}
