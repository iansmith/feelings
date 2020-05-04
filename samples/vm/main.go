package main

import (
	"feelings/src/golang/fmt"
	"feelings/src/hardware/videocore"
	"feelings/src/lib/trust"
	rt "feelings/src/tinygo_runtime"
	"unsafe"

	"github.com/tinygo-org/tinygo/src/device/arm"
)

//go:export raw_exception_handler
//go:noinline
func raw_exception_handler(t uint64, esr uint64, addr uint64) {
	trust.Fatalf(1, "raw_exception caught: which=%x, esr=%x, addr=%x", t, esr, addr)
	for i := 0; i < 1000000000; i++ {
		arm.Asm("nop")
	}
}

//go:extern _binary_font_psf_start
var binary_font_psf_start [0]byte

/* PC Screen Font as used by Linux Console */
type PCScreenFont struct {
	Magic         uint32
	Version       uint32
	Headersize    uint32
	Flags         uint32
	NumGlyphs     uint32
	BytesPerGlyph uint32
	Height        uint32
	Width         uint32
}

//these are indices into the MAIR register
const MemoryDeviceNoGatherNoReorderNoEarlyWriteAck = 0
const MemoryNoCache = 1
const MemoryNormal = 2

//these are values for the MAIR register
const MemoryDeviceNoGatherNoReorderNoEarlyWriteAckValue = 0x00 //it's hardware regs
const MemoryNoCacheValue = 0x44                                //not inner or outer cacheable
const MemoryNormalValue = 0xFF                                 //cache all you want, including using TLB

const UndocumentedTTBRCNP = 0x1

//export _enable_mmu_tables
func enableMMUTables(mairVal uint64, tcrVal uint64, sctrlVal uint64, ttbr0 uint64, ttbr1 uint64)

var logger *trust.Logger

