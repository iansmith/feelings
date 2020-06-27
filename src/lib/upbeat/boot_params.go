package upbeat

type BootloaderParamsDef struct {
	EntryPoint   uint64
	KernelLast   uint64
	UnixTime     uint64
	StackPointer uint64
	HeapStart    uint64
	HeapEnd      uint64
}

//go:extern bootloader_params
var BootloaderParams BootloaderParamsDef
