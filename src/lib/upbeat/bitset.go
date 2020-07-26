package upbeat

import (
	"fmt"
	"unsafe"
)

type BitSet struct {
	size uint32
	data unsafe.Pointer //actually an array of uint64s
}

type BitIndex uint32

//bitsets have to be multiples of 64.  the ptr provided should be
//already allocated statically to be the place to store the data.
//caller must be sure it is aligned properly.
func NewBitSet(size uint32, ptr unsafe.Pointer) *BitSet {
	mask := ^(uint32(0x3f))
	if size&mask != size {
		fmt.Printf("your bitset size is not a multiple of 64: %d", size)
		return nil
	}
	result := &BitSet{
		data: ptr,
		size: size,
	}
	result.ClearAll()
	return result
}

func (b *BitSet) On(bit BitIndex) bool {
	boff := uintptr(bit >> 6)       //divide by 8 to get number of bytes, 8 again for which uint64
	mask := uint64(1 << (bit % 64)) //which bit in the right
	tmp := (*uint64)(unsafe.Pointer(uintptr(b.data) + (8 * boff)))
	return (*tmp)&mask != 0
}
func (b *BitSet) Set(bit BitIndex) {
	boff := uintptr(bit >> 6)       //divide by 8 to get number of bytes, 8 again for which uint64
	mask := uint64(1 << (bit % 64)) //which bit in the right
	tmp := (*uint64)(unsafe.Pointer(uintptr(b.data) + (8 * boff)))
	v := (*tmp) | mask
	*tmp = v
}

func (b *BitSet) Clear(bit BitIndex) {
	boff := uintptr(bit >> 6)       //divide by 8 to get number of bytes, 8 again for which uint64
	mask := uint64(1 << (bit % 64)) //which bit in the right
	mask = ^mask
	tmp := (*uint64)(unsafe.Pointer(uintptr(b.data) + (8 * boff)))
	v := (*tmp) & mask
	*tmp = v
}

func (b *BitSet) ClearAll() {
	numUint64s := b.size >> 6 //really dividing by 64 because  512/8
	for i := uint32(0); i < numUint64s; i++ {
		curr := (*uint64)(unsafe.Pointer(uintptr(i*8) + uintptr(unsafe.Pointer(b.data))))
		*curr = 0
	}
}