func xxmain() {
	rt.MiniUART = rt.NewUART()
	_ = rt.MiniUART.Configure(rt.UARTConfig{ /*no interrupt*/ })
	trust.Errorf("hello world, UART initialized for logging")

	var size, base uint32

	//info := videocore.SetFramebufferRes1920x1200()
	//if info == nil {
	//	rt.Abort("giving up")
	//}
	info := videocore.SetFramebufferRes1024x768()
	if info == nil {
		rt.Abort("giving up")
	}

	logger = videocore.NewConsoleLogger()

	id, ok := videocore.BoardID()
	if ok == false {
		trust.Errorf("can't get board id\n")
		return
	}
	logger.Infof("board id         : %016x\n", id)

	v, ok := videocore.FirmwareVersion()
	if ok == false {
		trust.Errorf("can't get firmware version id\n")
		return
	}
	logger.Infof("firmware version : %08x\n", v)

	rev, ok := videocore.BoardRevision()
	if ok == false {
		trust.Errorf("can't get board revision id\n")
		return
	}
	logger.Infof("board revision   : %08x %s\n", rev, rt.BoardRevisionDecode(fmt.Sprintf("%x", rev)))

	cr, ok := videocore.GetClockRate()
	if ok == false {
		rt.MiniUART.WriteString("can't get clock rate\n")
		return
	}
	logger.Infof("clock rate       : %d hz\n", cr)

	base, size, ok = videocore.GetARMMemoryAndBase()
	if ok == false {
		rt.MiniUART.WriteString("can't get arm memory\n")
		return
	}
	logger.Infof("ARM Memory       : 0x%x bytes @ 0x%x\n", size, base)

	base, size, ok = videocore.GetVCMemoryAndBase()
	if ok == false {
		rt.MiniUART.WriteString("can't get vc memory\n")
		return
	}
	logger.Infof("VidCore IV Memory: 0x%x bytes @ 0x%x\n", size, base)

	//setup memory types and attributes
	MAIRVal := uint64(((MemoryDeviceNoGatherNoReorderNoEarlyWriteAckValue << (MemoryDeviceNoGatherNoReorderNoEarlyWriteAck * 8)) |
		(MemoryNoCacheValue << (MemoryNoCache * 8)) |
		(MemoryNormalValue << (MemoryNormal * 8))))

	// zero on these fields
	// TBI - no tag bits
	// IPS - 32 bit (4GB)
	// EPD1 - enable walks in kernel
	// EPD0 - enable walks in userspc
	//TCR REG https://developer.arm.com/docs/ddi0595/b/aarch64-system-registers/tcr_el1

	TCREL1Val := uint64(((0b11 << 30) | // granule size in kernel (values are different in kernel and user!)
		(0b11 << 28) | // inner shareable
		(0b01 << 26) | // write back (outer)
		(0b01 << 24) | // write back (inner)
		(20 << 16) | //20 is T1SZ, 42bit addr space (same as example https://developer.arm.com/docs/den0024/latest/the-memory-management-unit/translating-a-virtual-address-to-a-physical-address)
		(0b10 << 14) | // granule size in user (values are different in kernel in user!)
		(0b11 << 12) | //inner shareable
		(0b01 << 10) | //write back (outer)
		(0b01 << 8) | //write back (inner)
		(20 << 0))) //20 is T0SZ, 42 bit addr space

	// Undocumented TTBRCNP from BZT's tutorial....
	//TTBR0Val := uint64((0x100000 << 7) | UndocumentedTTBRCNP) //base addr 0x10_0000, no other shenanigans
	//TTBR1Val := uint64((0x100000 << 7) | UndocumentedTTBRCNP) //base addr 0x10_0000, no other shenanigans
	TTBR0Val := uint64(0x10_0000)
	TTBR1Val := uint64(0x10_0000)

	SCTRLEL1Val := uint64((0xC00800) | //mandatory reserved 1 bits
		(1 << 12) | // I Cache for both el1 and el0
		(1 << 4) | // SA0 stack alignment check in el0
		(1 << 3) | // SA stack alignment check in el1
		(1 << 2) | //  D Cache for both el1 and el0
		(1 << 1) | //  Alignment check enable
		(1 << 0)) // MMU ENABLED!! THE BIG DOG

	//we don't use level 1 because we have 42 bit address and 64k granules
	sizeOfLevel2 := uintptr(0x10000) //8192 entries (only 4 in use, the lowest 4)
	sizeOfLevel3 := uintptr(0x10000) //8192 entries

	//level 2 table has 8192 entries selected by bits 41:29 but we are only interested in bits
	//29 and 30 because the rest are always the same... so we need to point to 4 different level
	//3 tables in the lowest positions
	root := uintptr(0x100000)
	logger.Infof("Setting up level 2 table (8192 entries) @ 0x%08x", root)
	level2Ptr := root
	level3PtrsBase := root + sizeOfLevel2
	for i := uintptr(0); i < 8192; i++ { //fill in entries to cause a fault
		//entries are bad other than first 4
		asUint64 := (*uint64)(unsafe.Pointer(level2Ptr + (i * 8))) //8 bytes entry
		if i < 4 {                                                 //4 entries means we have 4x32bits or 4x4GB or 64GB addr space
			target := level3PtrsBase + (i * sizeOfLevel3)
			*asUint64 = makeTableEntry(target)
			logger.Infof("level 2, entry %d is 0x%08x", i, *asUint64)
			continue
		}
		*asUint64 = makeBadEntry()
	}

	for i := uintptr(0); i < 4; i++ {
		level3Ptr := level3PtrsBase + (i * sizeOfLevel3)
		logger.Infof("Setting up level 3 table %d (8192 entries) @ 0x%08x", i, level3Ptr)
		for j := uintptr(0); j < 8192; j++ { //12 bits
			asUint64 := ((*uint64)(unsafe.Pointer(level3Ptr + (j * 8)))) //8 bytes entry
			target := (i << 29) | (j << 16)                              //j only has 12 bits, so these don't overlap
			memType := uint64(MemoryNormal)
			if i == 1 && j >= 0x1C00 && j < 0x1F00 { //framebuffer
				memType = MemoryNoCache
			}
			if i == 1 && j >= 0x1F00 { //peripheral space
				memType = MemoryDeviceNoGatherNoReorderNoEarlyWriteAck
			}
			if i == 2 && j < 32 { //mailboxes
				memType = MemoryDeviceNoGatherNoReorderNoEarlyWriteAck
			}
			if (i == 2 && j >= 32) || (i == 3) {
				*asUint64 = makeBadEntry()
				continue
			}
			*asUint64 = makePhysicalBlockEntry(target, memType)
		}
	}

	//go live!
	logger.Infof("going live!")
	logger.Debugf("self test zero                   0x0 = 0x%16x", selfTest(0))
	logger.Debugf("self test no page             0xffff = 0x%16x", selfTest(0xFFFF))
	logger.Debugf("self test 1st page           0x10000 = 0x%16x", selfTest(0x10000))
	logger.Debugf("self test end 1st page       0x1ffff = 0x%16x", selfTest(0x1ffff))
	logger.Debugf("self test boot load addr     0x80820 = 0x%16x", selfTest(0x80820))
	logger.Debugf("self test start of tables   0x100000 = 0x%16x", selfTest(0x100000))
	logger.Debugf("self test middle of tables  0x1108C0 = 0x%16x", selfTest(0x1108C0))
	logger.Debugf("self test base load addr    0x300000 = 0x%16x", selfTest(0x300000))
	logger.Debugf("self test start of vc4ram 0x3C000000 = 0x%16x", selfTest(0x3C000000))
	logger.Debugf("self test end of vc4ram   0x3EFFFFFF = 0x%16x", selfTest(0x3EFFFFFF))
	logger.Debugf("self test start of perpih 0x3F000000 = 0x%16x", selfTest(0x3F000000))
	logger.Debugf("self test end of periph   0x3FFFFFFF = 0x%16x", selfTest(0x3FFFFFFF))
	logger.Debugf("self test start of mbox   0x40000000 = 0x%16x", selfTest(0x40000000))
	logger.Debugf("self test end of mbox     0x401FFFFF = 0x%16x", selfTest(0x401FFFFF))
	logger.Debugf("self test TTBR0                      = 0x%16x", selfTest(uintptr(TTBR0Val)))
	//for {
	//	arm.Asm("nop")
	//}
	enableMMUTables(MAIRVal, TCREL1Val, SCTRLEL1Val, TTBR0Val, TTBR1Val)

	//we are live!
	logger.Infof("we are done!")
}

