package emmc

import (
	"bytes"
	"encoding/binary"
	"io"
	"path/filepath"
	"strings"
	"unicode/utf16"

	"lib/trust"
)

// in linux
// dd if=/dev/zero of=foo bs=1M count=128
// fdisk foo -- create two partitions of type 'b'
// losetup -Pf foo  -- mounts as /dev/loop0p1 and /dev/loop0p2
// mkdir /mnt/p1
// mkdir /mnt/p2
// mount /dev/loop1p1 /mnt/p1
// mount /dev/loop1p2 /mnt/p2
// cd /mnt/p1  -- and copy your files to each partition
// copy all text files in a heirarchy with
// find /etc -type f -exec grep -Iq . {} \; -print -exec cp {} . \;

const directoryEntrySize = 0x20
const showCommands = true
const fat32EOCBoundary = 0xFFFFFF8 //anything at or above this is EOC
const fat16EOCBoundary = 0xFFF8    //anything at or above this is EOC
const sizeOfPackedBpbShared = 0x24
const mbrUnusedSize = 446
const sizeOfPackedPartitionInfo = 0x10
const directoryEnd = 0x0
const directoryEntryDeleted = 0xE5
const directoryEntryLFN = 0xF
const attributeSubdirectory = 0x10
const sectorSize = 0x200
const dumpFatInfo = false

// these two are windows NT specific
//const lowercaseName = 0x10
//const lowercaseExt = 0x8

type fat32LFN struct { //used after we confirmed the raw values are ok
	sequenceNumber  byte
	isLast          bool
	isFirstPhysical bool
	segment         string
	firstClusterLo  uint16
}

type rawFat32LFN struct {
	SequenceNumber  byte
	name0           [5]uint16
	Attributes      byte
	Type            byte
	DOSNameChecksum byte
	name1           [6]uint16
	FirstCluster    uint16
	name2           [2]uint16
}

