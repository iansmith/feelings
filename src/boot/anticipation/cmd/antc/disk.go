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

var sectionBuffer [sectionBufferSize]byte

func canBootFromDisk(logger *trust.Logger) (emmc.EmmcFile, bool) {
	var err error
	var fp emmc.EmmcFile
	if emmc.Impl.Init() != emmc.EmmcOk {
		logger.Errorf("Unable to initialize EMMC driver, " +
			"booting from serial port...")
		return nil, false
	}
	fp, err = emmc.Impl.Open(maddie)
	if err != nil {
		logger.Errorf("Unable to find %s binary, "+
			"booting from serial port...", maddie)
		return nil, false

	}
	logger.Infof("found bootable lady: %s", maddie)
	return fp, true
}

// This does the work of loading the binary in pieces and then attaching
// it to pages.
func bootDisk(fp emmc.EmmcFile, logger *trust.Logger) {
	elfFile, err := elf.NewFile(fp)
	if err != nil {
		trust.Debugf("Error attaching elf reader: %v", err)
		return
	}
	logger.Debugf("entry point: %016x", elfFile.FileHeader.Entry)

	totalExcText := uint64(0)

	nameToSizeInPages := make(map[string]uint64)
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
				nameToSizeInPages[sect.Name] = sizeInPages
			}
		}
	}
	overhang := uint64(1)
	if totalExcText%KernelPageSize == 0 {
		overhang = 0
	}
	nameToSizeInPages[".text"] = totalExcText/KernelPageSize + overhang
	totalCodePages := 0
	for _, v := range nameToSizeInPages {
		totalCodePages += int(v)
	}

	currentPhys := uintptr(bootloader.MadeleinePlacement)
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
				return
			}
			for i := 0; i < n; i++ {
				ptr := (*byte)(unsafe.Pointer(currentPhys + uintptr(i)))
				*ptr = sectionBuffer[i]
			}
			currentPhys += uintptr(n)
			read += uint64(n)
		}
		if sectName != ".exc" { //glom the .exc and the .text together
			if currentPhys%KernelPageSize != 0 {
				diff := KernelPageSize - (currentPhys % KernelPageSize)
				currentPhys += uintptr(diff)
			}
		}
	}
	symbols, err := elfFile.Symbols()
	if err != nil {
		trust.Errorf("unable access symbol table: %v", err)
		return
	}
	bp := uintptr(0)
	for _, s := range symbols {
		if s.Name == "bootloader_params" {
			bp = uintptr(s.Value)
		}
	}
	if bp == 0 {
		trust.Errorf("could not find the bootloader_params inside binary")
	}
	buildMaddyTables(nameToSizeInPages, elfFile.FileHeader.Entry, uint64(bp))
	for {
		arm.Asm("nop")
	}
}

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

const kernelProcHeapFixedSize = 0x4  //4mb, this is in pages
const kernelProcStackFixedSize = 0x2 //128kb

