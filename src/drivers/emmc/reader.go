package emmc

import (
	"fmt"
	"unsafe"

	"lib/trust"
)

const readerDebug = false
const readerDumpSector = false

// fat data reader has three levels of cycling:
// 1.top level is the cluster id which is a chain through the FAT tables
// 2.each sector in the cluster
// 3.each byte of each sector
type fatDataReader struct {
	tranquil      bufferManager
	cluster       clusterNumber
	sector        sectorNumber
	sectorData    unsafe.Pointer // sectorSize
	current       uint32         // [0, sectorSize)
	finishedInit  bool
	size          uint32
	totalConsumed uint32
	partition     *fatPartition
}

func newFATDataReader(cluster clusterNumber, partition *fatPartition,
	t bufferManager, size uint32) (*fatDataReader, EmmcError) {
	dr := &fatDataReader{
		cluster:   cluster,
		tranquil:  t,
		partition: partition,
		size:      size,
	}
	if readerDebug {
		trust.Debugf("new fat data reader: cluster=%d", cluster)
	}

	//we want to initialize the page data
	if e := dr.getMoreData(); e != EmmcOk {
		trust.Errorf("unable to setup the FAT data reader, "+
			"can't get initial data in cluster %d", cluster)
		return nil, EmmcBadInitialRead
	}
	return dr, EmmcOk
}

func (f *fatDataReader) Read(p []byte) (int, error) {
	if f.endOfClusterChain() { //our work here is done, already finished reading
		return 0, EmmcEOF
	}
	l := len(p)
	if l == 0 {
		return 0, EmmcNoBuffer //nothing to do, cause infinite loop?
	}
	atEOF := false
	isError := false
	result := 0
	var err EmmcError

	//this is the case where we want to stop, despite not being out of data... have to be
	//careful with size 0 directories...
	if f.size > 0 && f.totalConsumed == f.size {
		atEOF = true
		goto returnError
	}
	//check to make sure we don't read past the end of the file, but we don't know how
	//large a directory is (size==0)
	if f.size > 0 && l < sectorSize && l > int(f.size-f.totalConsumed) {
		l = int(f.size - f.totalConsumed) //clip it to the amount remaining w.r.t. size
	}
	//wants less than what is available on this page
	if readerDebug {
		trust.Debugf("READ: f.current=%d, buffersize=%d (sum=%d) compared to %d -- total consumed=%d",
			f.current, uint32(l), f.current+uint32(l), sectorSize, f.totalConsumed)
	}
	if f.current+uint32(l) < sectorSize {
		base := (uintptr)(unsafe.Pointer(uintptr(f.sectorData) + uintptr(f.current)))
		for i := 0; i < l; i++ {
			p[i] = *((*uint8)(unsafe.Pointer(base + uintptr(i))))
		}
		f.current += uint32(l)
		f.totalConsumed += uint32(l)
		result = l
	} else {
		//this is the case of reading the remainder of this page
		remaining := sectorSize - f.current
		for i := 0; i < int(remaining); i++ {
			p[i] = *((*uint8)(unsafe.Pointer(uintptr(f.sectorData) + uintptr(f.current) + uintptr(i))))
		}
		f.current += remaining //makes it sectorSize
		f.totalConsumed += uint32(remaining)
		result = int(remaining)
	}
	//at the edge of a page
	if readerDebug {
		trust.Debugf("READ? are we at end of page? f.current=%d compared to %d",
			f.current, sectorSize)
	}
	if f.current == sectorSize {
		if f.endOfClusterChain() { //this is the EOF cause by no more data
			atEOF = true
			goto returnError
		}
		//deal with the case where we need another page
		if readerDebug {
			trust.Debugf("READ: need more data!")
		}
		err = f.getMoreData()
		if err != EmmcOk {
			isError = true
			goto returnError
		}
	}
	//everything looks ok... this is the happy path
	return result, nil
returnError:
	if isError {
		return 0, err
	}
	if atEOF {
		return 0, EmmcEOF
	}
	panic("unknown read state")
}

func (f *fatDataReader) endOfClusterChain() bool {
	if f.partition.isFAT16 {
		return f.cluster < 2 || f.cluster >= fat16EOCBoundary
	}
	return f.cluster < 2 || f.cluster >= fat32EOCBoundary
}