type rawDirEnt struct {
	name           [8]uint8
	ext            [3]uint8
	Attrib         byte
	NTReserved     byte
	TimeTenth      byte
	WriteTime      uint16
	WriteDate      uint16
	LastAccessDate uint16
	FirstClusterHi uint16
	CreateTime     uint16
	CreateDate     uint16
	FirstClusterLo uint16
	Size           uint32
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

type sectorNumber uint32
type clusterNumber uint32
type inodeNumber uint64

func strlenWithTerminator(p []uint8, terminator uint8) int {
	l := 0
	// tricky: order of the terms in the && matters
	for l < len(p) && p[l] != terminator {
		l++
	}
	//rt.MiniUART.WriteString("strlen " + string(p) + "\n")

	return l
}

//
// Get the starting LBA address of the first partition
// so that we know where our FAT file system starts, and
// read that volume's BIOS Parameter Block
//
func fatGetPartition(buffer []uint8) (*sdCardInfo, EmmcError) { //xxx should be passed in setup by Init
	sdCard := sdCardInfo{
		activePartition: &fatPartition{},
	}
	_, buffer, err := sdReadblock(0, 1)
	if err != EmmcOk {
		return nil, err
	}
	//we need to check and see if boot sector...
	bpb := biosParamBlockShared{}
	if !bpb.unpack(buffer) {
		return nil, EmmcBadBIOSParamBlock
	}
	if bpb.jump[0] != 0xE9 && bpb.jump[0] != 0xEB {
		mbrData := bytes.NewBuffer(buffer)
		var mbr mbrInfo
		if err := binary.Read(mbrData, binary.LittleEndian, &mbr); err != nil {
			trust.Errorf("unable to read MBR: %v ", err.Error())
			return nil, EmmcBadMBR
		}
		if mbr.Signature != 0xaa55 {
			trust.Errorf("bad magic number in MBR (%x)", mbr.Signature)
			return nil, EmmcNoMBRSignature
		}
		if err := unpackPartitionBuffer(buffer, mbrUnusedSize, 0, sizeOfPackedPartitionInfo, &mbr.Partition1); err != nil {
			return nil, EmmcBadPartitions
		}
		if err := unpackPartitionBuffer(buffer, mbrUnusedSize, sizeOfPackedPartitionInfo, 2*sizeOfPackedPartitionInfo, &mbr.Partition2); err != nil {
			return nil, EmmcBadPartitions
		}
		if err := unpackPartitionBuffer(buffer, mbrUnusedSize, sizeOfPackedPartitionInfo*2, 3*sizeOfPackedPartitionInfo, &mbr.Partition3); err != nil {
			return nil, EmmcBadPartitions
		}
		if err := unpackPartitionBuffer(buffer, mbrUnusedSize, sizeOfPackedPartitionInfo*3, 4*sizeOfPackedPartitionInfo, &mbr.Partition4); err != nil {
			return nil, EmmcBadPartitions
		}
		// log.Printf("read MBR: %+v", mbr)
		// log.Printf("active partiton: %+v", mbr.Partition1)
		// log.Printf(
		// 	"dead partiton: %+v", mbr.Partition2)
		//deeply dubious xxx
		first := uint32(mbr.Partition1.FirstSector) // FAT16 needs this value so hold it
		sdCard.activePartition.unusedSectors = first
		//try again for BPB at firstDataSector
		_, buffer, err = sdReadblock(mbr.Partition1.FirstSector, 1)
		if err != EmmcOk {
			return nil, err
		}
		bpb = biosParamBlockShared{}
		if !bpb.unpack(buffer) {
			return nil, EmmcBadBIOSParamBlock
		}
		if bpb.jump[0] != 0xE9 && bpb.jump[0] != 0xEB {
			trust.Errorf("did not find a BIOS Parameter Block")
			return nil, EmmcBadBIOSParamBlock
		}
	}
	// we have only read the shared part, so read extension as appropriate
	var ext16 *biosParamBlockFat16Extension
	var ext32 *biosParamBlockFat32Extension
	if bpb.Spf16 > 0 && bpb.NumRootEntries > 0 {
		ext16 := &biosParamBlockFat16Extension{}
		if ext16.unpack(buffer[sizeOfPackedBpbShared:]) {
			return nil, EmmcBadBIOSParamBlock
		}
	} else {
		ext32 = &biosParamBlockFat32Extension{}
		if !ext32.unpack(buffer[sizeOfPackedBpbShared:]) {
			return nil, EmmcBadBIOSParamBlock
		}
	}
	bpbFull := newBIOSParamBlock(&bpb, ext16, ext32)
	sdCard.activePartition.bytesPerSector = uint32(bpbFull.BytesPerSector) // Bytes per sector on partition
	sdCard.activePartition.sectorsPerCluster = uint32(bpbFull.Spc)         // Hold the sector per cluster count
	sdCard.activePartition.reservedSectorCount = uint32(bpbFull.Rsc)       // Hold the reserved sector count
	if !bpbFull.isFat16 {                                                  // Check if FAT16/FAT32
		// FAT32
		sdCard.activePartition.rootCluster = clusterNumber(bpbFull.fat32.RootCluster) // Hold partition root cluster
		raw := uint32(bpbFull.Rsc) + bpbFull.Hs + sdCard.activePartition.unusedSectors
		sdCard.activePartition.fatOrigin = sectorNumber(raw)
		if dumpFatInfo {
			trust.Infof("FAT32 Partition Info FAT Origin : 0x%x ", sdCard.activePartition.fatOrigin)
		}
		sdCard.activePartition.dataSectors = bpbFull.Ts32 - uint32(bpbFull.Rsc) - (bpbFull.fat32.FATSize32 * uint32(bpbFull.Nf))
		//sdCard.partition.dataSectors = bpb->TotalSectors32 - bpb->ReservedSectorCount - (bpb->FSTypeData.fat32.FATSize32 * bpb->NumFATs);
		if dumpFatInfo {
			trust.Infof("FAT32 Partition Info Total Sectors : 0x%x ", bpbFull.Ts32)
			trust.Infof("FAT32 Partition Info Data Sectors : 0x%x", sdCard.activePartition.dataSectors)
		}
		if bpbFull.fat32.BootSig != 0x29 {
			trust.Errorf("FAT32 volume has bad boot signature: %v", uint32(bpbFull.fat32.BootSig))
			return nil, EmmcBadFAT32BootSignature
		}
		if dumpFatInfo {
			trust.Infof("FAT32 Volume Label: '%s', ID: 0x%08x\n",
				string(bpbFull.fat32.volumeLabel[:]), bpbFull.fat32.VolumeID) //xxx because of problem in tinyo reflection with bpbfull
		}
		sdCard.activePartition.fatSize = uint32(bpbFull.Nf) * bpbFull.fat32.FATSize32
		if dumpFatInfo {
			trust.Infof("active partition: fat origin          %d", sdCard.activePartition.fatOrigin)
			trust.Infof("active partition: fat size            %d", sdCard.activePartition.fatSize)
			trust.Infof("active partition: root cluster        %d", sdCard.activePartition.rootCluster)
			trust.Infof("active partition: sectors per cluster %d", sdCard.activePartition.sectorsPerCluster)
			trust.Infof("active partition: bytes per sector    %d", sdCard.activePartition.bytesPerSector)
		}

		if bpbFull.fat32.fileSystemType[0] != 'F' ||
			bpbFull.fat32.fileSystemType[1] != 'A' ||
			bpbFull.fat32.fileSystemType[2] != 'T' {
			return nil, EmmcBadFAT32FilesystemType
		}
	} else {
		// FAT16
		sdCard.activePartition.rootCluster = 2 // Hold partition root cluster, FAT16 always start at 2
		raw := sdCard.activePartition.unusedSectors + (uint32(bpbFull.Nf) * uint32(bpbFull.Spf16)) + 1
		sdCard.activePartition.fatOrigin = sectorNumber(raw)
		// data sectors x sectorsize = capacity ... I have check this on PC and gives right calc
		sdCard.activePartition.dataSectors = bpbFull.Ts32 - (uint32(bpbFull.Nf) * uint32(bpbFull.Spf16)) - 33
		trust.Infof("FAT16 reserved sectors: 0x%x", uint32(bpbFull.Rsc))
		trust.Infof("FAT16 FAT origin sector: 0x%x", sdCard.activePartition.fatOrigin)
		trust.Infof("FAT32 Volume Label: %s, ID: 0x%08x\n",
			bpbFull.fat16.volumeLabel, bpbFull.fat16.VolumeID)
		if bpbFull.fat16.BootSig != 0x29 {
			trust.Errorf("FAT16 volume has bad boot signature:  %x", uint32(bpbFull.fat16.BootSig))
			return nil, EmmcBadFAT16BootSignature
		}
		if bpbFull.fat16.fileSystemType[0] != 'F' ||
			bpbFull.fat16.fileSystemType[1] != 'A' ||
			bpbFull.fat16.fileSystemType[2] != 'T' {
			trust.Errorf("Wrong filesystem type (not FAT)")
			return nil, EmmcBadFAT16FilesystemType
		}
		sdCard.activePartition.fatSize = uint32(bpbFull.Nf) * uint32(bpbFull.Spf16)
		sdCard.activePartition.isFAT16 = true
	}
	return &sdCard, EmmcOk
}

func (f *fatPartition) clusterNumberToSector(numFats uint32, c clusterNumber) sectorNumber {
	cnum := uint32(c)
	sect := ((cnum - 2) * f.sectorsPerCluster) + uint32(f.fatOrigin) + (f.fatSize * numFats)
	return sectorNumber(sect)
}

func (f *rawDirEnt) unpack(buffer []uint8) bool {
	buf := bytes.NewBuffer(buffer)
	//rt.MiniUART.Dump(unsafe.Pointer(&buf.Bytes()[0]))

	err := binary.Read(buf, binary.LittleEndian, f)
	if err != nil {
		trust.Errorf("failed to read binary data for directory entry: %x", err.Error())
		return false
	}
	copy(f.name[:], buffer[0:8])
	copy(f.ext[:], buffer[8:11])
	return true
}

func longFilename(in []byte) *fat32LFN {
	buffer := bytes.NewBuffer(in)
	raw := &rawFat32LFN{}
	if err := binary.Read(buffer, binary.LittleEndian, raw); err != nil {
		trust.Errorf("Unable to decode binary format for long filename: %x", err.Error())
		return nil
	}
	buffer.Reset()
	offset := 1
	for i := 0; i < 5; i++ {
		raw.name0[i] = uint16(in[(i*2)+1+offset])<<8 + uint16(in[i*2+offset])
	}
	offset = 0xE
	for i := 0; i < 6; i++ {
		raw.name1[i] = uint16(in[(i*2)+1+offset])<<8 + uint16(in[i*2+offset])
	}
	offset = 0x1C
	for i := 0; i < 2; i++ {
		raw.name2[i] = uint16(in[(i*2)+1+offset])<<8 + uint16(in[i*2+offset])
	}
	if raw.Attributes != 0xF || raw.FirstCluster != 0 || raw.Type != 0 {
		trust.Warnf("badly formed fat32 long filename record: 0x%x, 0x%x 0x%x",
			uint32(raw.Attributes), uint32(raw.FirstCluster), uint32(raw.Type))
		return nil
	}
	output := &fat32LFN{}
	output.isLast = false
	if raw.SequenceNumber&0x40 > 0 {
		output.isLast = true
	}
	if raw.SequenceNumber&0x20 > 0 {
		output.isFirstPhysical = true
	}
	sn := raw.SequenceNumber & 0x1F
	output.sequenceNumber = sn
	output.firstClusterLo = raw.FirstCluster
	if output.sequenceNumber < 1 || output.sequenceNumber > 0x14 {
		trust.Warnf("badly formed fat32 long filename record, sequence number: 0x%x ",
			uint32(output.sequenceNumber))
	}
	count := uint32(0)
	done := false
	var holdChars [14]uint16
outer0:
	for i := 0; i < 5; i++ {
		switch raw.name0[i] {
		case 0x0, 0xffff:
			done = true
			break outer0
		default:
			holdChars[count] = raw.name0[i]
			count++
		}
	}
	//fmt.Printf("----0 holdchars(%d) '%+v'\n", count, holdChars)
	if !done {
	outer1:
		for i := 0; i < 6; i++ {
			switch raw.name1[i] {
			case 0x0, 0xffff:
				done = true
				break outer1
			default:
				holdChars[count] = raw.name1[i]
				count++
			}
		}
	}
	//fmt.Printf("----1 holdchars(%d) '%+v'\n", count, holdChars)
	if !done {
	outer2:
		for i := 0; i < 2; i++ {
			switch raw.name2[i] {
			case 0x0, 0xffff:
				done = true
				break outer2
			default:
				holdChars[count] = raw.name2[i]
				count++
			}
		}
	}
	//fmt.Printf("----2 holdchars(%d) '%+v'\n", count, holdChars)

	runes := make([]rune, count)
	ct := 0
	for i := 0; i < int(count); i++ {
		r, pair := ucs2ToRune(holdChars[i])
		if !pair {
			runes[ct] = r
			ct++
			continue
		}
		//nasty case
		if len(holdChars)-1 == i {
			trust.Warnf("bad utf-16 in surrogate pair encoding at end of name, ignored")
			continue
		}
		r2, pair := ucs2ToRune(holdChars[i+1])
		if !pair {
			trust.Warnf("bad utf-16 in surrogate pair encoding in name at character, ignored : %x", uint32(i+1))
			continue
		}
		runes[ct] = utf16.DecodeRune(r, r2)
		ct++
	}

	output.segment = string(runes)
	return output
}
func ucs2ToRune(u uint16) (rune, bool) {
	if u >= 0xD800 && u <= 0xDFFF {
		return 0, true
	}
	return rune(u), false
}

func unpackPartitionBuffer(buffer []byte, initialPadding int, start int, end int, part *partitionInfo) error {
	pbuf := bytes.NewBuffer(buffer[initialPadding+start : initialPadding+end])
	if err := binary.Read(pbuf, binary.LittleEndian, part); err != nil {
		trust.Errorf("unable to read partition descriptor of partition 1: %x", err.Error())
		return err
	}
	return nil
}

type FAT32Filesystem struct {
	tranq     bufferManager
	sdcard    *sdCardInfo
	inodeMap  map[string]inodeNumber
	nextInode inodeNumber
}

func NewFAT32Filesystem(tranq *Tranquil, sdcard *sdCardInfo) *FAT32Filesystem {
	return &FAT32Filesystem{
		tranq:     tranq,
		sdcard:    sdcard,
		inodeMap:  make(map[string]inodeNumber),
		nextInode: 1,
	}
}
func (f *FAT32Filesystem) NewInode() inodeNumber {
	result := f.nextInode
	f.nextInode++
	return result
}

// OpenDir returns a directory pointer that you can then call ReadDir on, or it returns null.
func (f *FAT32Filesystem) openRootDir() (*Dir, EmmcError) {
	trust.Debugf("openRootDir")
	return f.readDirFromCluster("/", f.sdcard.activePartition.rootCluster)
}

func (f *FAT32Filesystem) Open(path string) (*fatDataReader, EmmcError) {
	trust.Infof("open file %s: trying to resolve path", path)
	entry, err := f.resolvePath(path, true)
	if err != EmmcOk {
		return nil, err
	}
	trust.Infof("opening %s: %x,%x (start cluster => %x)", path, uint32(entry.firstClusterHi),
		uint32(entry.firstClusterLo),
		uint32(entry.firstClusterHi)*256+uint32(entry.firstClusterLo))
	raw := uint32(entry.firstClusterHi)*256 + uint32(entry.firstClusterLo)
	cnum := clusterNumber(raw)
	reader, err := newFATDataReader(cnum, f.sdcard.activePartition, f.tranq, entry.size)
	if reader == nil {
		trust.Errorf("unable to create new FATDataReader! need better error\n")
		return nil, err
	}
	return reader, EmmcOk
}

func (f *FAT32Filesystem) OpenDir(path string) (*dirEnt, EmmcError) {
	trust.Infof("open dir %s: trying to resolve path", path)
	entry, err := f.resolvePath(path, true)
	if err != EmmcOk {
		return nil, err
	}
	trust.Infof("opening directory %s: %x,%x (start cluster => %x)", path, uint32(entry.firstClusterHi),
		uint32(entry.firstClusterLo),
		uint32(entry.firstClusterHi)*256+uint32(entry.firstClusterLo))
	raw := uint32(entry.firstClusterHi)*256 + uint32(entry.firstClusterLo)
	cnum := clusterNumber(raw)
	trust.Errorf("OpenDir got directory %s, with cnum: %d", path, cnum)
	return nil, EmmcOk
}

func (f *FAT32Filesystem) openDirFromEntry(entry *dirEnt) (*Dir, EmmcError) {
	trust.Debugf("openDirFromEntry %s", entry.Name())
	path := filepath.Clean(entry.Path)
	raw := uint32(entry.firstClusterHi)*256 + uint32(entry.firstClusterLo)
	cnum := clusterNumber(raw)
	return f.readDirFromCluster(path, cnum)
}

func (f *FAT32Filesystem) resolvePath(path string, isLast bool) (*dirEnt, EmmcError) {
	path = filepath.Clean(path)
	trust.Debugf("resolve path entered: %s isLast=%v", path, isLast)
	dirPath, file := filepath.Split(path)
	var dir *Dir
	var err EmmcError
	var entry *dirEnt

	if dirPath == "/" {
		dir, err = f.openRootDir()
		if err != EmmcOk {
			return nil, err
		}
	} else {
		entry, err = f.resolvePath(dirPath, false)
		if err != EmmcOk {
			return nil, err
		}
		trust.Debugf("resolve path: resolve path of %s finished, entry %s", dirPath, entry.name)
		dir, err = f.openDirFromEntry(entry)
		trust.Debugf("resolve path: openDirFromEntry %s", entry.name)
		if err != EmmcOk {
			return nil, err
		}
	}
	file = strings.ToUpper(file)
	for _, e := range dir.contents {
		entryName := strings.ToUpper(e.Name())
		if entryName == file {
			if !isLast && !e.IsDir() {
				return nil, EmmcNotFile
			}
			return &e, EmmcOk
		}
	}
	return nil, EmmcNoEnt
}

func (f *FAT32Filesystem) readDirFromCluster(path string, cnum clusterNumber) (*Dir, EmmcError) {
	trust.Debugf("readDirFromCluster %s", path)
	entries := 0
	buf := make([]byte, directoryEntrySize)
	lfnSeq := ""
	lfnSeqCurr := 0 //lfn's numbered from 1
	var raw rawDirEnt
	var err EmmcError
	trust.Infof("readDirFromCluster: %s,%d", path, cnum)
	snum := f.sdcard.activePartition.clusterNumberToSector(1 /*xxx*/, cnum)
	fr, errFat := newFATDataReader(cnum, f.sdcard.activePartition, f.tranq, 0) //get root directory
	if errFat != EmmcOk {
		return nil, errFat
	}
	result := NewDir(f, path, snum, 32)
outer:
	for {
		curr := 0
		for curr < directoryEntrySize {
			r, err := fr.Read(buf[curr : directoryEntrySize-curr])
			if err == io.EOF {
				break outer
			}
			if err != nil {
				break outer
			}
			curr += r
		}
		if ok := raw.unpack(buf); !ok {
			break outer
		}
		entries++
		switch {
		case raw.name[0] == directoryEnd:
			f.tranq.DumpStats(false)
			break outer
		case raw.name[0] == directoryEntryDeleted:
			continue
		case raw.Attrib == directoryEntryLFN:
			lfn := longFilename(buf[0:directoryEntrySize])
			if lfn == nil {
				trust.Errorf("unable to understand long file name in directory")
				continue
			}
			if int(lfn.sequenceNumber) > lfnSeqCurr {
				lfnSeq = lfnSeq + lfn.segment
			} else {
				lfnSeq = lfn.segment + lfnSeq
			}
			lfnSeqCurr = int(lfn.sequenceNumber)
		default:
			nameLen := strlenWithTerminator(raw.name[:], ' ')
			extLen := strlenWithTerminator(raw.ext[:], ' ')
			shortName := string(raw.name[:nameLen])
			if extLen > 0 {
				shortName += "." + string(raw.ext[:extLen])
			}
			if len(shortName) == 0 {
				trust.Warnf("found a short name for a file that is empty!")
			}
			longName := shortName
			if lfnSeqCurr > 0 {
				longName = lfnSeq
			}
			lfnSeq = ""
			lfnSeqCurr = 0 // lfn's seqence numbers start at 1??
			result.addEntry(longName, &raw)
		}
	}
	if err == EmmcEOF {
		trust.Warnf("finished reading all the directory entries, " +
			" but shouldn't we have gotten a directory end entry?")
	}
	if err != EmmcOk {
		return nil, EmmcUnknown
	}
	return result, EmmcOk

}
