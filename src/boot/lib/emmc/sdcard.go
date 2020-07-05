package main

import (
	"errors"
	"unsafe"

	"machine"

	"lib/trust"
)

const sdOk = 0
const sdTimeout = -1
const sdError = -2

// this is the either the whole disk or the 1st partition
type sdCardInfo struct {
	// xxx add details about the card itself
	activePartition *fatPartition
}

func (s *sdCardInfo) readInto(sector uint32, data unsafe.Pointer) error {
	result := sdReadblockInto(sector, 1, data)
	if result == 0 {
		errors.New("should be a read error type")
	}
	return nil
}

//reads into a buffer created on the heap
func sdReadblock(lba uint32, num uint32) (int, []byte) {
	buffer := make([]byte, sectorSize*num)
	buf := unsafe.Pointer(&buffer[0])
	read := sdReadblockInto(lba, num, buf)
	return read, buffer
}

//reads num sectors starting at lba into a buffer
//provided
func sdReadblockInto(lba uint32, num uint32, buf unsafe.Pointer) int {
	if num < 1 {
		trust.Errorf("sdreadblock: requested bad number of blocks (%d), using 1 instead",
			num)
		num = 1
	}
	var resp [4]uint32
	trust.Debugf("start reading %d blocks, first block @%d: ", num, lba)
	machine.EMMC.BlockSizeAndCount.SetBlkCnt(num)
	machine.EMMC.BlockSizeAndCount.SetBlkSize(sectorSize)
	if num == 1 {
		if emmccmd(ReadSingle, lba, &resp) != 0 {
			trust.Errorf("aborting read block into for block %d", lba)
			return sdError
		}
		return sdOk
	} else {
		if emmccmd(ReadMulti, lba, &resp) != 0 {
			trust.Errorf("aborting read multi block into for block %d", lba)
			return sdError
		}
	}
	syncdata(false)
	c := uint32(0)
	for c < num {
		ptr := unsafe.Pointer(uintptr(buf) + uintptr(c*sectorSize))
		if err := syncio(false, ptr, sectorSize); err != sdOk {
			return err
		}
		c++ //yech
	}
	return sdOk
}

func syncdata(write bool) int {
	// enable interrupt
	machine.EMMC.EnableInterrupt.Set(machine.EMMC.EnableInterrupt.Get() | dataDoneOrError)
	for j := 0; j < 30; j++ {
		if dataDone() {
			break
		}
	}
	i := machine.EMMC.Interrupt.Get() & everythingButCardIntr
	if i&Datadone == 0 {
		trust.Errorf("emmcio: write=%v timeout intr %x stat %x", write, i,
			machine.EMMC.DebugStatus.Get())
		machine.EMMC.Interrupt.Set(i) // clear the intrupts
		return sdError
	}
	if i&Err != 0 {
		trust.Errorf("emmcio: write=%v error intr %x stat %x\n",
			write, machine.EMMC.Interrupt.Get(), machine.EMMC.DebugStatus.Get())
		machine.EMMC.Interrupt.Set(i) // clear the intrupts
		return sdError
	}
	return sdOk
}

//sync io is poor :-( note that this does NOT clear interrupts because
//we may doing multiplereads
func syncio(write bool, buf unsafe.Pointer, bufSize uint32) int {

	// must be 4 byte aligned
	if uintptr(buf)&03 != 0 {
		panic("data sent or received to sdio card must be 4 byte aligned")
	}

	if !write {
		for d := uint32(0); d < bufSize/4; d++ {
			buffer := (*uint32)(unsafe.Pointer(uintptr(buf) + uintptr(d*4)))
			*buffer = machine.EMMC.Data.Get()
		}
	} else {
		panic("not implemented yet")
	}
	return 0
}
