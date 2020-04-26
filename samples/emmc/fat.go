package main

import (
	"bytes"
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

var partitionlba = uint32(0)

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

type fat32LFN struct { //used after we confirmed the raw values are ok
	sequenceNumber byte
	isLast         bool
	segment        string
	firstClusterLo uint16
}

type BIOSParamBlock struct { //in the Volume Boot Record
	jump  [3]uint8  //0x0
	oem   [8]uint8  //0x3
	Bps0  uint8     //0xB
	Bps1  uint8     //0xC
	Spc   uint8     //0xD sectors per clusters
	Rsc   uint16    //0xE  reserved sector count
	Nf    uint8     //0x10  number of fats
	Nr0   uint8     //0x11  number root entries 0
	Nr1   uint8     //0x12 number root entries 1
	Ts16  uint16    //0x13 total sectors
	Media uint8     //0x15  media descriptors
	Spf16 uint16    //0x16 sectors per fat
	Spt   uint16    //0x18  sectors per track
	Nh    uint16    //0x1A  number of heads
	Hs    uint32    //0x1C  hidden sectors
	Ts32  uint32    //0x20
	Spf32 uint32    //0x24  sectors per fat
	Flg   uint32    //0x28 flags and version (last two)
	Rc    uint32    //0x2C  root cluster
	vol   [6]uint8  //0x30
	fst   [8]uint8  //0x36  file system type
	dmy   [20]uint8 //0x3E
	fst2  [8]uint8  // 0x52 file system type
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
func fatGetPartition(buffer []uint8) *BIOSParamBlock {
	read, buffer := sdReadblock(0, 1)
	if read == 0 {
		return nil
	}
	if buffer[510] != 0x55 || buffer[511] != 0xAA {
		errorMessage("bad magic number in MBR")
		return nil
	}
	//// check partition type
	if buffer[0x1C2] != 0xE /*FAT16 LBA*/ && buffer[0x1C2] != 0xC /*FAT32 LBA*/ && buffer[0x1C2] != 0xB {
		errorMessage("Wrong partition type")
		return nil
	}
	partitionlba = uint32(buffer[0x1C6]) + (uint32(buffer[0x1C7]) << 8) + (uint32(buffer[0x1C8]) << 16) +
		(uint32(buffer[0x1C9]) << 24)

	read, buffer = sdReadblock(partitionlba, 1)
	if read == 0 {
		errorMessage("Unable to read boot record")
		return nil
	}
	bpb := newBIOSParamBlock()
	if !bpb.unpack(buffer) {
		return nil
	}
	if bpb.Spf16 > 0 && bpb.Rsc > 0 {
		errorMessage("fat16 should not have reserved sectors")
	}
	if !(bpb.fst[0] == 'F' && bpb.fst[1] == 'A' && bpb.fst[2] == 'T') &&
		!(bpb.fst2[0] == 'F' && bpb.fst2[1] == 'A' && bpb.fst2[2] == 'T') {
		errorMessage("ERROR: Unknown file system type")
		return nil
	}
	buffer = nil //safety
	return bpb
}

/**
 * Find a file in root directory entries, root directory is exactly 1 sector?
 */
func fatGetCluster(fn string, bpb *BIOSParamBlock) uint32 {
	var root_sec, s uint32

	var size uint32
	if bpb.Spf16 != 0 {
		size = uint32(bpb.Spf16)
	} else {
		size = bpb.Spf32
	}
	size = size * uint32(bpb.Nf)
	root_sec = size + uint32(bpb.Rsc)
	s = (uint32(bpb.Nr0) + (uint32(bpb.Nr1) << 8)) * /*uint32(unsafe.Sizeof(fatDir))*/ directoryEntrySize
	if bpb.Spf16 == 0 { //adjust for fat32?
		root_sec += (bpb.Rc - 2) * uint32(bpb.Spc)
		infoMessage("root cluster:", bpb.Rc, uint32(bpb.Spc))
	}
	root_sec += partitionlba
	infoMessage("root sector:", root_sec)
	infoMessage("numerator:", s)

	// load the root directory
	read, rootDir := sdReadblock(root_sec, s/512+1)
	infoMessage("read block:", s/512+1, uint32(read), (uint32(bpb.Nr0) + (uint32(bpb.Nr1) << 8)))
	if read != 0 {
		for dptr := uintptr(0); dptr < uintptr(512+512); dptr = dptr + directoryEntrySize {
			dirEntryBuffer := rootDir[dptr : dptr+directoryEntrySize]
			dir := newFATDir()
			if !dir.unpack(dirEntryBuffer) {
				return 0
			}
			if dir.name[0] == 0 {
				infoMessage("bailing out because found end of list entry @ position: ", uint32(dptr/directoryEntrySize))
				break
			}
			if dir.name[0] == 0xE5 {
				continue
			}
			if dir.Attrib == 0xF {
				lfn := longFilename(dirEntryBuffer)
				infoMessage("long filename segment found: " + lfn.segment)
				continue
			}
			nameLen := strlenWithTerminator(dir.name[:], ' ')
			extLen := strlenWithTerminator(dir.ext[:], ' ')

			if compareBytewise(append(dir.name[:nameLen], dir.ext[:extLen]...), fn, nameLen+extLen) {
				start := uint32(dir.FirstClusterHi)<<16 | uint32(dir.FirstClusterLo)
				infoMessage("FAT File "+fn+" starts at cluster ", (uint32(dir.FirstClusterHi)<<16)|uint32(dir.FirstClusterLo))
				// if so, return starting cluster
				return start
			}
		}
		return 0
	} else {
		errorMessage("Unable to load root directory")
		//fallthrough
	}
	return 0
}

func (b *BIOSParamBlock) unpack(buffer []uint8) bool {
	buf := bytes.NewBuffer(buffer)
	//rt.MiniUART.Dump(unsafe.Pointer(&buf.Bytes()[0]))

	//hack := BIOSParamBlock{}
	err := binary.Read(buf, binary.LittleEndian, b)
	if err != nil {
		errorMessage("failed to read binary data for bios param block: " + err.Error())
		return false
	}
	copy(b.jump[0:3], buffer[0:3])
	copy(b.oem[0:8], buffer[3:11])
	copy(b.vol[0:6], buffer[0x30:0x30+6])
	copy(b.fst[0:8], buffer[0x36:0x36+8])
	copy(b.fst2[0:8], buffer[0x52:0x52+8])
	infoMessage("unpacking bpb:", uint32(b.Nr0), uint32(b.Nr1))
	return true
}

func newBIOSParamBlock() *BIOSParamBlock {
	result := BIOSParamBlock{}
	return &result
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
func fatReadfile(cluster uint32, bpb *BIOSParamBlock, partitionlba uint32) []byte {
	var data_sec, s uint32

	// find the LBA of the first data sector
	if bpb.Spf16 > 0 {
		data_sec = uint32(bpb.Spf16)
	} else {
		data_sec = bpb.Spf32
	}
	data_sec *= uint32(bpb.Nf)
	data_sec += uint32(bpb.Rsc)
	//data_sec=((bpb->spf16?bpb->spf16:bpb->spf32)*bpb->nf)+bpb->rsc;
	s = (uint32(bpb.Nr0) + (uint32(bpb.Nr1) << 8)) * directoryEntrySize
	//s = (bpb->nr0 + (bpb->nr1 << 8)) * sizeof(fatdir_t);
	if bpb.Spf16 > 0 {
		// adjust for FAT16
		data_sec += (s + 511) >> 9
	}
	// add partition LBA
	data_sec += partitionlba
	// dump important properties
	infoMessage("FAT Bytes per Sector: ", uint32(bpb.Bps0)+(uint32(bpb.Bps1)<<8))
	infoMessage("FAT Sectors per Cluster: ", uint32(bpb.Spc))
	infoMessage("FAT Number of FAT: ", uint32(bpb.Nf))
	spf := bpb.Spf32
	if bpb.Spf16 > 0 {
		spf = uint32(bpb.Spf16)
	}
	infoMessage("FAT Sectors per FAT: ", spf)
	infoMessage("FAT Reserved Sectors Count: ", uint32(bpb.Rsc))
	infoMessage("FAT First data sector: ", data_sec)
	// load FAT table
	result := []byte{}
	num := bpb.Spf32
	if bpb.Spf16 > 0 {
		num = uint32(bpb.Spf16)
	}
	read, fatTable := sdReadblock(partitionlba+1+uint32(bpb.Rsc), num)
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

	if bpb.Spf16 > 0 {
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
	for cluster > 1 && cluster < eoc {
		switch cluster {
		case unusual0, unusual1, unusual2, unusual3, unusual4, unusual5:
			warnMessage("unusual byte value found in cluster chain:", cluster)
		case formatFiller:
			warnMessage("unexpected use of reserved value in cluster chain:", cluster)
		case badSector:
			warnMessage("ignoring bad sector value cluster chain:", cluster)
		}
		read, buffer := sdReadblock((cluster-2)*uint32(bpb.Spc)+data_sec, uint32(bpb.Spc))
		if read == 0 {
			errorMessage("unable to read cluster for file")
			return nil
		}
		result = append(result, buffer...)
		// get the next cluster in chain
		var next uint32
		if bpb.Spf16 > 0 {
			//consider this fat an array of uint16
			base := (*uint16)(unsafe.Pointer(&fatTable[0]))
			base = (*uint16)(unsafe.Pointer(uintptr(unsafe.Pointer(base)) + uintptr((cluster << 1)) - 512)) // <<1 is because 2 bytes per
			next = uint32(*base)
		} else {
			//consider this fat an array of uint32
			base := (*uint32)(unsafe.Pointer(&fatTable[0]))
			base = (*uint32)(unsafe.Pointer(uintptr(unsafe.Pointer(base)) + uintptr((cluster << 2)) - /*512*/ 0)) // <<2 is because 4 bytes per
			next = *base
		}
		cluster = next
		infoMessage("next cluster is ", cluster)
	}
	return result
}

func errorMessage(s string) {
	if showErrors {
		fmt.Printf("ERROR:" + s + "\n")
	}
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
	offset := 1
	for i := 0; i < 4; i++ {
		raw.name0[i] = uint16(in[(i*2)+1+offset])<<8 + uint16(in[i*2+offset])
	}
	offset = 0xE
	for i := 0; i < 6; i++ {
		raw.name1[i] = uint16(in[(i*2)+1+offset])<<8 + uint16(in[i*2+offset])
	}
	offset = 0x1C
	for i := 0; i < 2; i++ {
		raw.name1[i] = uint16(in[(i*2)+1+offset])<<8 + uint16(in[i*2+offset])
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
	for i := 0; i < 4; i++ {
		switch raw.name0[i] {
		case 0x0, 0xffff:
			done = true
			break outer0
		default:
			holdChars[count] = raw.name0[i]
			count++
		}
	}
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
			infoMessage("FAT File "+fn+" starts at cluster ", (uint32(dir.FirstClusterLo)<<16)|uint32(dir.FirstClusterLo)
			// if so, return starting cluster
			return start
		}
	}
	return 0
}
