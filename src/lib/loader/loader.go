package loader

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

//these are indices into the MAIR register
const MemoryDeviceNoGatherNoReorderNoEarlyWriteAck = 0
const MemoryNoCache = 1
const MemoryNormal = 2

//these are values for the MAIR register
const MemoryDeviceNoGatherNoReorderNoEarlyWriteAckValue = 0x00 //it's hardware regs
const MemoryNoCacheValue = 0x44                                //not inner or outer cacheable
const MemoryNormalValue = 0xFF                                 //cache all you want, including using TLB

//setup memory types and attributes
var MAIRVal = uint64(((MemoryDeviceNoGatherNoReorderNoEarlyWriteAckValue << (MemoryDeviceNoGatherNoReorderNoEarlyWriteAck * 8)) |
	(MemoryNoCacheValue << (MemoryNoCache * 8)) |
	(MemoryNormalValue << (MemoryNormal * 8))))

// zero on these fields
// TBI - no tag bits
// IPS - 32 bit (4GB)
// EPD1 - enable walks in kernel
// EPD0 - enable walks in userspc
//TCR REG https://developer.arm.com/docs/ddi0595/b/aarch64-system-registers/tcr_el1
var TCREL1Val = uint64(((0b11 << 30) | // granule size in kernel
	(0b11 << 28) | // inner shareable
	(0b01 << 26) | // write back (outer)
	(0b01 << 24) | // write back (inner)
	(22 << 16) | //22 is T1SZ, 42bit addr space (same as example https://developer.arm.com/docs/den0024/latest/the-memory-management-unit/translating-a-virtual-address-to-a-physical-address)
	(0b1 << 14) | // granule size in user
	(0b11 << 12) | //inner shareable
	(0b01 << 10) | //write back (outer)
	(0b01 << 8) | //write back (inner)
	(22 << 0))) //22 is T0SZ, 42 bit addr space

var SCTRLEL1Val = uint64((0xC00800) | //mandatory reserved 1 bits
	(1 << 12) | // I Cache for both el1 and el0
	(1 << 4) | // SA0 stack alignment check in el0
	(1 << 3) | // SA stack alignment check in el1
	(1 << 2) | //  D Cache for both el1 and el0
	(1 << 1) | //  Alignment check enable
	(1 << 0)) // MMU ENABLED!! THE BIG DOG

const TTBR0Val = uint64(0x10000) //this is where we START our page tables, must be 64K aligned

//drop bottom 64k
const noLast16 = 0xffffffffffff0000

//as we read blobs from disk, they are stored here
var sectionBuffer [sectionBufferSize]byte

//export _enable_mmu_tables
func enableMMUTables(mairVal uint64, tcrVal uint64, sctrlVal uint64, ttbr0 uint64, ttbr1 uint64)

//export _enable_mmu_tables_other_core
func enableMMUTablesOtherCore(mairVal uint64, tcrVal uint64, sctrlVal uint64, ttbr0 uint64, ttbr1 uint64, core uint64) int

//export jump_to_kernel_proc
func jumpToKernelProc(procId uint64, ttbr1 uint64, entryPoint uint64, paramPtrSource uint64, paramPtrDest uint64, _ uint64)

//ptypes are used to iterate through the various parts of setting up the VM tables
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

//
// var maddyParams = KernelProcStartupInfo{
// 	Filename:              maddie,
// 	KernelPageSize:        KernelPageSize,
// 	Processor:             bootloader.MadeleineProcessor,
// 	ProcCodePlacementPhys: bootloader.MadeleinePlacement,
// 	ProcLevel2Phys:        bootloader.MadelineTableLevel2,
// 	ProcLevel3Phys:        bootloader.MadelineTableLevel3Start,
// 	ProcHeapLoc:           bootloader.KernelProcessHeap,
// 	ProcHeapVirt:          bootloader.KernelProcessHeapAddr,
// 	ProcHeapSize:          fixedSizeOfHeap,
// 	ProcStackLoc:          bootloader.KernelProcessStack,
// 	ProcStackVirt:         bootloader.KernelProcessStackAddr,
// 	ProcLinkVirt:          bootloader.KernelProcessLinkAddr,
// 	ProcStackSize:         fixedSizeOfStack,
// 	MAIRVal:               MAIRVal,
// 	SCTRLEL1Val:           SCTRLEL1Val,
// 	TCREL1Val:             TCREL1Val,
// 	TTBR0Val:              TTBR0Val,
// 	SymbolToVirt:          map[string]uint64{bootloaderParams: 0},
// 	NameToSizeInPages:     nil, //return values overwrite
// 	EntryPoint:            0,   //return value should overwrite 0
// }

