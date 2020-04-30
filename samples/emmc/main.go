package main

import (
	"feelings/src/golang/io"
	"feelings/src/lib/trust"
	"fmt"

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
			fs := NewFAT32Filesystem(tranq, sdcard)
			rd, err := fs.Open("/etc/init.d/procps")
			if err != nil {
				trust.Errorf("unable to open path: %s", err.Error())
			}
			buffer := make([]uint8, 256)
			for {
				n, err := rd.Read(buffer)
				if err == io.EOF {
					break
				}
				if err != nil {
					trust.Errorf("failed reading file: %s", err.Error())
				}
				if n == 0 {
					continue
				}
				s := string(buffer[n])
				fmt.Printf(s)
			}
		}
	}
}
