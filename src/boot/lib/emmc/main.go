package main

import (
	"fmt"
	"io"
	"unsafe"

	"device/arm"
	"machine"

	"lib/trust"
)

//export raw_exception_handler
func rawExceptionHandler(t uint64, esr uint64, addr uint64, el uint64, procId uint64) {
	trust.Errorf("interrupt: type=%d, esr=%x, addr=%x, el=%d,  procId=%d",
		t, esr, addr, el, procId)
	for {
		arm.Asm("nop")
	}
}

type mbrInfo struct {
	unused     [mbrUnusedSize]uint8
	Partition1 partitionInfo //customary to number from 1
	Partition2 partitionInfo
	Partition3 partitionInfo
	Partition4 partitionInfo
	Signature  uint16 //0xaa55
}

type partitionInfo struct {
	Status         uint8  // 0x80 - active partition
	HeadStart      uint8  // starting head
	CylSelectStart uint16 // starting cylinder and sector
	Type           uint8  // partition type (01h = 12bit FAT, 04h = 16bit FAT, 05h = Ex MSDOS, 06h = 16bit FAT (>32Mb), 0Bh = 32bit FAT (<2048GB))
	HeadEnd        uint8  // ending head of the partition
	CylSectEnd     uint16 // ending cylinder and sector
	FirstSector    uint32 // total sectors between MBR & the first sector of the partition
	SectorsTotal   uint32 // size of this partition in sectors
}

type fatPartition struct {
	rootCluster         uint32 // Active partition rootCluster
	sectorsPerCluster   uint32 // Active partition sectors per cluster
	bytesPerSector      uint32 // Active partition bytes per sector
	fatOrigin           uint32 // The beginning of the 1 or more FATs (sector number)
	fatSize             uint32 // fat size in sectors, including all FATs
	dataSectors         uint32 // Active partition data sectors
	unusedSectors       uint32 // Active partition unused sectors (this is also the offset of the partition)
	reservedSectorCount uint32 // Active partition reserved sectors
	isFAT16             bool
}

func main() {
	machine.MiniUART = machine.NewUART()
	_ = machine.MiniUART.Configure(&machine.UARTConfig{ /*no interrupt*/ })

	buffer := make([]byte, 512)
	//for now, hold the buffers on stack
	sectorCache := make([]byte, 0x200<<6) //0x40 pages
	sectorBitSet := make([]uint64, 1)

	trust.Debugf("init1")
	if emmcinit() != 0 {
		trust.Errorf("Unable init emmc interface")
		machine.Abort()
	}
	emmcenable()
	emmcgoidle()
	trust.Debugf("init2")
	sdcard := fatGetPartition(buffer) //data read into this buffer
	if sdcard == nil {
		trust.Errorf("Unable to read MBR or unable to parse BIOS parameter block")
		machine.Abort()
	}
	tranq := NewTraquilBufferManager(unsafe.Pointer(&sectorCache[0]), 0x40,
		unsafe.Pointer(&sectorBitSet[0]), sdcard.readInto, nil)
	fs := NewFAT32Filesystem(tranq, sdcard)
	rd, err := fs.Open("/readme")
	if err != nil {
		trust.Errorf("unable to open path: %s", err.Error())
	}
	readerBuf := make([]uint8, 256)
	for {
		n, err := rd.Read(readerBuf)
		if err == io.EOF {
			break
		}
		if err != nil {
			trust.Errorf("failed reading file: %s", err.Error())
		}
		if n == 0 {
			continue
		}
		s := string(readerBuf[:n])
		fmt.Printf(s)
	}
}
