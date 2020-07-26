package main

import (
	"debug/elf"
	"fmt"
	"io"
	"unsafe"

	"device/arm"

	"boot/bootloader"
	"drivers/emmc"
	"lib/trust"
)

const maddie = "/feelings/madeleine"
const KernelPageSize = 0x10000
const sectionBufferSize = 512
const fixedSizeOfHeap = 4  //in pages
const fixedSizeOfStack = 2 //in pages

var sectionBuffer [sectionBufferSize]byte

type ptype int

const executableCode ptype = 1
const readOnlyData ptype = 2
const readWriteData ptype = 3
const unitializedData ptype = 4
const heapSpace ptype = 5
const stackSpace ptype = 6
const ptypeDone ptype = 7

var ptypeToSection = map[ptype]string{
	executableCode:  ".text",
	readOnlyData:    ".rodata",
	readWriteData:   ".data",
	unitializedData: ".bss",
}

func canBootFromDisk(logger *trust.Logger) bool {
	if emmc.Impl.Init() != emmc.EmmcOk {
		logger.Errorf("Unable to initialize EMMC driver, " +
			"booting from serial port...")
		return false
	}
	return true
}

const bootloaderParams = "bootloader_params"

type KernelProcStartupInfo struct {
	Filename              string
	KernelPageSize        uint64
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

var maddyParams = KernelProcStartupInfo{
	Filename:              maddie,
	KernelPageSize:        KernelPageSize,
	Processor:             bootloader.MadeleineProcessor,
	ProcCodePlacementPhys: bootloader.MadeleinePlacement,
	ProcLevel2Phys:        bootloader.MadelineTableLevel2,
	ProcLevel3Phys:        bootloader.MadelineTableLevel3Start,
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

func virtToPhysInKernelProc(table3Start uint64, vaddr uint64) uint64 {
	pagePtr := vaddr & (0x0000_0000_ffff_0000) //get rid of kernel prefix and index bits
	pagePtr = pagePtr >> 16                    //shift to make it an index into table level 3
	offsetInTable := pagePtr * 8               //64 bits per element
	index := (*uint64)((unsafe.Pointer)(uintptr(bootloader.MadelineTableLevel3Start) + uintptr(offsetInTable)))
	truePage := (*index & noLast16) // cruft in the last few bits for VM use only
	physAddr := (uintptr)(unsafe.Pointer(uintptr(truePage + (vaddr & (0xffff)))))
	return uint64(physAddr)
}

type LoaderError int

const LoaderNoError LoaderError = 0
const LoaderKernelProcFileNotFound LoaderError = -1
const LoaderKernelProcCannotAttachElf LoaderError = -2
const LoaderKernelProcCannotReadElfSection LoaderError = -3
const LoaderKernelProcCannotReadSymbolTable LoaderError = -4
const LoaderKernelProcCannotFindBootloaderParameters LoaderError = -5

func (e LoaderError) Error() string {
	return e.String()
}

func (e LoaderError) String() string {
	switch e {
	case 0:
		return "LoaderNoError"
	case -1:
		return "LoaderKernelProcNotFound"
	case -2:
		return "LoaderKernelProcCannotAttachElf"
	case -3:
		return "LoaderKernelProcCannotReadElfSection"
	case -4:
		return "LoaderKernelProcCannotSymbolTable"
	case -5:
		return "LoaderKernelProcCannotFindBootloaderParameters"
	default:
		return "unknown loader error code"
	}
}

func (k *KernelProcStartupInfo) KernelProcBootFromDisk() LoaderError {
	fp, err := emmc.Impl.Open(k.Filename)
	if err != nil {
		logger.Errorf("Unable to find %s binary, "+
			"booting from serial port...", maddie)
		return LoaderKernelProcFileNotFound

	}
	elfFile, err := elf.NewFile(fp)
	if err != nil {
		trust.Debugf("Error attaching elf reader: %v", err)
		return LoaderKernelProcCannotAttachElf
	}
	k.EntryPoint = elfFile.FileHeader.Entry

	totalExcText := uint64(0)

	k.NameToSizeInPages = make(map[string]uint64)
	//figure out the page sizes for these
	for _, sect := range elfFile.Sections {
		switch sect.Name {
		case ".text", ".bss", ".data", ".rodata", ".exc":
			overhang := uint64(1)
			if sect.Size%KernelPageSize == 0 {
				overhang = 0
			}
			sizeInPages := (sect.Size / KernelPageSize) + overhang
			if sect.Name == ".text" || sect.Name == ".exc" {
				totalExcText += sizeInPages
			} else {
				k.NameToSizeInPages[sect.Name] = sizeInPages
			}
		}
	}
	overhang := uint64(1)
	if totalExcText%KernelPageSize == 0 {
		overhang = 0
	}
	k.NameToSizeInPages[".text"] = totalExcText/KernelPageSize + overhang
	for _, v := range k.NameToSizeInPages {
		k.TotalCodePages += v
	}

	currentPhys := k.ProcCodePlacementPhys
	for _, sectName := range []string{".exc", ".text", ".rodata", ".data", ".bss"} {
		section := elfFile.Section(sectName)
		logger.Debugf("loading %-10s @ 0x%x", sectName, currentPhys)
		read := uint64(0)
		for read < section.Size {
			reader := elfFile.Section(sectName).Open()
			l := sectionBufferSize
			if section.Size-read < sectionBufferSize {
				l = int(section.Size - read)
			}
			n, err := reader.Read(sectionBuffer[:l])
			if err != nil {
				if err == io.EOF {
					break
				}
				logger.Errorf("Unable to read section %s: %v", sectName, err)
				return LoaderKernelProcCannotReadElfSection
			}
			for i := 0; i < n; i++ {
				ptr := (*byte)(unsafe.Pointer(uintptr(currentPhys) + uintptr(i)))
				*ptr = sectionBuffer[i]
			}
			currentPhys += uint64(n)
			read += uint64(n)
		}
		if sectName != ".exc" { //glom the .exc and the .text together
			if currentPhys%k.KernelPageSize != 0 {
				diff := k.KernelPageSize - (currentPhys % k.KernelPageSize)
				currentPhys += diff
			}
		}
	}
	symbols, err := elfFile.Symbols()
	if err != nil {
		trust.Errorf("unable access symbol table: %v", err)
		return LoaderKernelProcCannotReadSymbolTable
	}
	for _, s := range symbols {
		for target, _ := range k.SymbolToVirt {
			if s.Name == target {
				k.SymbolToVirt[target] = s.Value
				break
			}
		}
	}
	if k.SymbolToVirt["bootloader_params"] == 0 {
		return LoaderKernelProcCannotFindBootloaderParameters
	}

	k.buildVMTransTables()
	k.injectBootloaderParams()

	for {
		arm.Asm("nop")
	}

}

func (k *KernelProcStartupInfo) buildVMTransTables() {
	// there are 8192 entries on this table, but we only care about 1 of
	// them, which is the 0 entry..

	// level 2
	for i := 0; i < 8192; i++ {
		ptr := (*uint64)(unsafe.Pointer(uintptr(k.ProcLevel2Phys) +
			(uintptr(i) * 8)))
		if i == 0 {
			*ptr = makeTableEntry(uintptr(k.ProcLevel3Phys))
		} else {
			*ptr = makeBadEntry() // not a valid entry, so it will crap out with page fault
		}
	}

	// level 3

	pt := executableCode
	current := 0
	pageStart := k.ProcCodePlacementPhys // page we are pointing TO

	fixedSizedHeap := uint64(0)
	fixedSizedStack := uint64(0)

	// level 3
	for i := 0; i < 8192 && pt != ptypeDone; i++ { // index in the table
		ptr := (*uint64)(unsafe.Pointer(uintptr(k.ProcLevel3Phys) +
			(uintptr(i) * 8)))

		// make nil deref fail
		if i == 0 {
			// first page is left empty and the linker knows to
			// not use it
			logger.Debugf("first page, bad entry")
			*ptr = makeBadEntry()
			continue
		}
		// heap?
		if pt == heapSpace {
			// are we matching the page value exactly, or have we already started the heap
			if uint64(i) == k.ProcHeapLoc || fixedSizedHeap > 0 {
				*ptr = makePhysicalBlockEntry(uintptr(pageStart), MemoryNormal)
				fixedSizedHeap++
				pageStart += k.KernelPageSize
				if fixedSizedHeap == k.ProcHeapSize {
					pt++
				}
			} else {
				*ptr = makeBadEntry()
			}
			continue
		}
		// stack?
		if pt == stackSpace {
			// are we into the stack pages?
			if uintptr(i) > uintptr(k.ProcStackLoc-fixedSizedStack) {
				*ptr = makePhysicalBlockEntry(uintptr(pageStart), MemoryNormal)
				fixedSizedStack++
				pageStart += k.KernelPageSize
				if fixedSizedStack == k.ProcStackSize {
					pt++
				}
			} else {
				*ptr = makeBadEntry()
			}
			continue
		}

		// this is used when we are making the page mappings match the
		// binary we have already placed down in ram
		*ptr = makePhysicalBlockEntry(uintptr(pageStart), MemoryNormal)
		size := k.NameToSizeInPages[ptypeToSection[pt]]
		current++
		pageStart += k.KernelPageSize
		if current == int(size) {
			current = 0
			pt++
			switch pt {
			case readOnlyData:
				k.ROLowestVirt = k.ProcLinkVirt + (uint64(i) * k.KernelPageSize)
			case readWriteData:
				k.RWLowestVirt = k.ProcLinkVirt + (uint64(i) * k.KernelPageSize)
			}
		}
	}
}
func (k *KernelProcStartupInfo) injectBootloaderParams() {
	// setup the injected parameters
	bootloader.InjectedParams.HeapStart = k.ProcHeapVirt
	bootloader.InjectedParams.StackStart = k.ProcStackVirt - ((k.ProcStackSize - 1) * 0x10000)
	bootloader.InjectedParams.EntryPoint = k.EntryPoint
	bootloader.InjectedParams.KernelCodeStart = k.ProcLinkVirt
	bootloader.InjectedParams.UnixTime = 0
	bootloader.InjectedParams.ReadOnlyStart = k.ROLowestVirt
	bootloader.InjectedParams.ReadWriteStart = k.RWLowestVirt
	codeSize := uint8(k.NameToSizeInPages[".text"])
	bootloader.InjectedParams.SetKernelCodePages(codeSize)
	roSize := uint8(k.NameToSizeInPages[".rodata"])
	bootloader.InjectedParams.SetReadOnlyPages(roSize)
	rwSize := uint8(k.NameToSizeInPages[".data"])
	bootloader.InjectedParams.SetReadOnlyPages(rwSize)
	bssSize := uint8(k.NameToSizeInPages[".bss"])
	bootloader.InjectedParams.SetUnitializedPages(bssSize)
	bootloader.InjectedParams.SetHeapPages(uint8(k.ProcHeapSize))
	bootloader.InjectedParams.SetStackPages(uint8(k.ProcStackSize))

	logger.Debugf("HeapStart           :0x%016x", bootloader.InjectedParams.HeapStart)
	logger.Debugf("StackStart          :0x%016x", bootloader.InjectedParams.StackStart)
	logger.Debugf("KernelCodeStart     :0x%016x", bootloader.InjectedParams.KernelCodeStart)
	logger.Debugf("ReadOnlyStart       :0x%016x", bootloader.InjectedParams.ReadOnlyStart)
	logger.Debugf("ReadWriteStart      :0x%016x", bootloader.InjectedParams.ReadWriteStart)
	logger.Debugf("UnitializedStart    :0x%016x", bootloader.InjectedParams.ReadWriteStart)
	logger.Debugf("EntryPoint          :0x%016x", bootloader.InjectedParams.EntryPoint)
	logger.Debugf("EntryPoint          :0x%016x", bootloader.InjectedParams.EntryPoint)
	logger.Debugf("PagesHeap           :%02d", bootloader.InjectedParams.HeapPages())
	logger.Debugf("PagesStack          :%02d", bootloader.InjectedParams.StackPages())
	logger.Debugf("PagesKernel         :%02d", bootloader.InjectedParams.KernelCodePages())
	logger.Debugf("PagesUnitialized    :%02d", bootloader.InjectedParams.UnitializedPages())
	logger.Debugf("PagesRO             :%02d", bootloader.InjectedParams.ReadOnlyPages())
	logger.Debugf("PagesRW             :%02d", bootloader.InjectedParams.ReadWritePages())

	// tricky: we have to compute the physical address ourselves for this because the virtual
	// address is generated by the linker and doesn't know about our VM tables ...
	truePhys := virtToPhysInKernelProc(k.ProcLevel3Phys, k.SymbolToVirt[bootloaderParams])
	if err := enableMMUTablesOtherCore(k.MAIRVal, k.TCREL1Val, k.SCTRLEL1Val,
		k.TTBR0Val, k.ProcLevel2Phys, k.Processor); err != 0 {
		fmt.Errorf("failed to enable MMU tables for core %d", k.Processor)
		return
	}
	jumpToKernelProc(k.Processor, k.ProcLevel2Phys, k.EntryPoint,
		(uint64)(uintptr((unsafe.Pointer(&bootloader.InjectedParams)))),
		truePhys /*derived from symbol location of bootloader_params*/, 0)
	logger.Debugf("Bootloader deadlooping on proc 0")
	for {
		arm.Asm("nop")
	}
}