package main

import (
	"feelings/src/golang/bytes"
	"feelings/src/golang/encoding/binary"
	"feelings/src/golang/io"
	"feelings/src/golang/path/filepath"
	"feelings/src/golang/strings"
	"feelings/src/golang/unicode/utf16"
	"feelings/src/lib/trust"
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
const showCommands = false
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
func fatGetPartition(buffer []uint8) *sdCardInfo { //xxx should be passed in setup by Init
	sdCard := sdCardInfo{
		activePartition: &fatPartition{},
	}
	read, buffer := sdReadblock(0, 1)
	if read == 0 {
		return nil
	}
	//we need to check and see if boot sector...
	bpb := biosParamBlockShared{}
	if !bpb.unpack(buffer) {
		return nil
	}
	if bpb.jump[0] != 0xE9 && bpb.jump[0] != 0xEB {
		mbrData := bytes.NewBuffer(buffer)
		var mbr mbrInfo
		if err := binary.Read(mbrData, binary.LittleEndian, &mbr); err != nil {
			trust.Errorf("unable to read MBR: %v ", err.Error())
			return nil
		}
		if mbr.Signature != 0xaa55 {
			trust.Errorf("bad magic number in MBR")
			return nil
		}
		if err := unpackPartitionBuffer(buffer, mbrUnusedSize, 0, sizeOfPackedPartitionInfo, &mbr.Partition1); err != nil {
			return nil
		}
		if err := unpackPartitionBuffer(buffer, mbrUnusedSize, sizeOfPackedPartitionInfo, 2*sizeOfPackedPartitionInfo, &mbr.Partition2); err != nil {
			return nil
		}
		if err := unpackPartitionBuffer(buffer, mbrUnusedSize, sizeOfPackedPartitionInfo*2, 3*sizeOfPackedPartitionInfo, &mbr.Partition3); err != nil {
			return nil
		}
		if err := unpackPartitionBuffer(buffer, mbrUnusedSize, sizeOfPackedPartitionInfo*3, 4*sizeOfPackedPartitionInfo, &mbr.Partition4); err != nil {
			return nil
		}
		sdCard.activePartition.unusedSectors = mbr.Partition1.FirstSector // FAT16 needs this value so hold it
		//try again for BPB at firstDataSector
		read, buffer = sdReadblock(mbr.Partition1.FirstSector, 1)
		if read == 0 {
			return nil
		}
		bpb = biosParamBlockShared{}
		if !bpb.unpack(buffer) {
			return nil
		}
		if bpb.jump[0] != 0xE9 && bpb.jump[0] != 0xEB {
			trust.Errorf("did not find a BIOS Parameter Block")
			return nil
		}
	}
	// we have only read the shared part, so read extension as appropriate
	var ext16 *biosParamBlockFat16Extension
	var ext32 *biosParamBlockFat32Extension
	if bpb.Spf16 > 0 && bpb.NumRootEntries > 0 {
		ext16 := &biosParamBlockFat16Extension{}
		if ext16.unpack(buffer[sizeOfPackedBpbShared:]) {
			return nil
		}
	} else {
		ext32 = &biosParamBlockFat32Extension{}
		if !ext32.unpack(buffer[sizeOfPackedBpbShared:]) {
			return nil
		}
	}
	bpbFull := newBIOSParamBlock(&bpb, ext16, ext32)
	sdCard.activePartition.bytesPerSector = uint32(bpbFull.BytesPerSector) // Bytes per sector on partition
	sdCard.activePartition.sectorsPerCluster = uint32(bpbFull.Spc)         // Hold the sector per cluster count
	sdCard.activePartition.reservedSectorCount = uint32(bpbFull.Rsc)       // Hold the reserved sector count

	if !bpbFull.isFat16 { // Check if FAT16/FAT32
		// FAT32
		sdCard.activePartition.rootCluster = bpbFull.fat32.RootCluster // Hold partition root cluster
		sdCard.activePartition.fatOrigin = uint32(bpbFull.Rsc) + bpbFull.Hs + sdCard.activePartition.unusedSectors
		trust.Infof("FAT32 Partition Info FAT Origin : 0x%x ", sdCard.activePartition.fatOrigin)
		sdCard.activePartition.dataSectors = bpbFull.Ts32 - uint32(bpbFull.Rsc) - (bpbFull.fat32.FATSize32 * uint32(bpbFull.Nf))
		//sdCard.partition.dataSectors = bpb->TotalSectors32 - bpb->ReservedSectorCount - (bpb->FSTypeData.fat32.FATSize32 * bpb->NumFATs);
		trust.Infof("FAT32 Partition Info Total Sectors : 0x%x ", bpbFull.Ts32)
		trust.Infof("FAT32 Partition Info Data Sectors : 0x%x", sdCard.activePartition.dataSectors)
		if bpbFull.fat32.BootSig != 0x29 {
			trust.Errorf("FAT32 volume has bad boot signature: %v", uint32(bpbFull.fat32.BootSig))
			return nil
		}
		trust.Infof("FAT32 Volume Label: '%s', ID: 0x%08x\n",
			string(bpbFull.fat32.volumeLabel[:]), bpbFull.fat32.VolumeID) //xxx because of problem in tinyo reflection with bpbfull
		sdCard.activePartition.fatSize = uint32(bpbFull.Nf) * bpbFull.fat32.FATSize32

		if bpbFull.fat32.fileSystemType[0] != 'F' ||
			bpbFull.fat32.fileSystemType[1] != 'A' ||
			bpbFull.fat32.fileSystemType[2] != 'T' {
			trust.Errorf("Wrong filesystem type (not FAT)")
		}
	} else {
		// FAT16
		sdCard.activePartition.rootCluster = 2 // Hold partition root cluster, FAT16 always start at 2
		sdCard.activePartition.fatOrigin = sdCard.activePartition.unusedSectors + (uint32(bpbFull.Nf) * uint32(bpbFull.Spf16)) + 1
		// data sectors x sectorsize = capacity ... I have check this on PC and gives right calc
		sdCard.activePartition.dataSectors = bpbFull.Ts32 - (uint32(bpbFull.Nf) * uint32(bpbFull.Spf16)) - 33
		trust.Infof("FAT16 reserved sectors: 0x%x", uint32(bpbFull.Rsc))
		trust.Infof("FAT16 FAT origin sector: 0x%x", sdCard.activePartition.fatOrigin)
		trust.Infof("FAT32 Volume Label: %s, ID: 0x%08x\n",
			bpbFull.fat16.volumeLabel, bpbFull.fat16.VolumeID)
		if bpbFull.fat16.BootSig != 0x29 {
			trust.Errorf("FAT16 volume has bad boot signature:  %x", uint32(bpbFull.fat16.BootSig))
			return nil
		}
		if bpbFull.fat16.fileSystemType[0] != 'F' ||
			bpbFull.fat16.fileSystemType[1] != 'A' ||
			bpbFull.fat16.fileSystemType[2] != 'T' {
			trust.Errorf("Wrong filesystem type (not FAT)")
			return nil
		}
		sdCard.activePartition.fatSize = uint32(bpbFull.Nf) * uint32(bpbFull.Spf16)
		sdCard.activePartition.isFAT16 = true
	}
	return &sdCard
}

func (f *fatPartition) clusterNumberToSector(clusterNumber uint32) uint32 {
	return ((clusterNumber - 2) * f.sectorsPerCluster) + f.fatOrigin + f.fatSize
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
	inodeMap  map[string]uint64
	nextInode uint64
}

func NewFAT32Filesystem(tranq *Tranquil, sdcard *sdCardInfo) *FAT32Filesystem {
	return &FAT32Filesystem{
		tranq:     tranq,
		sdcard:    sdcard,
		inodeMap:  make(map[string]uint64),
		nextInode: 1,
	}
}
func (f *FAT32Filesystem) NewInode() uint64 {
	result := f.nextInode
	f.nextInode++
	return result
}

// ReadDir returns a list of dirents into the already allocated structure given by
// it's parameter.   Path should already have been cleaned.
func (f *FAT32Filesystem) ReadDir(path string, entries []*DirEnt) *PosixError {
	panic("not implemented")
}

// OpenDir returns a directory pointer that you can then call ReadDir on, or it returns null.
func (f *FAT32Filesystem) openRootDir() (*Dir, *PosixError) {
	return f.readDirFromSector("/", f.sdcard.activePartition.rootCluster)
}

func (f *FAT32Filesystem) Open(path string) (io.Reader, *PosixError) {
	entry, err := f.resolvePath(path, true)
	if err != nil {
		return nil, err
	}
	trust.Infof("resolved %s: %+v", entry.Name, entry)
	reader := newFATDataReader(uint32(entry.firstClusterHi)*256+uint32(entry.firstClusterLo),
		f.sdcard.activePartition, f.tranq, entry.Size)
	if reader == nil {
		trust.Errorf("unable to create new FATDataReader! need better error\n")
		return nil, EUnknown // xxxx errors.New("should be the correct error here")
	}
	return reader, nil
}

func (f *FAT32Filesystem) openDirFromEntry(entry *DirEnt) (*Dir, *PosixError) {
	trust.Debugf("openDirFromEntry %s", entry.Name)
	path := filepath.Clean(entry.Path)
	return f.readDirFromSector(path, uint32(entry.firstClusterHi)*256+uint32(entry.firstClusterLo))
}

func (f *FAT32Filesystem) resolvePath(path string, isLast bool) (*DirEnt, *PosixError) {
	trust.Debugf("resolve path %s, %v", path, isLast)
	path = filepath.Clean(path)
	dirPath, file := filepath.Split(path)
	var dir *Dir
	var err *PosixError
	var entry *DirEnt

	if dirPath == "/" {
		dir, err = f.openRootDir()
		if err != nil {
			return nil, err
		}
	} else {
		entry, err = f.resolvePath(dirPath, false)
		if err != nil {
			return nil, err
		}
		dir, err = f.openDirFromEntry(entry)
		if err != nil {
			return nil, err
		}
	}
	file = strings.ToUpper(file)
	for _, e := range dir.contents {
		entryName := strings.ToUpper(e.Name)
		if entryName == file {
			if !isLast && !e.IsDir {
				return nil, ENoEnt
			}
			return &e, nil
		}
	}
	return nil, ENoEnt
}

func (f *FAT32Filesystem) readDirFromSector(path string, sector uint32) (*Dir, *PosixError) {
	entries := 0
	buf := make([]byte, directoryEntrySize)
	lfnSeq := ""
	var err error
	var r int
	lfnSeqCurr := 0 //lfn's numbered from 1
	var raw rawDirEnt
	fr := newFATDataReader(sector, f.sdcard.activePartition, f.tranq, 0) //get root directory
	result := NewDir(f, path, sector, 32)
outer:
	for {
		curr := 0
		for curr < directoryEntrySize {
			//fmt.Printf("reading entry %d, byte %d\n", entries, curr)
			r, err = fr.Read(buf[curr : directoryEntrySize-curr])
			if err == io.EOF {
				break outer
			}
			if err != nil {
				trust.Errorf("unknown error caught: %v", err.Error())
				break outer
			}
			curr += r
		}
		if ok := raw.unpack(buf); !ok {
			trust.Errorf("unable to unpack directory: %v ", err.Error())
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
			lfnSeqCurr = 0 // lfn's seqence numbers start at 1
			result.addEntry(longName, &raw)
		}
	}
	if err == io.EOF {
		trust.Warnf("finished reading all the directory entries, but shouldn't we have gotten a directory end entry?")
	}
	if err != nil {
		return nil, EUnknown
	}
	return result, nil
}