func NewKernelProcStartupInfo(filename string, processor uint64 /*0-3*/) *KernelProcStartupInfo {
	return &KernelProcStartupInfo{
		Filename:              filename,
		KernelPageSize:        KernelPageSize,
		Processor:             processor,
		ProcCodePlacementPhys: bootloader.MadeleinePlacement, //xxx only hardcoded value
		ProcLevel2Phys:        procToLevel2Phys(processor),
		ProcLevel3Phys:        procToLevel3Phys(processor),
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

func procToLevel2Phys(proc uint64 /*0-3*/) uint64 {
	return 0x3000_0000 + (proc * 0x2_0000)
}
func procToLevel3Phys(proc uint64 /*0-3*/) uint64 {
	return procToLevel2Phys(proc) + 0x1_0000
}

//go:noinline
func virtToPhysInKernelProc(_ *trust.Logger, table3Start uint64, vaddr uint64) uint64 {
	pagePtr := vaddr & (0x0000_0000_ffff_0000) //get rid of kernel prefix and index bits
	pagePtr = pagePtr >> 16                    //shift to make it an index into table level 3
	offsetInTable := pagePtr * 8               //64 bits per element
	index := (*uint64)((unsafe.Pointer)(uintptr(table3Start) + uintptr(offsetInTable)))
	truePage := (*index) & noLast16 // cruft in the last few bits for VM use only
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

func (k *KernelProcStartupInfo) KernelProcBootFromDisk(logger *trust.Logger) LoaderError {
	fp, err := emmc.Impl.Open(k.Filename)
	if err != nil {
		trust.Errorf("Unable to find %s binary, "+
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
	logger.Debugf("here1")
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
		trust.Debugf("loading %-10s @ 0x%x", sectName, currentPhys)
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
				trust.Errorf("Unable to read section %s: %v", sectName, err)
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
	logger.Debugf("here3")
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

	k.buildVMTransTables(logger)
	k.injectBootloaderParams(logger)

	for {
		arm.Asm("nop")
	}

}

func (k *KernelProcStartupInfo) buildVMTransTables(logger *trust.Logger) {
	// there are 8192 entries on this table, but we only care about 1 of
	// them, which is the 0 entry..

	// level 2
	for i := 0; i < 8192; i++ {
		ptr := (*uint64)(unsafe.Pointer(uintptr(k.ProcLevel2Phys) +
			(uintptr(i) * 8)))
		if i == 0 {
			*ptr = makeTableEntry(uintptr(k.ProcLevel3Phys))
			logger.Debugf("ptr is %x, value is %x, and %x,%x", ptr, *ptr,
				k.ProcLevel3Phys, procToLevel3Phys(3))
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
			// first page is left empty and the linker knows to not use it
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
func (k *KernelProcStartupInfo) injectBootloaderParams(logger *trust.Logger) {
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

	// tricky: we have to compute the physical address ourselves for this because the virtual
	// address is generated by the linker and doesn't know about our VM tables ...
	truePhys := virtToPhysInKernelProc(logger, k.ProcLevel3Phys, k.SymbolToVirt[bootloaderParams])
	if err := enableMMUTablesOtherCore(k.MAIRVal, k.TCREL1Val, k.SCTRLEL1Val,
		k.TTBR0Val, k.ProcLevel2Phys, k.Processor); err != 0 {
		fmt.Errorf("failed to enable MMU tables for core %d", k.Processor)
		return
	}
	jumpToKernelProc(k.Processor, k.ProcLevel2Phys, k.EntryPoint,
		(uint64)(uintptr((unsafe.Pointer(&bootloader.InjectedParams)))),
		truePhys /*derived from symbol location of bootloader_params*/, 0)
	trust.Debugf("Bootloader deadlooping on proc 0")
	for {
		arm.Asm("nop")
	}
}

func makeTableEntry(destination uintptr) uint64 {

	//low 2 bits as 0b11
	//bits 5:2 are index into mair table (4 bits to ref which one)
	//bits 7:6 el0 no read, no write (no read 0b01, no write 0b10)
	//bits 9:8 inner and outer shareable (inner 0b11, outer 0b10)
	//bit 10 AF flag, lets OS know if page is first accessed if 0
	//bits 47:16 address (32 bits, because we are 64K granules)
	result := uint64(
		uint64(1<<63) | //secure state?!? why is it needed?
			uint64(destination&noLast16) | //address of the _BASE_ of next table, just a mask because >>16 and then <<16
			uint64(3<<0)) //last two bits indicate page tbl
	return result
}

func makeBadEntry() uint64 {
	//*ANY* entry in a page table must have the last bit high, so 0 is always bad (easy!)
	return 0
}

// mair index == MemoryNormal
// mair index == MemoryNoCache for video ram
// mair index == MemoryDeviceNoGatherNoReorderNoEarlyWriteAckValue for device memory
// destination is top 32 bits of final memory location
func makePhysicalBlockEntry(destination uintptr, mairIndex uint64) uint64 {

	//https://armv8-ref.codingbelief.com/en/chapter_d4/d44_1_memory_access_control.html
	//we have SCTLR_EL1.WXN =0
	//we have AP set to 0b00
	//we have UXN set to 0
	//we have PXN set to 0
	//this implies: RWX from EL1, X only from EL0

	result := uint64(
		uint64(destination&noLast16) | //address of the PAGE, without low order
			//no use of bit 11, this controls if access in per process id
			(0b1 << 10) | //we are not yet using the ACCESS flag
			(0b1 << 5) | //non-secure
			(mairIndex&0x7)<<2 | // index in the MAIR register
			uint64(0b11<<0)) //last two bits are 0b11 to indicate block entry
	//note: last two bits are 0b01 *IF* you are not at level 3 of translation
	//note: at level3, there is this special encoding and we are at level 3
	//https://armv8-ref.codingbelief.com/en/chapter_d4/d43_2_armv8_translation_table_level_3_descriptor_formats.html
	return result
}
