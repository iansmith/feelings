package main

import (
	"errors"
	"io"
	"lib/trust"
	"unsafe"
)

// fat data reader has three levels of cycling:
// 1.top level is the cluster id which is a chain through the FAT tables
// 2.each sector in the cluster
// 3.each byte of each sector
type fatDataReader struct {
	tranquil      bufferManager
	cluster       uint32
	sector        uint32
	sectorData    unsafe.Pointer // sectorSize
	current       uint32         // [0, sectorSize)
	finishedInit  bool
	size          uint32
	totalConsumed uint32
	partition     *fatPartition
}

func newFATDataReader(cluster uint32, partition *fatPartition, t bufferManager, size uint32) *fatDataReader {
	dr := &fatDataReader{
		cluster:   cluster,
		tranquil:  t,
		partition: partition,
		size:      size,
	}
	//we want to initialize the page data
	if e := dr.getMoreData(); e != sdOk {
		trust.Errorf("unable to setup the FAT data reader, can't get initial data %d", cluster)
		return nil
	}
	return dr
}

func (f *fatDataReader) Read(p []byte) (int, error) {
	if f.endOfClusterChain() { //our work here is done, already finished reading
		return 0, io.EOF
	}
	l := len(p)
	if l == 0 {
		return 0, nil //nothing to do, cause infinite loop?
	}
	atEOF := false
	isError := false
	result := 0
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
	if f.current == sectorSize {
		if f.endOfClusterChain() { //this is the EOF cause by no more data
			atEOF = true
			goto returnError
		}
		//deal with the case where we need another page
		ok := f.getMoreData()
		if ok != sdOk {
			isError = true
			goto returnError
		}
	}
	//everything looks ok... this is the happy path
	return result, nil
returnError:
	if isError {
		trust.Errorf("error occured reading the sector of FAT32")
		return 0, errors.New("need to return a better error code from read")
	}
	if atEOF {
		return 0, io.EOF
	}
	panic("unknown read state")
}

func (f *fatDataReader) endOfClusterChain() bool {
	if f.partition.isFAT16 {
		return f.cluster < 2 || f.cluster >= fat16EOCBoundary
	}
	return f.cluster < 2 || f.cluster >= fat32EOCBoundary
}

func (f *fatDataReader) getNextClusterInChain() (uint32, int) {

	if f.endOfClusterChain() {
		trust.Errorf("should not be calling getNextClusterInChain when already at end of chain")
		return f.cluster, sdOk
	}
	var next uint32

	//load the needed page
	distance := uintptr(f.cluster) << 2
	if f.partition.isFAT16 {
		distance = uintptr(f.cluster) << 1
	}
	sectorOfFAT := distance >> 9 // divide by sectorSize
	ptr, err := f.tranquil.PossiblyLoad(f.partition.fatOrigin + uint32(sectorOfFAT))
	if err != nil {
		trust.Errorf("error reading fat sector " + err.Error())
		return 0, sdError
	}
	offset := distance % sectorSize

	if f.partition.isFAT16 {
		base := (*uint16)(unsafe.Pointer(uintptr(ptr) + offset)) // <<1 is because 2 bytes per
		next = uint32(*base)
	} else {
		base := (*uint32)(unsafe.Pointer(uintptr(ptr) + offset)) // <<2 is because 4 bytes per
		next = *base
	}
	f.cluster = next
	if f.partition.isFAT16 {
		warnFAT16ChainValue(next)
	} else {
		warnFAT32ChainValue(next)
	}
	return f.cluster, sdOk
}

func (f *fatDataReader) getMoreData() int {
	// on first call, we just load the buffer we were asked to start with, otherwise
	// we are here because we need the _next_ blob
	if f.finishedInit == false {
		f.finishedInit = true
	} else {
		f.current = 0
		c, err := f.getNextClusterInChain()
		if err != sdOk {
			return err
		}
		f.cluster = c
	}
	if !f.endOfClusterChain() {
		//fetch the next page
		var err error
		f.sectorData, err = f.tranquil.PossiblyLoad(f.partition.clusterNumberToSector(f.cluster))
		if err != nil {
			trust.Errorf("unable to read data sector: %v", err.Error())
			return sdError
		}
	}
	return sdOk
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
