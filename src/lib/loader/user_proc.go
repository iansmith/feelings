package loader
package loader

import "boot/bootloader"

const PageSize = 0x1_0000
type UserProcStartupInfo struct {
	Filename              string
	PageSize        uint64
	Processor             uint64 // 0-3 on RPi3
	ProcCodePlacementPhys uint64
	ProcLevel2Phys        uint64
	ProcLevel3Phys        uint64
	ProcHeapLoc           uint64 //in pages, for computing placement
	ProcHeapVirt          uint64
	ProcHeapSize          uint64 //in pages, must be <256
	ProcStackLoc          uint64 //in pages, for computing placement
	ProcStackVirt         uint64
	ProcStackSize         uint64 //in pages, must be < 256
	ProcLinkVirt          uint64 // where proc THINKs it is running (VM unaware)
	MAIRVal               uint64
	TCREL1Val             uint64
	SCTRLEL1Val           uint64
	TTBR0Val              uint64 // user level
	// both in and out param, pass in names, returns with virt addrs from sym table
	SymbolToVirt map[string]uint64 //provided by the linker, thus are VM unaware
	// these are return values back to the caller
	NameToSizeInPages map[string]uint64
	EntryPoint        uint64
	TotalCodePages    uint64 //includes (.exc,.text,.data,.rodata,.bss)
	ROLowestVirt      uint64
	RWLowestVirt      uint64
}

func NewUserProcStartupInfo(filename string, processor uint64 /*0-3*/) *UserProcStartupInfo {
	return &UserProcStartupInfo{
		Filename:              filename,
		PageSize:       	   PageSize,
		Processor:             processor,
		ProcCodePlacementPhys: bootloader.MadeleinePlacement, //xxx only hardcoded value
		ProcLevel2Phys:        kernelProcToLevel2Phys(processor),
		ProcLevel3Phys:        kernelProcToLevel3Phys(processor),
		ProcHeapLoc:           bootloader.KernelProcessHeap,
		ProcHeapVirt:          bootloader.KernelProcessHeapAddr,
		ProcHeapSize:          fixedSizeOfHeap,
		ProcStackLoc:          bootloader.KernelProcessStack,
		ProcStackVirt:         bootloader.KernelProcessStackAddr,
		ProcLinkVirt:          bootloader.KernelProcessLinkAddr,
		ProcStackSize:         fixedSizeOfStack,
		MAIRVal:               MAIRVal,
		SCTRLEL1Val:           SCTRLEL1Val,
		TCREL1Val:             TCREL1Val,
		TTBR0Val:              TTBR0Val,
		SymbolToVirt:          map[string]uint64{bootloaderParams: 0},
		NameToSizeInPages:     nil, //return values overwrite
		EntryPoint:            0,   //return value should overwrite 0
	}
}
