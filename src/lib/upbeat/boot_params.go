package upbeat

type BootloaderParamsDef struct {
	EntryPoint   uint64
	KernelLast   uint64
	UnixTime     uint64
	StackPointer uint64
}
