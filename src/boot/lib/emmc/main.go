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
	Status         uint8        // 0x80 - active partition
	HeadStart      uint8        // starting head
	CylSelectStart uint16       // starting cylinder and sector
	Type           uint8        // partition type (01h = 12bit FAT, 04h = 16bit FAT, 05h = Ex MSDOS, 06h = 16bit FAT (>32Mb), 0Bh = 32bit FAT (<2048GB))
	HeadEnd        uint8        // ending head of the partition
	CylSectEnd     uint16       // ending cylinder and sector
	FirstSector    sectorNumber // total sectors between MBR & the first sector of the partition
	SectorsTotal   uint32       // size of this partition in sectors
}

type fatPartition struct {
	rootCluster         clusterNumber // Active partition rootCluster
	sectorsPerCluster   uint32        // Active partition sectors per cluster
	bytesPerSector      uint32        // Active partition bytes per sector
	fatOrigin           sectorNumber  // The beginning of the 1 or more FATs (sector number)
	fatSize             uint32        // fat size in sectors, including all FATs
	dataSectors         uint32        // Active partition data sectors
	unusedSectors       uint32        // Active partition unused sectors (this is also the offset of the partition)
	reservedSectorCount uint32        // Active partition reserved sectors
	isFAT16             bool
}

func main() {
	machine.MiniUART = machine.NewUART()
	_ = machine.MiniUART.Configure(&machine.UARTConfig{ /*no interrupt*/ })

	buffer := make([]byte, 512)
	//for now, hold the buffers on stack
	sectorCache := make([]byte, 0x200<<6) //0x40 pages
	sectorBitSet := make([]uint64, 1)
	//trust.DefaultLogger.SetLevel(trust.EverythingButDebug)

	//raw init of interface
	if emmcinit() != 0 {
		trust.Errorf("Unable init emmc interface")
		machine.Abort()
	}
	// set the clock to the init speed (slow) and set some flags so
	// we will be ready for proper init
	emmcenable()

	if err := sdfullinit(); err != sdOk {
		trust.Errorf("Unable to do a full initialization of the EMMC interafce, aborting")
		machine.Abort()
	}

	sdcard := fatGetPartition(buffer) //data read into this buffer
	if sdcard == nil {
		trust.Errorf("Unable to read MBR or unable to parse BIOS parameter block")
		machine.Abort()
	}

	buf := make([]byte, 512)
	if err := readInto(sectorNumber(4228), unsafe.Pointer(&buf[0])); err != sdOk {
		trust.Errorf("unable to read sector %d: %v", 19929, err)
		machine.Abort()
	}
	bytesRead, _ := sdReadblock( /*was 0x20*/ sdcard.activePartition.clusterNumberToSector(2, 0x1d), 1)
	trust.Debugf("read lba 0x20: %d bytes", bytesRead)

	tranq := NewTraquilBufferManager(unsafe.Pointer(&sectorCache[0]), 0x40,
		unsafe.Pointer(&sectorBitSet[0]), nil, nil)
	fs := NewFAT32Filesystem(tranq, sdcard)
	path := "/motd-news"
	rd, err := fs.Open(path)
	if err != nil {
		trust.Fatalf(1, "unable to open path: %s", path)
		machine.Abort()
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
