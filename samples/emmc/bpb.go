package main

import (
	"feelings/src/golang/bytes"
	"feelings/src/golang/encoding/binary"
	"feelings/src/lib/trust"
)

type biosParamBlock struct {
	isFat16 bool
	biosParamBlockShared
	fat16 *biosParamBlockFat16Extension
	fat32 *biosParamBlockFat32Extension
}

type biosParamBlockShared struct { //in the Volume Boot Record
	jump           [3]uint8 //0x0
	oem            [8]uint8 //0x3
	BytesPerSector uint16   //0xB
	Spc            uint8    //0xD   sectors per clusters
	Rsc            uint16   //0xE   reserved sector count
	Nf             uint8    //0x10  number of fats
	NumRootEntries uint16   //0x11  number root entries 0
	Ts16           uint16   //0x13  total sectors
	Media          uint8    //0x15  media descriptors
	Spf16          uint16   //0x16  sectors per fat
	Spt            uint16   //0x18  sectors per track
	Nh             uint16   //0x1A  number of heads
	Hs             uint32   //0x1C  hidden sectors
	Ts32           uint32   //0x20  total sectors 32bits
}
type biosParamBlockFat16Extension struct {
	DriveNumber    uint8     // 0x24 0x0      ( As used in PC interrupt 13)
	reserved1      uint8     // 0x25 0x1
	BootSig        uint8     // 0x26 0x2		(0x29)
	VolumeID       uint32    // 0x27 0x3
	volumeLabel    [11]uint8 // 0x2B 0x7	    (Label or  "NO NAME    ")
	fileSystemType [8]uint8  // 0x36 0x12      ("FAT16   ", "FAT     ", or all zero.)
}

type biosParamBlockFat32Extension struct {
	FATSize32      uint32    // 0x24 0x0  	(Number of sectors per FAT on FAT32)
	ExtFlags       uint16    // 0x28 0x4  	(Mirror flags Bits 0-3: number of active FAT (if bit 7 is 1) Bit 7: 1 = single active FAT; zero: all FATs are updated at runtime; Bits 4-6 & 8-15 : reserved)
	FSVersion      uint16    // 0x2A 0x6
	RootCluster    uint32    // 0x2C 0x8  	(usually 2)
	FSInfo         uint16    // 0x30 0xC  	(usually 1)
	BkBootSec      uint16    // 0x32 0xE  	(usually 6)
	reserved       [12]uint8 // 0x34 0x10
	DriveNumber    uint8     // 0x40 0x1C
	reserved1      uint8     // 0x41 0x1D
	BootSig        uint8     // 0x42 0x1E   (0x29: extend signature .. Early FAT32 stop here, 0x4f )
	VolumeID       uint32    // 0x43 0x1F
	volumeLabel    [11]uint8 // 0x47 0x23   (Label or  "NO NAME    ")
	fileSystemType [8]uint8  // 0x50 0x2E	("FAT32   ", "FAT     ", or all zero.)
}

func (b *biosParamBlockShared) unpack(buffer []uint8) bool {
	buf := bytes.NewBuffer(buffer)

	err := binary.Read(buf, binary.LittleEndian, b)
	if err != nil {
		trust.Errorf("failed to read binary data for bios param block: %v ", err.Error())
		return false
	}
	copy(b.jump[0:3], buffer[0:3])
	copy(b.oem[0:8], buffer[3:11])
	return true
}

func (b *biosParamBlockFat32Extension) unpack(buffer []uint8) bool {
	buf := bytes.NewBuffer(buffer)

	err := binary.Read(buf, binary.LittleEndian, b)
	if err != nil {
		trust.Errorf("failed to read binary data for bios param block fat 16 extension: %v", err.Error())
		return false
	}
	copy(b.volumeLabel[0:11], buffer[0x23:0x23+11])
	copy(b.fileSystemType[0:8], buffer[0x2E:0x2E+8])
	return true
}

func (b *biosParamBlockFat16Extension) unpack(buffer []uint8) bool {
	buf := bytes.NewBuffer(buffer)

	err := binary.Read(buf, binary.LittleEndian, b)
	if err != nil {
		trust.Errorf("failed to read binary data for bios param block fat 16 extension: %v ", err.Error())
		return false
	}
	copy(b.volumeLabel[0:11], buffer[0x7:0x7+11])
	copy(b.fileSystemType[0:8], buffer[0x12:0x12+8])
	return true
}

func newBIOSParamBlock(shared *biosParamBlockShared, f16 *biosParamBlockFat16Extension, f32 *biosParamBlockFat32Extension) *biosParamBlock {
	result := &biosParamBlock{
		biosParamBlockShared: *shared,
	}
	if f16 != nil {
		result.fat16 = f16
		result.isFat16 = true
	} else {
		result.fat32 = f32
	}
	return result
}
