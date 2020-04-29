package main

import (
	"bytes"
	"errors"
	"feelings/src/golang/encoding/binary"
	"feelings/src/golang/fmt"
	rt "feelings/src/tinygo_runtime"
	"unicode/utf16"
	"unsafe"
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
const showErrors = true
const showInfo = true
const showCommands = false
const fat32EOCBoundary = 0xFFFFFF8 //anything at or above this is EOC
const fat16EOCBoundary = 0xFFF8    //anything at or above this is EOC
const showWarn = true
const sizeOfPackedBpbShared = 0x24
const mbrUnusedSize = 446
const sizeOfPackedPartitionInfo = 0x10
const directoryEnd = 0x0
const directoryEntryDeleted = 0xE5
const directoryEntryLFN = 0xF
const attributeSubdirectory = 0x10
const lowercaseName = 0x10
const lowercaseExt = 0x8

//var partitionlba = uint32(0)

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

type fatDir struct {
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

// if n==0 returns false
func compareBytewise(s1 []uint8, s2 string, n int) bool {
	if len(s1) == 0 || len(s2) == 0 || n == 0 {
		return false
	}
	count := 0
	//if len(s1) == len(s2) {
	//	rt.MiniUART.WriteString("candidate " + s2 + "\n")
	//	rt.MiniUART.WriteString("versus " + string(s1) + "\n")
	//}
	for count < n && count < len(s1) && count < len(s2) {
		if s1[count] != s2[count] {
			return false
		}
		count++
	}
	if count != n {
		return false
	}
	return true
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

/**
 * Get the starting LBA address of the first partition
 * so that we know where our FAT file system starts, and
 * read that volume's BIOS Parameter Block
 */
func fatGetPartition(buffer []uint8) *sdCardInfo { //xxx should be passed in setup by Init
	sdCard := sdCardInfo{}
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
			errorMessage("unable to read MBR: " + err.Error())
			return nil
		}
		if mbr.Signature != 0xaa55 {
			errorMessage("bad magic number in MBR")
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
			errorMessage("did not find a BIOS Parameter Block")
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
		infoMessage("FAT32 Partition Info FAT Origin : ", sdCard.activePartition.fatOrigin)
		sdCard.activePartition.dataSectors = bpbFull.Ts32 - uint32(bpbFull.Rsc) - (bpbFull.fat32.FATSize32 * uint32(bpbFull.Nf))
		//sdCard.partition.dataSectors = bpb->TotalSectors32 - bpb->ReservedSectorCount - (bpb->FSTypeData.fat32.FATSize32 * bpb->NumFATs);
		infoMessage("FAT32 Partition Info Total Sectors : ", bpbFull.Ts32)
		infoMessage("FAT32 Partition Info Data Sectors : ", sdCard.activePartition.dataSectors)
		if bpbFull.fat32.BootSig != 0x29 {
			errorMessage("FAT32 volume has bad boot signature: ", uint32(ext32.BootSig))
			return nil
		}
		fmt.Printf("FAT32 Volume Label: '%s', ID: %08x\n",
			string(ext32.volumeLabel[:]), ext32.VolumeID) //xxx because of problem in tinyo reflection with bpbfull
		sdCard.activePartition.fatSize = (uint32(bpbFull.Nf) * uint32(bpbFull.fat32.FATSize32))

		if bpbFull.fat32.fileSystemType[0] != 'F' ||
			bpbFull.fat32.fileSystemType[1] != 'A' ||
			bpbFull.fat32.fileSystemType[2] != 'T' {
			errorMessage("Wrong filesystem type (not FAT)")
		}
	} else {
		// FAT16
		sdCard.activePartition.rootCluster = 2 // Hold partition root cluster, FAT16 always start at 2
		sdCard.activePartition.fatOrigin = sdCard.activePartition.unusedSectors + (uint32(bpbFull.Nf) * uint32(bpbFull.Spf16)) + 1
		// data sectors x sectorsize = capacity ... I have check this on PC and gives right calc
		sdCard.activePartition.dataSectors = bpbFull.Ts32 - (uint32(bpbFull.Nf) * uint32(bpbFull.Spf16)) - 33
		infoMessage("FAT16 reserved sectors: ", uint32(bpbFull.Rsc))
		infoMessage("FAT16 FAT origin sector: ", uint32(sdCard.activePartition.fatOrigin))
		fmt.Printf("FAT32 Volume Label: %s, ID: %08x\n",
			bpbFull.fat16.volumeLabel, bpbFull.fat16.VolumeID)
		if bpbFull.fat16.BootSig != 0x29 {
			errorMessage("FAT16 volume has bad boot signature: ", uint32(bpbFull.fat16.BootSig))
			return nil
		}
		if bpbFull.fat16.fileSystemType[0] != 'F' ||
			bpbFull.fat16.fileSystemType[1] != 'A' ||
			bpbFull.fat16.fileSystemType[2] != 'T' {
			errorMessage("Wrong filesystem type (not FAT)")
			return nil
		}
		sdCard.activePartition.fatSize = (uint32(bpbFull.Nf) * uint32(bpbFull.Spf16))
		sdCard.activePartition.isFat16 = true
	}
	return &sdCard
}

func (s *sdCardInfo) clusterNumberToSector(clusterNumber uint32) uint32 {
	return ((clusterNumber - 2) * s.activePartition.sectorsPerCluster) + s.activePartition.fatOrigin + s.activePartition.fatSize
}

//func getFirstSector(clusterNumber uint32, sectorPerCluster uint32, firstDataSector uint32) uint32 {
//	return (((clusterNumber - 2) * sectorPerCluster) + firstDataSector)
//}

func locateFATEntry(filename string, info *sdCardInfo) error {
	startSector := info.clusterNumberToSector(info.activePartition.rootCluster)
	read, _ := sdReadblock(startSector, 1)
	if read == 0 {
		return errors.New("unable to read start sector")
	}

	return nil
}

/**
 * Find a file in root directory entries, root directory is exactly 1 sector?
 */
func fatGetCluster(fn string, sdcard *sdCardInfo) uint32 {
	var items uint32 //only set by fat16

	rootSector := sdcard.clusterNumberToSector(sdcard.activePartition.rootCluster)

	if sdcard.activePartition.isFat16 {
		panic("isfat16 not implemented yet")
	}
	// load the root directory
	read, rootDir := sdReadblock(rootSector, items/sectorSize+1)
	infoMessage("read block:", items/sectorSize+1, uint32(read))
	if read != 0 {
		for dptr := uintptr(0); dptr < uintptr(sectorSize); dptr = dptr + directoryEntrySize {
			//rt.MiniUART.Dump(unsafe.Pointer(&rootDir[0]))
			dirEntryBuffer := rootDir[dptr : dptr+directoryEntrySize]
			dir := newFATDir()
			if !dir.unpack(dirEntryBuffer) {
				return 0
			}
			if dir.name[0] == directoryEnd {
				infoMessage("bailing out because found end of list entry @ position: ", uint32(dptr/directoryEntrySize))
				break
			}
			if dir.name[0] == directoryEntryDeleted {
				infoMessage("skipping because deleted at @ position: ", uint32(dptr/directoryEntrySize))
				continue
			}
			if dir.Attrib == directoryEntryLFN {
				infoMessage("found long filename @: ", uint32(dptr/directoryEntrySize))
				lfn := longFilename(dirEntryBuffer)
				infoMessage("long filename segment found: " + lfn.segment)
				continue
			}
		}
		return 0
	} else {
		errorMessage("Unable to load root directory")
		//fallthrough
	}
	return 0
}

func newFATDir() *fatDir {
	result := &fatDir{}
	return result
}

func (f *fatDir) unpack(buffer []uint8) bool {
	buf := bytes.NewBuffer(buffer)
	//rt.MiniUART.Dump(unsafe.Pointer(&buf.Bytes()[0]))

	err := binary.Read(buf, binary.LittleEndian, f)
	if err != nil {
		errorMessage("failed to read binary data for directory entry: " + err.Error())
		return false
	}
	copy(f.name[:], buffer[0:8])
	copy(f.ext[:], buffer[8:11])
	return true
}

/**
 * Read a file into memory
 */
func fatReadfile(cluster uint32, sdcard *sdCardInfo) []byte {

	// dump important properties
	infoMessage("FAT Bytes per Sector: ", sdcard.activePartition.bytesPerSector)
	infoMessage("FAT Sectors per Cluster: ", sdcard.activePartition.sectorsPerCluster)
	infoMessage("FAT Reserved Sectors Count: ", sdcard.activePartition.reservedSectorCount)

	//read, fatTable := sdReadblock(sdcard.activePartition.firstDataSector, 1)
	read, fatTable := sdReadblock(0x820, 0x3f1)
	if read == 0 {
		errorMessage("failed to read FAT")
		return nil
	}

	eoc := uint32(fat32EOCBoundary)
	unusual0 := uint32(0xFFFFFFF0)
	unusual1 := uint32(0xFFFFFFF1)
	unusual2 := uint32(0xFFFFFFF2)
	unusual3 := uint32(0xFFFFFFF3)
	unusual4 := uint32(0xFFFFFFF4)
	unusual5 := uint32(0xFFFFFFF5)
	formatFiller := uint32(0xFFFFFFF6)
	badSector := uint32(0xFFFFFFF7)

	if sdcard.activePartition.isFat16 {
		eoc = uint32(fat16EOCBoundary)
		unusual0 = uint32(0xFFF0)
		unusual1 = uint32(0xFFF1)
		unusual2 = uint32(0xFFF2)
		unusual3 = uint32(0xFFF3)
		unusual4 = uint32(0xFFF4)
		unusual5 = uint32(0xFFF5)
		formatFiller = uint32(0xFFF6)
		badSector = uint32(0xFFF7)
	}
	result := []byte{}
	for cluster > 1 && cluster < eoc {
		switch cluster {
		case unusual0, unusual1, unusual2, unusual3, unusual4, unusual5:
			warnMessage("unusual byte value found in cluster chain:", cluster)
		case formatFiller:
			warnMessage("unexpected use of reserved value in cluster chain:", cluster)
		case badSector:
			warnMessage("ignoring bad sector value cluster chain:", cluster)
		}
		read, buffer := sdReadblock(((cluster-2)*uint32(sdcard.activePartition.sectorsPerCluster))+(2*0x3f1)+0x821,
			sdcard.activePartition.sectorsPerCluster)
		if read == 0 {
			errorMessage("unable to read cluster for file", (cluster-2)*uint32(sdcard.activePartition.sectorsPerCluster)+(2*0x3f1))
			return nil
		}
		result = append(result, buffer...)
		// get the next cluster in chain
		var next uint32
		if sdcard.activePartition.isFat16 {
			//consider this fat an array of uint16
			base := (*uint16)(unsafe.Pointer(&fatTable[0]))
			base = (*uint16)(unsafe.Pointer(uintptr(unsafe.Pointer(base)) + uintptr(((cluster) << 1)) - 512)) // <<1 is because 2 bytes per
			next = uint32(*base)
		} else {
			//consider this fat an array of uint32
			base := (*uint32)(unsafe.Pointer(&fatTable[0]))
			base = (*uint32)(unsafe.Pointer(uintptr(unsafe.Pointer(base)) + uintptr(((cluster) << 2)) - /*512*/ 0)) // <<2 is because 4 bytes per
			next = *base
		}
		cluster = next
		infoMessage("next cluster is ", cluster)
	}
	return result
}

func errorMessage(s string, values ...uint32) {
	if showErrors {
		fmt.Printf("ERROR:" + s + "\n")
	}
	for _, v := range values {
		fmt.Printf("%08x ", v)
	}
	fmt.Printf("\n")
}

func infoMessage(s string, values ...uint32) {
	if showInfo {
		fmt.Printf("INFO " + s)
		for _, v := range values {
			fmt.Printf("%08x ", v)
		}
		fmt.Printf("\n")
	}
}
func warnMessage(s string, values ...uint32) {
	if showWarn {
		rt.MiniUART.WriteString("WARN " + s)
		for _, v := range values {
			rt.MiniUART.Hex32string(v)
		}
		rt.MiniUART.WriteString("\n")
	}
}

func longFilename(in []byte) *fat32LFN {
	buffer := bytes.NewBuffer(in)
	raw := &rawFat32LFN{}
	if err := binary.Read(buffer, binary.LittleEndian, raw); err != nil {
		errorMessage("Unable to decode binary format for long filename" + err.Error())
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
		warnMessage("badly formed fat32 long filename record: ",
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
		warnMessage("badly formed fat32 long filename record, sequence number: ",
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
			warnMessage("bad utf-16 in surrogate pair encoding at end of name, ignored")
			continue
		}
		r2, pair := ucs2ToRune(holdChars[i+1])
		if !pair {
			warnMessage("bad utf-16 in surrogate pair encoding in name at character, ignored ", uint32(i+1))
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

func iterate(buffer []byte, fn string, temp uint32) uint32 {
	for dptr := uintptr(0); dptr < uintptr(len(buffer)); dptr = dptr + directoryEntrySize {
		dirEntryBuffer := buffer[dptr : dptr+directoryEntrySize]
		dir := newFATDir()
		if !dir.unpack(dirEntryBuffer) {
			return 0
		}
		if dir.name[0] == 0 {
			//infoMessage("bailing out because found end of list entry @ position: ", uint32(dptr/directoryEntrySize))
			break
		}
		if dir.name[0] == 0xE5 {
			continue
		}
		if dir.Attrib == 0xF {
			lfn := longFilename(dirEntryBuffer)
			if lfn == nil {
				continue
			}
			warnMessage("long filename segment found: "+lfn.segment+" => sequence, sector:",
				uint32(lfn.sequenceNumber), temp)
			continue
		}
		nameLen := strlenWithTerminator(dir.name[:], ' ')
		extLen := strlenWithTerminator(dir.ext[:], ' ')

		if compareBytewise(append(dir.name[:nameLen], dir.ext[:extLen]...), fn, nameLen+extLen) {
			start := uint32(dir.FirstClusterHi)<<16 | uint32(dir.FirstClusterLo)
			infoMessage("FAT File "+fn+" starts at cluster ", (uint32(dir.FirstClusterLo)<<16)|uint32(dir.FirstClusterLo))
			// if so, return starting cluster
			return start
		}
	}
	return 0
}

func unpackPartitionBuffer(buffer []byte, initialPadding int, start int, end int, part *partitionInfo) error {
	pbuf := bytes.NewBuffer(buffer[initialPadding+start : initialPadding+end])
	if err := binary.Read(pbuf, binary.LittleEndian, part); err != nil {
		errorMessage("unable to read partition descriptor of partition 1:" + err.Error())
		return err
	}
	return nil
}