// this mask is to differentiate TTBR1 from TTBR0--TTBR1 is all 1s, TTBR0 is all 0s
const leftPart = 0xffff_fc00_0000_0000

// bits 41:29 are the index area of VA in L2
const l2IndexArea = 0x3FF_E000_0000

// bits 28:16 are the index area of VA in L3
const l3IndexArea = 0x1FFF_0000

// bits 15:0 are the index area into real block
const physIndexArea = 0xFFFF

// bits 47:16 are the PTR area of L3 and L2
const ptrArea = 0xFFFF_FFFF_0000

func selfTest(va uintptr) uint64 {
	//logger.Debugf("entering self test with va %x", va)
	ttbr0 := uintptr(0x10_0000)       //its user space
	index := (va & l2IndexArea) >> 29 //bits 41:29
	asUint64 := ((*uint64)(unsafe.Pointer(ttbr0 + (index * 8))))
	level3Ptr := uintptr((*asUint64)) & ptrArea //points to BASE of next table
	index = (va & l3IndexArea) >> 16            //pull out the index in this table
	asUint64 = ((*uint64)(unsafe.Pointer(level3Ptr + (index * 8))))
	noLower16 := ^uint64(0xffff)
	maskedEntry := (*asUint64 & noLower16)
	//logger.Debugf("table ptr %x table value %x (masked %x) and lower 16 %x\n", asUint64, *asUint64, maskedEntry, va&physIndexArea)
	return maskedEntry | uint64(va&physIndexArea) //could do + here but these don't overlap
}

const noLast16 = 0xffff_ffff_ffff_0000

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

var ok = false

// mair index == MemoryNormal
// mair index == MemoryNoCache for video ram
// mair index == MemoryDeviceNoGatherNoReorderNoEarlyWriteAckValue for device memory
// destination is top 32 bits of final memory location
func makePhysicalBlockEntry(destination uintptr, mairIndex uint64) uint64 {
	result := uint64(
		uint64(destination&noLast16) | //address of the _BASE_ of next table, it's just a mask because its >>16 and <<16
			uint64(1<<10) | //AF, not used for us
			uint64(3<<8) | //inner
			uint64(mairIndex<<2) | uint64(1<<0)) //last two bits are 0b01 to indicate block entry
	return result
}

func main() {
	rt.MiniUART = rt.NewUART()
	_ = rt.MiniUART.Configure(rt.UARTConfig{ /*no interrupt*/ })
	trust.Errorf("hello world, UART initialized for logging")

}
