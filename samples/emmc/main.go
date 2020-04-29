package main

import (
	"feelings/src/golang/fmt"
	rt "feelings/src/tinygo_runtime"
	"io"
	"unsafe"
)

//export raw_exception_handler
func raw_exception_handler() {
	rt.MiniUART.WriteString("TRAPPED INTR\n") //should not happen
}

var sd_ocr, sd_rca, sd_err, sd_hv uint64
var sd_scr [2]uint64

const sectorSize = 0x200

//func mainNormal() {
//	rt.MiniUART = rt.NewUART()
//	_ = rt.MiniUART.Configure(rt.UARTConfig{ /*no interrupt*/ })
//
//	buffer := make([]byte, 512)
//	if err := sdInit(); err == nil {
//		bpb := fatGetPartition(buffer) //data read into this buffer
//		if bpb == nil {
//			errorMessage("Unable to read MBR or unable to parse BIOS parameter block")
//		} else {
//			fn := "FOO.CFG"
//			cluster := fatGetCluster(fn, bpb)
//			if cluster == 0 {
//				errorMessage("file not found")
//			} else {
//				infoMessage("cluster value sent to fatReadFile ", cluster)
//				data := fatReadfile(cluster, bpb, partitionlba)
//				if data == nil {
//					errorMessage("unable to read cluster data for" + fn)
//				}
//				infoMessage("file raw size is:", uint32(len(data)))
//				rt.MiniUART.Dump(unsafe.Pointer(&data[0]))
//			}
//		}
//	} else {
//		_ = rt.MiniUART.WriteString("ERROR unable to init card: ")
//		rt.MiniUART.WriteString(err.Error())
//	}
//	rt.MiniUART.WriteCR()
//	for {
//		arm.Asm("nop")
//	}
//}

func mainBug() {
	buffer := make([]byte, 512)
	for i := 0; i < 512; i++ {
		buffer[i] = byte(i) //0->255 then 0->255, corresponding to the index number as byte
	}
	base := uintptr(unsafe.Pointer(&buffer[0]))
	for dptr := uintptr(0); dptr < 512; dptr += 0x20 {
		dirEntry := buffer[int(dptr) : int(dptr)+0x20] //32 byte slice
		for i := 0; i < 20; i++ {
			d := int(dptr)
			bptr := (*byte)(unsafe.Pointer(base + dptr + uintptr(i)))
			if buffer[d+i] != byte(d+i) || dirEntry[i] != byte(d+i) || *bptr != byte(d+i) {
				print("bogus\n")
			}
		}
	}
}

func main() {
	rt.MiniUART = rt.NewUART()
	_ = rt.MiniUART.Configure(rt.UARTConfig{ /*no interrupt*/ })
	buffer := make([]byte, 512)

	//for now, hold the buffers on stack
	sectorCache := make([]byte, 0x200<<6) //0x40 pages
	sectorBitSet := make([]uint64, 1)

	if err := sdInit(); err == nil {
		sdcard := fatGetPartition(buffer) //data read into this buffer
		if sdcard == nil {
			errorMessage("Unable to read MBR or unable to parse BIOS parameter block")
		} else {
			tranq := NewTraquilBufferManager(unsafe.Pointer(&sectorCache[0]), 0x40,
				unsafe.Pointer(&sectorBitSet[0]), sdcard.readInto, nil)
			var dir *fatDir = newFATDir()
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
						errorMessage("unknown error caught:" + err.Error())
						break outer
					}
					curr += r
				}
				if ok := dir.unpack(buf); !ok {
					errorMessage("unable to unpack directory: " + err.Error())
					break outer
				}
				entries++
				switch {
				case dir.name[0] == directoryEnd:
					fmt.Printf("cache hits: %d, cache misses %d, cache hit %2.0f%%, ousters %d\n",
						tranq.cacheHits, tranq.cacheMisses,
						(float64(tranq.cacheHits)/(float64(tranq.cacheHits)+float64(tranq.cacheMisses)))*100.0,
						tranq.cacheOusters)

					break outer
				case dir.name[0] == directoryEntryDeleted:
					continue
				case dir.Attrib == directoryEntryLFN:
					lfn := longFilename(buf[0:directoryEntrySize])
					if lfn == nil {
						errorMessage("unable to understand long file name in directory")
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
						warnMessage("found a short name for a file that is empty!")
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
					fmt.Printf("%-20s %10d\n", longName, dir.Size)
				}
			}
			if err == io.EOF {
				infoMessage("finished reading all the directory entries, but shouldn't we have gotten a directory end entry?")
			}
			if err != nil {
				panic("aborting due to error reading root dir")
			}
		}
	} else {
		errorMessage("failed to launch")
	}

}