func buildMaddyTables(nameToSizeInPages map[string]uint64, entryPoint uint64,
	paramsPtr uint64) {
	//there are 8192 entries on this table, but we only care about 1 of
	//them, which is the 0 entry.. for bits
	// maddy is linked at KernelProcessLinkAddr which is the same as
	// all the other kernel processes

	// level 2
	for i := 0; i < 8192; i++ {
		ptr := (*uint64)(unsafe.Pointer(uintptr(bootloader.MadelineTableLevel2) +
			(uintptr(i) * 8)))
		if i == 0 {
			*ptr = makeTableEntry(bootloader.MadelineTableLevel3Start)
			logger.Debugf("make table entry %016x -> %016x", ptr,
				makeTableEntry(bootloader.MadelineTableLevel3Start))
		} else {
			*ptr = makeBadEntry() //not a valid entry, so it will crap out with page fault
		}
	}

	pt := executableCode
	current := 0
	pageStart := bootloader.MadeleinePlacement //page we pointing TO

	fixedSizedHeap := 0
	fixedSizedStack := 0

	roLowestAddr := uint64(0)
	rwLowestAddr := uint64(0)

	//level 3
	for i := 0; i < 8192 && pt != ptypeDone; i++ { //index in the table
		ptr := (*uint64)(unsafe.Pointer(uintptr(bootloader.MadelineTableLevel3Start) +
			(uintptr(i) * 8)))

		//make nil deref fail
		if i == 0 {
			//first page is left empty and the linker knows to
			//not use it
			logger.Debugf("first page, bad entry")
			*ptr = makeBadEntry()
			continue
		}
		//heap?
		if pt == heapSpace {
			if uintptr(i) == bootloader.KernelProcessHeap || fixedSizedHeap > 0 {
				*ptr = makePhysicalBlockEntry(uintptr(pageStart), MemoryNormal)
				logger.Debugf("made HEAP entry %d: %x -> %x, (%d)",
					i, ptr, pageStart, pt)
				fixedSizedHeap++
				pageStart += KernelPageSize
				if fixedSizedHeap == kernelProcHeapFixedSize {
					pt++
				}
			} else {
				*ptr = makeBadEntry()
			}
			continue
		}
		//stack?
		if pt == stackSpace {
			if uintptr(i) > uintptr(bootloader.KernelProcessStack-fixedSizedStack) {
				*ptr = makePhysicalBlockEntry(uintptr(pageStart), MemoryNormal)
				if fixedSizedStack == 0 {
					logger.Debugf("mapping STACK entry %d: %x->%x", i, ptr, pageStart)
				}
				fixedSizedStack++
				pageStart += KernelPageSize
				if fixedSizedStack == kernelProcStackFixedSize {
					pt++
				}
			} else {
				*ptr = makeBadEntry()
			}
			continue
		}

		//this is used when we are making the page mappings match the
		//binary we have already placed down in ram
		*ptr = makePhysicalBlockEntry(uintptr(pageStart), MemoryNormal)
		logger.Debugf("made block entry %d: %x -> %x, (%d)",
			i, ptr, pageStart, pt)
		size := nameToSizeInPages[ptypeToSection[pt]]
		current++
		pageStart += KernelPageSize
		if current == int(size) {
			current = 0
			pt++
			switch pt {
			case readOnlyData:
				roLowestAddr = bootloader.KernelProcessLinkAddr + (uint64(i) * KernelPageSize)
			case readWriteData:
				rwLowestAddr = bootloader.KernelProcessLinkAddr + (uint64(i) * KernelPageSize)
			}
		}
	}
	// setup the injected parameters
	bootloader.InjectedParams.HeapStart = bootloader.KernelProcessHeapAddr
	bootloader.InjectedParams.StackStart = bootloader.KernelProcessStackAddr - uint64((fixedSizedStack-1)*0x10000)
	bootloader.InjectedParams.EntryPoint = entryPoint
	bootloader.InjectedParams.KernelCodeStart = bootloader.KernelProcessLinkAddr
	bootloader.InjectedParams.UnixTime = 0
	bootloader.InjectedParams.ReadOnlyStart = roLowestAddr
	bootloader.InjectedParams.ReadWriteStart = rwLowestAddr
	codeSize := uint8(nameToSizeInPages[".text"])
	bootloader.InjectedParams.SetKernelCodePages(codeSize)
	roSize := uint8(nameToSizeInPages[".rodata"])
	bootloader.InjectedParams.SetReadOnlyPages(roSize)
	rwSize := uint8(nameToSizeInPages[".data"])
	bootloader.InjectedParams.SetReadOnlyPages(rwSize)
	bssSize := uint8(nameToSizeInPages[".bss"])
	bootloader.InjectedParams.SetUnitializedPages(bssSize)
	bootloader.InjectedParams.SetHeapPages(kernelProcHeapFixedSize)
	bootloader.InjectedParams.SetStackPages(kernelProcStackFixedSize)

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
	// address is generated by the linker and doesn't know about our VM tables ... so the
	// address given as params pointer is the "naive" address and we have to use our knowlege
	// of the VM tables to figure where that address is going to point to when the code is
	// actually running
	trueAddr := virtToPhysInKernelProc(bootloader.MadelineTableLevel3Start, paramsPtr)
	if err := enableMMUTablesOtherCore(MAIRVal, TCREL1Val, SCTRLEL1Val, TTBR0Val, bootloader.MadelineTableLevel2,
		bootloader.MadeleineProcessor); err != 0 {
		fmt.Errorf("failed to enable MMU tables for core %d", bootloader.MadeleineProcessor)
		return
	}
	jumpToKernelProc(bootloader.MadeleineProcessor, bootloader.MadelineTableLevel2,
		entryPoint, (uint64)(uintptr((unsafe.Pointer(&bootloader.InjectedParams)))),
		trueAddr /*derived from paramsPtr*/, 0)
	logger.Debugf("Bootloader deadloopin on proc 0")
	for {
		arm.Asm("nop")
	}
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
