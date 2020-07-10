package emmc

import (
	"fmt"
	"unsafe"

	"machine"

	"lib/trust"
)

// this is the either the whole disk or the 1st partition
type sdCardInfo struct {
	// xxx add details about the card itself
	activePartition *fatPartition
}

func readInto(sector sectorNumber, data unsafe.Pointer) EmmcError {
	//	trust.Infof("readInto buffer, sector %d", sector)
	result := sdReadblockInto(sector, 1, data)
	if result != EmmcOk {
		return result
	}
	return EmmcOk
}

//reads into a buffer created on the heap
func sdReadblock(lba sectorNumber, num uint32) (uint32, []byte, EmmcError) {
	buffer := make([]byte, sectorSize*num)
	buf := unsafe.Pointer(&buffer[0])
	err := sdReadblockInto(lba, num, buf)
	if err != EmmcOk {
		return 0, nil, err
	}
	return sectorSize * num, buffer, EmmcOk
}

//reads num sectors starting at lba into a buffer
//provided
func sdReadblockInto(lba sectorNumber, num uint32, buf unsafe.Pointer) EmmcError {
	machine.EMMC.BlockSizeAndCount.SetBlkCnt(num)

	if num < 1 {
		trust.Errorf("sdreadblock: requested bad number of blocks (%d), using 1 instead",
			num)
		num = 1
	}

	if emmcDriverDebug {
		trust.Debugf("-- testing data inhibit or error ---")
	}

	if machine.EMMC.DebugStatus.Get()&Datinhibit != 0 {
		//		trust.Debugf("waiting on data inhibit... @TICKS=%d", machine.SystemTime())
		ok := false
		for j := 0; j < 30; j++ {
			if machine.EMMC.DebugStatus.Get()&Datinhibit == 0 {
				ok = true
				break
			}
			if !machine.EMMC.Interrupt.ErrorIsSet() {
				ok = true
				break
			}
			delay(1)
		}
		if !ok {
			trust.Errorf("timed out waiting on data inhibit %v "+
				"or error (%v): @TICKS=%d",
				machine.EMMC.DebugStatus.Get()&Datinhibit != 0,
				machine.EMMC.Interrupt.ErrorIsSet(),
				machine.SystemTime())
			return EmmcDataInhibitTimeout
		}
	}

	var resp [4]uint32
	if readerDebug {
		trust.Infof("--- start reading %d blocks, first block @%d ---", num, lba)
	}
	if num == 1 {
		raw := uint32(lba << 9)
		if emmccmd(ReadSingle, raw, &resp) != 0 {
			trust.Errorf("aborting read block into for block %d", lba)
			return EmmcBadReadBlock
		}
	} else {
		raw := uint32(lba)
		if emmccmd(ReadMulti, raw, &resp) != 0 {
			trust.Errorf("aborting read multi block into for block %d", lba)
			return EmmcBadReadMultiBlock
		}
	}
	//trust.Debugf("emmccmd produced %+v response", resp)
	ct := 0
	//trust.Debugf("-- testing data read is ready ---")
	for !machine.EMMC.Interrupt.ReadReadyIsSet() && ct < 10 {
		delay(1)
		ct++
	}
	if !machine.EMMC.Interrupt.ReadReadyIsSet() {
		trust.Errorf("did not receive interrupt signal that data read is ready")
		return EmmcNoDataReady
	}
	c := uint32(0)

	for c < num {
		ptr := unsafe.Pointer(uintptr(buf) + uintptr(c*sectorSize))
		if err := syncio(false, ptr, sectorSize); err != EmmcOk {
			trust.Debugf("error reading in syncio: %d,%v",
				err, err != EmmcOk)
			return err
		}
		c++ //yech
	}
	return EmmcOk
}

//sync io is poor :-( note that this does NOT clear interrupts because
//we may doing multiplereads
func syncio(write bool, buf unsafe.Pointer, bufSize uint32) EmmcError {

	// must be 4 byte aligned
	if uintptr(buf)&03 != 0 {
		panic("data sent or received to sdio card must be 4 byte aligned")
	}

	if !write {
		for d := uint32(0); d < bufSize/4; d++ {
			if readerDebug {
				if d%8 == 0 {
					fmt.Printf("0x%03x:", d*4)
				}
			}
			buffer := (*uint32)(unsafe.Pointer(uintptr(buf) + uintptr(d*4)))
			*buffer = machine.EMMC.Data.Get()
			if readerDebug {
				fmt.Printf("%08x ", *buffer)
			}
			if readerDebug {
				if d%8 == 7 {
					fmt.Printf("\n")
				}
			}
		}
		if readerDebug {
			trust.Debugf("<--- read %d bytes", bufSize)
		}
	} else {
		panic("not implemented yet")
	}
	return EmmcOk
}
