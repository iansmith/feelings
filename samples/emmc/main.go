package main

import (
	"feelings/src/golang/fmt"
	"feelings/src/golang/io"
	"feelings/src/lib/trust"

	"unsafe"

	rt "feelings/src/tinygo_runtime"
)

//export raw_exception_handler
func rawExceptionHandler() {
	_ = rt.MiniUART.WriteString("TRAPPED INTR\n") //should not happen
}

var sdRca, sdErr uint64
var sdScr [2]uint64

func main() {
	rt.MiniUART = rt.NewUART()
	_ = rt.MiniUART.Configure(rt.UARTConfig{ /*no interrupt*/ })
	buffer := make([]byte, 512)
	dir := &fatDir{}
	//for now, hold the buffers on stack
	sectorCache := make([]byte, 0x200<<6) //0x40 pages
	sectorBitSet := make([]uint64, 1)

	if err := sdInit(); err == nil {
		sdcard := fatGetPartition(buffer) //data read into this buffer
		if sdcard == nil {
			trust.Errorf("Unable to read MBR or unable to parse BIOS parameter block")
		} else {
			tranq := NewTraquilBufferManager(unsafe.Pointer(&sectorCache[0]), 0x40,
				unsafe.Pointer(&sectorBitSet[0]), sdcard.readInto, nil)
			var err error
			var r int
			entries := 0
			buf := make([]byte, directoryEntrySize)
			lfnSeq := ""
			lfnSeqCurr := 0                                                           //lfn's numbered from 1
			fr := newFATDataReader(sdcard.activePartition.rootCluster, sdcard, tranq) //get root directory
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
				if ok := dir.unpack(buf); !ok {
					trust.Errorf("unable to unpack directory: %v ", err.Error())
					break outer
				}
				entries++
				switch {
				case dir.name[0] == directoryEnd:
					trust.Infof("cache hits: %d, cache misses %d, cache hit %2.0f%%, ousters %d\n",
						tranq.cacheHits, tranq.cacheMisses,
						(float64(tranq.cacheHits)/(float64(tranq.cacheHits)+float64(tranq.cacheMisses)))*100.0,
						tranq.cacheOusters)

					break outer
				case dir.name[0] == directoryEntryDeleted:
					continue
				case dir.Attrib == directoryEntryLFN:
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
					nameLen := strlenWithTerminator(dir.name[:], ' ')
					extLen := strlenWithTerminator(dir.ext[:], ' ')
					shortName := string(dir.name[:nameLen])
					if extLen > 0 {
						shortName += "." + string(dir.ext[:extLen])
					}
					if len(shortName) == 0 {
						trust.Warnf("found a short name for a file that is empty!")
					}
					//fmt.Printf("\t (%s, %s)\n", lfnSeq, shortName)
					longName := shortName
					if lfnSeqCurr > 0 {
						longName = lfnSeq
					}
					if dir.Attrib&attributeSubdirectory != 0 {
						longName += "/"
					}
					lfnSeq = ""
					lfnSeqCurr = 0 // lfn's seqence numbers start at 1
					_, _ = fmt.Printf("%-20s %10d\n", longName, dir.Size)
				}
			}
			if err == io.EOF {
				trust.Warnf("finished reading all the directory entries, but shouldn't we have gotten a directory end entry?")
			}
			if err != nil {
				panic("aborting due to error reading root dir")
			}
		}
	} else {
		trust.Errorf("failed to launch")
	}

}
