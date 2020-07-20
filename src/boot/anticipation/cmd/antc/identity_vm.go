package main

import "unsafe"

// xxx prob should not have anything in lowest 16 bits
const kernelBase = uintptr(0x3000_104C | isKernelAddrMask)

//
// setup the virtual memory to be identity mapped
// physical == virtual
//

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

func setupVM() {

	//we don't use level 1 because we have 42 bit address and 64k granules
	sizeOfLevel2 := uintptr(0x1_0000) //8192 entries (only 4 in use, the lowest 4)
	sizeOfLevel3 := uintptr(0x1_0000) //8192 entries

	//level 2 table has 8192 entries selected by bits 41:29 but we are only interested in bits
	//29 and 30 because the rest are always the same... so we need to point to 4 different level
	//3 tables in the lowest positions
	root := uintptr(0x1_0000)
	logger.Infof("=== Bringing up the MMU and virtual memory === ")

	logger.Infof("Setting up level 2 table (8192 entries) @ 0x%016x\n", root)
	level2Ptr := root
	level3PtrsBase := root + sizeOfLevel2
	logger.Infof("Level 3 tables begin at 0x%016x", level3PtrsBase)
	for i := uintptr(0); i < 8192; i++ { //fill in entries to cause a fault
		//entries are bad other than first 4
		asUint64 := (*uint64)(unsafe.Pointer(level2Ptr + (i * 8))) //8 bytes entry
		if i < 4 {                                                 //4 entries means we have 4x32bits or 4x4GB or 64GB addr space
			target := level3PtrsBase + (i * sizeOfLevel3)
			*asUint64 = makeTableEntry(target)
			logger.Infof("level 2, entry %d (@ 0x%016x) is 0x%016x (0x%016x)", i, asUint64, target, *asUint64)
			continue
		}
		*asUint64 = makeBadEntry()
	}

	for i := uintptr(0); i < 4; i++ {
		level3Ptr := level3PtrsBase + (i * sizeOfLevel3)
		logger.Infof("Setting up level 3 table %d (8192 entries) @ 0x%016x\n", i, level3Ptr)
		for j := uintptr(0); j < 8192; j++ { //12 bits
			asUint64 := ((*uint64)(unsafe.Pointer(level3Ptr + (j * 8)))) //8 bytes entry
			target := (i << 29) | (j << 16)                              //j only has 12 bits, so these don't overlap
			memType := uint64(MemoryNormal)
			if target >= 0x3C00_0000 && target < 0x3F00_0000 { //framebuffer
				memType = MemoryNoCache
			}
			if target >= 0x3F00_0000 && target < 0x4_0000_0000 { //peripherals
				memType = MemoryDeviceNoGatherNoReorderNoEarlyWriteAck
			}
			if target >= 0x4_0000_0000 && target < 0x4_0000_0100 { //local peripherals, qa7
				memType = MemoryDeviceNoGatherNoReorderNoEarlyWriteAck
			}
			if target >= 0x4_0000_0100 { // bad mojo
				*asUint64 = makeBadEntry()
				continue
			}
			//make the physical block entry
			*asUint64 = makePhysicalBlockEntry(target, memType)
		}
	}

	enableMMUTables(MAIRVal, TCREL1Val, SCTRLEL1Val, TTBR0Val, TTBR1Val)
	logger.Infof("=== MMUenabled ===")
	// ptr := ((*uint64)(unsafe.Pointer(uintptr(kernelBase))))
	// *ptr = 0x0123456776543210
	// logger.Debugf("Self test write completed to kernel space...")
	// x := ((*uint64)(unsafe.Pointer(kernelBase)))
	// logger.Debugf("Self test readback from kernel space 0x%016x\n", *x)
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
