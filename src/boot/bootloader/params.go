package bootloader

//go:extern bootloader_params
var InjectedParams ParamsDef

type ParamsDef struct {
	EntryPoint      uint64
	KernelCodeStart uint64 // Smallest address
	UnixTime        uint64
	StackStart      uint64 // Smallest address
	HeapStart       uint64 // Smallest address
	ReadOnlyStart   uint64 // Smallest address
	ReadWriteStart  uint64 // Smallest address
	PageCounts      uint64 // one byte each
}

const KernelCodePagesMask = (uint64(0xff << 0))
const StackPagesMask = (uint64(0xff << 8))
const HeapPagesMask = (uint64(0xff << 16))
const ReadOnlyPagesMask = (uint64(0xff << 24))
const ReadWritePagesMask = (uint64(0xff << 32))

func (b *ParamsDef) KernelCodePages() uint8 {
	v := b.PageCounts & KernelCodePagesMask
	return uint8(v)
}
func (b *ParamsDef) StackPages() uint8 {
	v := (b.PageCounts & StackPagesMask) >> 8
	return uint8(v)
}
func (b *ParamsDef) HeapPages() uint8 {
	v := (b.PageCounts & HeapPagesMask) >> 16
	return uint8(v)
}
func (b *ParamsDef) ReadOnlyPages() uint8 {
	v := (b.PageCounts & ReadOnlyPagesMask) >> 24
	return uint8(v)
}
func (b *ParamsDef) ReadWritePages() uint8 {
	v := (b.PageCounts & ReadWritePagesMask) >> 32
	return uint8(v)
}

func (b *ParamsDef) SetKernelCodePages(p uint8) {
	v := b.PageCounts & (^(KernelCodePagesMask))
	v |= uint64(p)
	b.PageCounts = v
}

func (b *ParamsDef) SetStackPages(p uint8) {
	v := b.PageCounts & (^(StackPagesMask))
	v |= (uint64(p) << 8)
	b.PageCounts = v
}

func (b *ParamsDef) SetHeapPages(p uint8) {
	v := b.PageCounts & (^(HeapPagesMask))
	v |= (uint64(p) << 16)
	b.PageCounts = v
}
func (b *ParamsDef) SetReadOnlyPages(p uint8) {
	v := b.PageCounts & (^(ReadOnlyPagesMask))
	v |= (uint64(p) << 24)
	b.PageCounts = v
}
func (b *ParamsDef) SetReadWritePages(p uint8) {
	v := b.PageCounts & (^(ReadWritePagesMask))
	v |= (uint64(p) << 32)
	b.PageCounts = v
}
