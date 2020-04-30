package main

import (
	"errors"
	"feelings/src/hardware/bcm2835"
	"io"
	"unsafe"
)

// fat data reader has three levels of cycling:
// 1.top level is the cluster id which is a chain through the FAT tables
// 2.each sector in the cluster
// 3.each byte of each sector
type fatDataReader struct {
	sdcard       *sdCardInfo
	tranquil     bufferManager
	cluster      uint32
	sector       uint32
	sectorData   unsafe.Pointer // sectorSize
	current      uint32         // [0, sectorSize)
	finishedInit bool
}

func newFATDataReader(cluster uint32, sdcard *sdCardInfo, t bufferManager) *fatDataReader {
	dr := &fatDataReader{
		cluster:  cluster,
		sdcard:   sdcard,
		tranquil: t,
	}
	//we want to initialize the page data
	if e := dr.getMoreData(); e != bcm2835.SDOk {
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
	//make the simple case fast
	if f.current+uint32(l) < sectorSize {
		for i := 0; i < l; i++ {
			p[i] = *((*uint8)(unsafe.Pointer(uintptr(f.sectorData) + uintptr(f.current) + uintptr(i))))
		}
		f.current += uint32(l)
		result = l
	} else {
		//this is the case of reading the remainder of this page
		remaining := sectorSize - f.current
		for i := 0; i < int(remaining); i++ {
			p[i] = *((*uint8)(unsafe.Pointer(uintptr(f.sectorData) + uintptr(f.current) + uintptr(i))))
		}
		f.current += remaining //makes it sectorSize
		result = int(remaining)
	}
	if f.current == sectorSize {
		//deal with the case where we need another page
		ok := f.getMoreData()
		if ok != bcm2835.SDOk {
			isError = true
			goto returnError
		}
	}
	//everything looks ok... this is the happy path
	return result, nil
returnError:
	if isError {
		return 0, errors.New("need to return a better error code from read")
	}
	if atEOF {
		return 0, io.EOF
	}
	panic("unknown read state")
}

func (f *fatDataReader) endOfClusterChain() bool {
	if f.sdcard.activePartition.isFat16 {
		return f.cluster < 2 || f.cluster >= fat16EOCBoundary
	}
	return f.cluster < 2 || f.cluster >= fat32EOCBoundary
}

func (f *fatDataReader) getNextClusterInChain() (uint32, int) {

	if f.endOfClusterChain() {
		errorMessage("should not be calling getNextClusterInChain when already at end of chain")
		return f.cluster, bcm2835.SDOk
	}
	var next uint32

	//load the needed page
	distance := uintptr(f.cluster) << 2
	if f.sdcard.activePartition.isFat16 {
		distance = uintptr(f.cluster) << 1
	}
	sectorOfFAT := distance >> 9 // divide by sectorSize
	ptr, err := f.tranquil.PossiblyLoad(f.sdcard.activePartition.fatOrigin + uint32(sectorOfFAT))
	if err != nil {
		errorMessage("error reading fat sector " + err.Error())
		return 0, bcm2835.SDError
	}
	offset := distance % sectorSize

	if f.sdcard.activePartition.isFat16 {
		base := (*uint16)(unsafe.Pointer(uintptr(ptr) + uintptr(offset))) // <<1 is because 2 bytes per
		next = uint32(*base)
	} else {
		base := (*uint32)(unsafe.Pointer(uintptr(ptr) + uintptr(offset))) // <<2 is because 4 bytes per
		next = *base
	}
	f.cluster = next
	if f.sdcard.activePartition.isFat16 {
		warnFAT16ChainValue(next)
	} else {
		warnFAT32ChainValue(next)
	}
	return f.cluster, bcm2835.SDOk
}

func (f *fatDataReader) getMoreData() int {
	// on first call, we just load the buffer we were asked to start with, otherwise
	// we are here because we need the _next_ blob
	if f.finishedInit == false {
		f.finishedInit = true
	} else {
		f.current = 0
		c, err := f.getNextClusterInChain()
		if err != bcm2835.SDOk {
			return err
		}
		f.cluster = c
	}
	if !f.endOfClusterChain() {
		//fetch the next page
		var err error
		f.sectorData, err = f.tranquil.PossiblyLoad(f.sdcard.clusterNumberToSector(f.cluster))
		if err != nil {
			errorMessage("unable to read data sector: " + err.Error())
			return bcm2835.SDError
		}
	}
	return bcm2835.SDOk
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
		warnMessage("Unusual value found in FAT32 chain, assuming end-of-cluster: ", v)
	case fat32formatFiller:
		warnMessage("Found format filler in the FAT32 chain, assuming end-of-cluster: ", v)
	case fat32BadSector:
		warnMessage("Ignoring bad sector value in FAT32 chain, assuming end-of-cluster: ", v)
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
		warnMessage("Unusual value found in FAT16 chain, assuming end-of-cluster: ", v)
	case fat16formatFiller:
		warnMessage("Found format filler in the FAT16 chain, assuming end-of-cluster: ", v)
	case fat16BadSector:
		warnMessage("Ignoring bad sector value in FAT16 chain, assuming end-of-cluster: ", v)
	}
}

//func open(fullyQualifiedRaw string) (*os.File, error) {
//	path := filepath.Clean(fullyQualifiedRaw)
//	path = strings.TrimSpace(path)
//	if path[0] != os.PathSeparator {
//		return nil, errors.New("only handles fully qualified path:" + fullyQualifiedRaw)
//	}
//	remainder := path[1:]
//	left := ""
//	for remainder != "" {
//		pieces := strings.SplitN(current, string(os.PathSeparator), 2)
//		if len(pieces) == 1 {
//			left = pieces[0]
//			remainder = ""
//		} else {
//			left = pieces[0]
//			remainder = pieces[1]
//		}
//
//	}
//
//}
