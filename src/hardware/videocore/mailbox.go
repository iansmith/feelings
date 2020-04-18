package videocore

import (
	"feelings/src/hardware/rpi"
	"feelings/src/joy/semihosting"
	"unsafe"

	"github.com/tinygo-org/tinygo/src/device/arm"

	"github.com/tinygo-org/tinygo/src/runtime/volatile"
)

var Mailbox *MailboxRegisterMap = (*MailboxRegisterMap)(unsafe.Pointer(rpi.MemoryMappedIO + 0x0000B880))

const MailboxFull = 0x80000000
const MailboxEmpty = 0x40000000
const MailboxResponse = 0x80000000
const MailboxRequest = 0x0

/* channels */
const MailboxChannelPower = 0
const MailboxChannelFramebuffer = 1
const MailboxChannelVUArt = 2
const MailboxChannelVCHIQ = 3
const MailboxChannelLEDs = 4
const MailboxChannelButtons = 5
const MailboxChannelTouch = 6
const MailboxChannelCount = 7
const MailboxChannelProperties = 8

/*tags*/
const MailboxTagSerial = 0x100004
const MailboxTagFirmwareVersion = 0x1
const MailboxTagBoardModel = 0x00010001
const MailboxTagBoardRevision = 0x00010002
const MailboxTagMACAddress = 0x00010003
const MailboxTagLast = 0x0

type MailboxRegisterMap struct {
	Read     volatile.Register32    //0x00
	reserved [3]volatile.Register32 //0x04-0x10
	Poll     volatile.Register32    //0x10
	Sender   volatile.Register32    // 0x14
	Status   volatile.Register32    // 0x18
	Config   volatile.Register32    //0x1c
	Write    volatile.Register32    //0x20
}

func Call(ch uint8, mboxBuffer *uint32) bool {
	mask := uintptr(^uint64(0xf))
	rawPtr := uintptr(unsafe.Pointer(mboxBuffer))
	if rawPtr&0xf != 0 {
		semihosting.Exit(7)
	}
	addrWithChannel := (uintptr(unsafe.Pointer(rawPtr)) & mask) | uintptr(ch&0xf)
	for {
		if Mailbox.Status.HasBits(MailboxFull) {
			arm.Asm("nop")
		} else {
			break
		}
	}
	Mailbox.Write.Set(uint32(addrWithChannel))
	//for i := 0; i < 20; i++ {
	//	joy.Console.Logf("%x,%x\n	", rawPtr, addrWithChannel)
	//}
	//joy.Console.Logf("wasting time so the mailbox won't feel in a hurry...")
	for {
		if Mailbox.Status.HasBits(MailboxEmpty) {
			arm.Asm("nop")
		} else {
			read := Mailbox.Read.Get()
			if read == uint32(addrWithChannel) {
				//joy.Console.Logf("mailbox response\n")
				//for i := 0; i < len(mboxBuffer); i++ {
				//	joy.Console.Logf("%d %04x\n", i, mboxBuffer[i].Get())
				//}
				//did we get a confirm? we have to use volatile here because we
				//could not guarantee alignment if we used volatile.Register32()
				return volatile.LoadUint32(slotOffset((*uint32)(unsafe.Pointer(rawPtr)), 1)) == MailboxResponse
			}
		}
	}
	return false //how would this happen?

}

func BoardID() (uint64, bool) {
	return boardInfoNoParams(MailboxTagSerial, 2)
}

func FirmwareVersion() (uint32, bool) {
	firmware, ok := boardInfoNoParams(MailboxTagFirmwareVersion, 1)
	if !ok {
		return 0x872720, ok
	}
	return uint32(firmware), true
}

func BoardModel() (uint32, bool) {
	model, ok := boardInfoNoParams(MailboxTagBoardModel, 1)
	if !ok {
		return 0x872728, ok
	}
	return uint32(model), true
}

func BoardRevision() (uint32, bool) {
	revision, ok := boardInfoNoParams(MailboxTagBoardRevision, 1)
	if !ok {
		return 0x872727, ok
	}
	return uint32(revision), true
}

func MACAddress() (uint64, bool) {
	addr, ok := boardInfoNoParams(MailboxTagMACAddress, 2)
	if !ok {
		return 0xab127348, false
	}
	addr &= 0x0000ffffffffffffffff
	return addr, true
}

func boardInfoNoParams(tag uint32, responseSlots int) (uint64, bool) {
	buffer := message(0, tag, 2)
	ok := Call(MailboxChannelProperties, buffer)
	if !ok {
		return 77281, false //strange constant so it is easy to find
	}
	if responseSlots == 1 {
		return uint64(*slotOffset(buffer, 5)), true
	}
	if responseSlots == 2 {
		upper := uint64((*slotOffset(buffer, 6) << 32))
		lower := uint64((*slotOffset(buffer, 5)))
		return upper + lower, true
	}
	panic("too many response slots")
}

// fill out the message fields for a request but does NOT fill in the
// request fields, so can only be used by itself for requests with no args
func message(requestSlots int, tag uint32, responseSlots int) *uint32 {
	totalSlots := 1 + 1 + 1 + requestSlots + 1 + 1 + responseSlots + 1
	ptr := sixteenByteAlignedPointer(uintptr(totalSlots << 2)) //32 bit slots
	ptr32 := ((*uint32)(unsafe.Pointer(ptr)))
	//irritating that we cannot use Register32() here
	*slotOffset(ptr32, 0) = uint32(totalSlots * 4) //num bytes
	*slotOffset(ptr32, 1) = MailboxRequest
	*slotOffset(ptr32, 2) = tag
	for i := 3; i < 3+int(requestSlots); i++ {
		*slotOffset(ptr32, i) = 0
	}
	*slotOffset(ptr32, 3+requestSlots) = uint32(responseSlots * 4) //buffer size
	*slotOffset(ptr32, 4+requestSlots) = 0                         //its a request
	for j := 4 + responseSlots + 1; j < 4+responseSlots+requestSlots+1; j++ {
		*slotOffset(ptr32, j) = 0
	}
	*slotOffset(ptr32, 4+responseSlots+requestSlots+1) = MailboxTagLast //end
	return ptr32
}
func slotOffset(ptr32 *uint32, slot int) *uint32 {
	newptr := uintptr(unsafe.Pointer(ptr32)) + uintptr(4*slot)
	return (*uint32)(unsafe.Pointer(newptr))
}

//pass in the number of bytes you want to be aligned to 16byte boundary
//the default allocator only ollocates things at their "natural" sizes
func sixteenByteAlignedPointer(size uintptr) *uint64 {
	units := (((size / 16) + 1) * 16) / 8
	//joy.Console.Logf("%x,%x", size, units)
	bigger := make([]uint64, units)
	hackFor16ByteAlignment := ((*uint64)(unsafe.Pointer(&bigger[0])))
	ptr := uintptr(unsafe.Pointer(hackFor16ByteAlignment))
	if ptr&0xf != 0 {
		diff := uintptr(16 - (ptr & 0xf))
		hackFor16ByteAlignment = ((*uint64)(unsafe.Pointer(ptr + diff)))
		//joy.Console.Logf("alignment hack: %8x,%x,%x", ptr, diff, uintptr(unsafe.Pointer(hackFor16ByteAlignment)))
	}
	return hackFor16ByteAlignment
}
