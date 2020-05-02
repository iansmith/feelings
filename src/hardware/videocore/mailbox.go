package videocore

import (
	"feelings/src/hardware/rpi"
	"feelings/src/lib/semihosting"
	"feelings/src/lib/trust"
	"fmt"
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
const MailboxTagGetClockRate = 0x00030002
const MailboxTagLast = 0x0
const MailboxTagGetVCMemory = 0x00010006
const MailboxTagGetARMMemory = 0x00010005

/* framebuffer related */
const MailboxTagSetPhysicalWidthHeight = 0x00048003
const MailboxTagSetVirtualWidthHeight = 0x48004
const MailboxTagSetVirtualOffset = 0x48009
const MailboxTagSetDepth = 0x48005
const MailboxTagSetPixelOrder = 0x48006
const MailboxTagGetFramebuffer = 0x40001
const MailboxTagGetPitch = 0x40008

type MailboxRegisterMap struct {
	Read     volatile.Register32    //0x00
	reserved [3]volatile.Register32 //0x04-0x10
	Poll     volatile.Register32    //0x10
	Sender   volatile.Register32    // 0x14
	Status   volatile.Register32    // 0x18
	Config   volatile.Register32    //0x1c
	Write    volatile.Register32    //0x20
}

type FrameBufferInfo struct {
	Buffer *byte
	Width  uint32
	Height uint32
	Pitch  uint32
}

// Uses of this function are NOT multithread safe. This is uses a single, shared
// mailbox data area.
func Call(ch uint8, mboxBuffer *sequenceOfSlots) bool {
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
	//	happiness.Console.Logf("%x,%x\n	", rawPtr, addrWithChannel)
	//}
	//happiness.Console.Logf("wasting time so the mailbox won't feel in a hurry...")
	for {
		if Mailbox.Status.HasBits(MailboxEmpty) {
			arm.Asm("nop")
		} else {
			read := Mailbox.Read.Get()
			if read == uint32(addrWithChannel) {
				//did we get a confirm?
				return mboxBuffer.s[1].Get() == MailboxResponse
			}
		}
	}
	return false //how would this happen?

}

func BoardID() (uint64, bool) {
	return MessageNoParams(MailboxTagSerial, 2)
}

func FirmwareVersion() (uint32, bool) {
	firmware, ok := MessageNoParams(MailboxTagFirmwareVersion, 1)
	if !ok {
		return 0x872720, ok
	}
	return uint32(firmware), true
}

func BoardModel() (uint32, bool) {
	model, ok := MessageNoParams(MailboxTagBoardModel, 1)
	if !ok {
		return 0x872728, ok
	}
	return uint32(model), true
}

func BoardRevision() (uint32, bool) {
	revision, ok := MessageNoParams(MailboxTagBoardRevision, 1)
	if !ok {
		return 0x872727, ok
	}
	return uint32(revision), true
}

func MACAddress() (uint64, bool) {
	addr, ok := MessageNoParams(MailboxTagMACAddress, 2)
	if !ok {
		return 0xab127348, false
	}

	addr &= 0x0000ffffffffffffffff
	return addr, true
}

func MessageNoParams(tag uint32, reqRespSlots int) (uint64, bool) {
	seq := message(0, tag, 2)
	if !Call(MailboxChannelProperties, seq) {
		return 77281, false
	}
	if reqRespSlots == 1 {
		return uint64(seq.s[5].Get()), true
	}
	if reqRespSlots == 2 {
		upper := uint64(seq.s[6].Get() << 32)
		lower := uint64(seq.s[5].Get())
		return upper + lower, true
	}
	panic("too many response slots")
}

// this type is super hairy.  It must be 16 byte aligned to be sent to the mailbox
// interface. Further, it has to be big enough to accomodate the largest message
// we send to the mailbox interface.  To do alignment use hackFor16ByteAlignment.
// This cannot be a slice because that introduces the slice structure above this
// object, which we can't use.
type sequenceOfSlots struct {
	s [36]volatile.Register32
}

//var seq sequenceOfSlots

func message(requestSlots int, tag uint32, responseSlots int) *sequenceOfSlots {

	totalSlots := uint32(1 + 1 + 1 + requestSlots + 1 + 1 + responseSlots + 1)
	larger := responseSlots
	if requestSlots > larger {
		larger = requestSlots
	}
	ptr := sixteenByteAlignedPointer(uintptr(totalSlots << 2)) //32 bit slots
	ptr32 := ((*uint32)(unsafe.Pointer(ptr)))
	seq := (*sequenceOfSlots)(unsafe.Pointer(ptr32))

	seq.s[0].Set(4 * totalSlots) //bytes of total size
	seq.s[1].Set(MailboxRequest)
	seq.s[2].Set(tag)
	seq.s[3].Set(uint32(larger) * 4)
	seq.s[4].Set(0) //request
	//s5...s5+larger-1 will be the outgoing data
	next := 5 + larger
	seq.s[next].Set(MailboxTagLast)
	return seq
}

/*

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
	*slotOffset(ptr32, 3+requestSlots) = uint32(responseSlots * 4) //Buffer size
	*slotOffset(ptr32, 4+requestSlots) = 0                         //its a request
	for j := 4 + responseSlots + 1; j < 4+responseSlots+requestSlots+1; j++ {
		*slotOffset(ptr32, j) = 0
	}
	*slotOffset(ptr32, 4+responseSlots+requestSlots+1) = MailboxTagLast //end
	return ptr32
}*/

// this returns the ARM clock rate, which can vary based on the underlying
// system clock speed
func GetClockRate() (uint32, bool) {
	buffer := message(2, MailboxTagGetClockRate, 2)

	buffer.s[5].Set(0x4)
	buffer.s[6].Set(0)
	if !Call(MailboxChannelProperties, buffer) {
		return 7903, false
	}
	if buffer.s[5].Get() != 4 {
		return 7904, false
	}
	return buffer.s[6].Get(), true
}
func GetVCMemoryAndBase() (uint32, uint32, bool) {
	buffer := message(0, MailboxTagGetVCMemory, 2)
	if !Call(MailboxChannelProperties, buffer) {
		return 7912, 17912, false
	}
	return buffer.s[5].Get(), buffer.s[6].Get(), true

}

func GetARMMemoryAndBase() (uint32, uint32, bool) {
	buffer := message(0, MailboxTagGetARMMemory, 2)
	if !Call(MailboxChannelProperties, buffer) {
		return 7914, 17914, false
	}
	return buffer.s[5].Get(), buffer.s[6].Get(), true

}

//pass in the number of bytes you want to be aligned to 16byte boundary
//the default allocator only ollocates things at their "natural" sizes
func sixteenByteAlignedPointer(size uintptr) *uint64 {
	units := (((size / 16) + 1) * 16) / 8
	bigger := make([]uint64, units)
	hackFor16ByteAlignment := ((*uint64)(unsafe.Pointer(&bigger[0])))
	ptr := uintptr(unsafe.Pointer(hackFor16ByteAlignment))
	if ptr&0xf != 0 {
		diff := uintptr(16 - (ptr & 0xf))
		hackFor16ByteAlignment = ((*uint64)(unsafe.Pointer(ptr + diff)))
	}
	return hackFor16ByteAlignment
}
func SetFramebufferRes1920x1200() *FrameBufferInfo {
	return setFramebufferRes(uint32(1920), uint32(1200))
}

func SetFramebufferRes1024x768() *FrameBufferInfo {
	return setFramebufferRes(uint32(1024), uint32(768))
}

func SetVirtualOffset(x uint32, y uint32) bool {
	buffer := message(2, MailboxTagSetVirtualOffset, 2)
	buffer.s[5].Set(x)
	buffer.s[6].Set(y)
	if !Call(MailboxChannelProperties, buffer) {
		return false
	}
	if buffer.s[5].Get() != x {
		fmt.Printf("unable to set virtual offset X to %d (got %d)", x, buffer.s[5].Get())
		return false
	}
	if buffer.s[6].Get() != y {
		fmt.Printf("unable to set virtual offset Y to %d (got %d)", x, buffer.s[5].Get())
		return false
	}
	return true
}

func setFramebufferRes(widthPixels uint32, heightPixels uint32) *FrameBufferInfo {

	ptr := sixteenByteAlignedPointer(uintptr(36 << 2)) //32 bit slots
	ptr32 := ((*uint32)(unsafe.Pointer(ptr)))
	mbox := (*sequenceOfSlots)(unsafe.Pointer(ptr32))

	mbox.s[0].Set(35 * 4) //size in bytes
	mbox.s[1].Set(MailboxRequest)

	mbox.s[2].Set(MailboxTagSetPhysicalWidthHeight)
	mbox.s[3].Set(8)
	mbox.s[4].Set(8)
	mbox.s[5].Set(widthPixels)  //Width
	mbox.s[6].Set(heightPixels) //height

	mbox.s[7].Set(MailboxTagSetVirtualWidthHeight)
	mbox.s[8].Set(8)
	mbox.s[9].Set(8)
	mbox.s[10].Set(widthPixels)  //virtual Width
	mbox.s[11].Set(heightPixels) //virtual Height

	mbox.s[12].Set(MailboxTagSetVirtualOffset)
	mbox.s[13].Set(8)
	mbox.s[14].Set(8)
	mbox.s[15].Set(0) //x offset
	mbox.s[16].Set(0) //y offset

	mbox.s[17].Set(MailboxTagSetDepth)
	mbox.s[18].Set(4)
	mbox.s[19].Set(4)
	mbox.s[20].Set(32) //depth

	mbox.s[21].Set(MailboxTagSetPixelOrder)
	mbox.s[22].Set(4)
	mbox.s[23].Set(4)
	mbox.s[24].Set(1) //RGB

	mbox.s[25].Set(MailboxTagGetFramebuffer)
	mbox.s[26].Set(8)
	mbox.s[27].Set(8)
	mbox.s[28].Set(4096)
	mbox.s[29].Set(0)

	mbox.s[30].Set(MailboxTagGetPitch)
	mbox.s[31].Set(4)
	mbox.s[32].Set(4)
	mbox.s[33].Set(0) //Pitch

	mbox.s[34].Set(MailboxTagLast) //we are done here
	if !Call(MailboxChannelProperties, mbox) {
		trust.Errorf("unable to send commands to mailbox for framebuffer setup")
		return nil
	}
	//check on the response
	fbuffer := mbox.s[28].Get()
	result := &FrameBufferInfo{}
	if mbox.s[20].Get() != 32 || fbuffer == 0 {
		trust.Errorf("unable to set Width & Height of framebuffer to 1024,768")
		return nil
	}

	fbuffer = fbuffer & 0x3FFFFFFF
	mbox.s[28].Set(fbuffer)
	result.Width = mbox.s[5].Get()
	result.Height = mbox.s[6].Get()
	result.Pitch = mbox.s[33].Get()
	result.Buffer = (*byte)(unsafe.Pointer(uintptr(mbox.s[28].Get())))
	return result
}