func (f *fatDataReader) getNextClusterInChain() (clusterNumber, EmmcError) {

	if f.endOfClusterChain() {
		return f.cluster, EmmcOk
	}
	var next uint32

	//load the needed page
	distance := uintptr(f.cluster) << 2 //<<2 is because 4 bytes per
	if f.partition.isFAT16 {
		distance = uintptr(f.cluster) << 1 // <<1 is because 2 bytes per
	}
	sectorOfFAT := sectorNumber(distance >> 9) // divide by sectorSize
	ptr, err := f.tranquil.PossiblyLoad(f.partition.fatOrigin + sectorOfFAT)
	if err != EmmcOk {
		return 0, err
	}
	if readerDumpSector {
		fmt.Printf("--- sector %d ---\n", f.partition.fatOrigin+sectorOfFAT)
		for i := 0; i < 512; i += 32 {
			fmt.Printf("0x%03x:", i)
			for j := 0; j < 32; j++ {
				bptr := (*byte)(unsafe.Pointer(uintptr(ptr) + uintptr(i+j)))
				fmt.Printf("%02x", *bptr)
				if *bptr > 32 && *bptr < 127 {
					fmt.Printf("%c ", *bptr)
				} else {
					fmt.Printf("  ")
				}
				if j == 16 {
					fmt.Printf("-")
				}
				if j != 0 && j != 16 && j%4 == 0 {
					fmt.Printf(" ")
				}
			}
			fmt.Printf("\n")
		}
	}
	offset := distance % sectorSize
	if readerDebug {
		trust.Debugf("getNextClusterInChain: on the sector of fat offset is 0x%x", offset)
	}
	// XXX is this reading cluster numbers? are we sure?
	if f.partition.isFAT16 {
		base := (*uint16)(unsafe.Pointer(uintptr(ptr) + offset))
		next = uint32(*base)
	} else {
		base := (*uint32)(unsafe.Pointer(uintptr(ptr) + offset))
		next = *base
	}
	if readerDebug {
		trust.Debugf("getNextClusterInChain: next cluster is 0x%x", next)
	}

	f.cluster = clusterNumber(next)
	if f.partition.isFAT16 {
		warnFAT16ChainValue(next)
	} else {
		warnFAT32ChainValue(next)
	}
	return f.cluster, EmmcOk
}

func (f *fatDataReader) getMoreData() EmmcError {
	// on first call, we just load the buffer we were asked to start with, otherwise
	// we are here because we need the _next_ blob
	if f.finishedInit == false {
		f.finishedInit = true
		if readerDebug {
			trust.Debugf("getMoreData: finished init was false, so not reading start")
		}
	} else {
		f.current = 0
		c, err := f.getNextClusterInChain()
		if err != EmmcOk {
			return err
		}
		if readerDebug {
			trust.Debugf("getMoreData: next cluster is %d", c)
		}
		f.cluster = c
	}
	if !f.endOfClusterChain() {
		//fetch the next page
		var err EmmcError
		snum := f.partition.clusterNumberToSector(1 /*xxx*/, f.cluster)
		if readerDebug {
			trust.Debugf("getMoreData: Not at end of chain, so about try to load sector %d", snum)
		}
		f.sectorData, err = f.tranquil.PossiblyLoad(snum)
		if err != EmmcOk {
			trust.Errorf("unable to read data sector: %v", err.Error())
			return err
		}
	} //otherwise, at end of cluster chain
	return EmmcOk
}

const fat32Unusual0 = uint32(0xFFFFFFF0)
const fat32Unusual1 = uint32(0xFFFFFFF1)
const fat32Unusual2 = uint32(0xFFFFFFF2)
const fat32Unusual3 = uint32(0xFFFFFFF3)
const fat32Unusual4 = uint32(0xFFFFFFF4)
const fat32Unusual5 = uint32(0xFFFFFFF5)
const fat32formatFiller = uint32(0xFFFFFFF6)
const fat32BadSector = uint32(0xFFFFFFF7)

func warnFAT32ChainValue(v uint32) {
	switch v {
	case fat32Unusual0, fat32Unusual1, fat32Unusual2, fat32Unusual3, fat32Unusual4, fat32Unusual5:
		trust.Warnf("Unusual value found in FAT32 chain, assuming end-of-cluster: %d ", v)
	case fat32formatFiller:
		trust.Warnf("Found format filler in the FAT32 chain, assuming end-of-cluster: 0x%x", v)
	case fat32BadSector:
		trust.Warnf("Ignoring bad sector value in FAT32 chain, assuming end-of-cluster: ", v)
	}
}

const fat16Unusual0 = uint32(0xFFF0)
const fat16Unusual1 = uint32(0xFFF1)
const fat16Unusual2 = uint32(0xFFF2)
const fat16Unusual3 = uint32(0xFFF3)
const fat16Unusual4 = uint32(0xFFF4)
const fat16Unusual5 = uint32(0xFFF5)
const fat16formatFiller = uint32(0xFFF6)
const fat16BadSector = uint32(0xFFF7)

func warnFAT16ChainValue(v uint32) {
	switch v {
	case fat16Unusual0, fat16Unusual1, fat16Unusual2, fat16Unusual3, fat16Unusual4, fat16Unusual5:
		trust.Warnf("Unusual value found in FAT16 chain, assuming end-of-cluster: ", v)
	case fat16formatFiller:
		trust.Warnf("Found format filler in the FAT16 chain, assuming end-of-cluster: ", v)
	case fat16BadSector:
		trust.Warnf("Ignoring bad sector value in FAT16 chain, assuming end-of-cluster: ", v)
	}
}
