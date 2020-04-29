package main

import (
	"feelings/src/hardware/bcm2835"
	rt "feelings/src/tinygo_runtime"
	"unsafe"

	"github.com/tinygo-org/tinygo/src/device/arm"
)

type mbrInfo struct {
	unused     [mbrUnusedSize]uint8
	Partition1 partitionInfo //customary to number from 1
	Partition2 partitionInfo
	Partition3 partitionInfo
	Partition4 partitionInfo
	Signature  uint16 //0xaa55
}

type partitionInfo struct {
	Status         uint8  // 0x80 - active partition
	HeadStart      uint8  // starting head
	CylSelectStart uint16 // starting cylinder and sector
	Type           uint8  // partition type (01h = 12bit FAT, 04h = 16bit FAT, 05h = Ex MSDOS, 06h = 16bit FAT (>32Mb), 0Bh = 32bit FAT (<2048GB))
	HeadEnd        uint8  // ending head of the partition
	CylSectEnd     uint16 // ending cylinder and sector
	FirstSector    uint32 // total sectors between MBR & the first sector of the partition
	SectorsTotal   uint32 // size of this partition in sectors
}

// this is the either the whole disk or the 1st partition
type sdCardInfo struct {
	// xxx add details about the card itself
	activePartition struct {
		rootCluster         uint32 // Active partition rootCluster
		sectorsPerCluster   uint32 // Active partition sectors per cluster
		bytesPerSector      uint32 // Active partition bytes per sector
		fatOrigin           uint32 // The beginning of the 1 or more FATs (sector number)
		fatSize             uint32 // fat size in sectors, including all FATs
		dataSectors         uint32 // Active partition data sectors
		unusedSectors       uint32 // Active partition unused sectors (this is also the offset of the partition)
		reservedSectorCount uint32 // Active partition reserved sectors
		isFat16             bool
		fat                 []byte
	}
}

func sdWaitForInterrupt(mask uint32) int {
	var r int
	var m = uint32(mask | bcm2835.InterruptErrorMask)
	cnt := 1000000
	for (bcm2835.EMCC.Interrupt.Get()&m == 0) && cnt > 0 {
		rt.WaitMuSec(1)
		cnt--
	}
	r = int(bcm2835.EMCC.Interrupt.Get())
	if cnt <= 0 || (r&bcm2835.InterruptCommandTimeout) > 0 || (r&bcm2835.InterruptDataTimeout) > 0 {
		rt.MiniUART.Hex64string(uint64(r & bcm2835.InterruptCommandTimeout))
		rt.MiniUART.Hex64string(uint64(r & bcm2835.InterruptDataTimeout))
		bcm2835.EMCC.Interrupt.Set(uint32(r))
		return bcm2835.SDTimeout
	}
	if (r & bcm2835.InterruptErrorMask) > 0 {
		bcm2835.EMCC.Interrupt.Set(uint32(r))
		return bcm2835.SDError
	}
	bcm2835.EMCC.Interrupt.Set(mask)
	return bcm2835.SDOk
}

func sdStatus(mask uint32) int {

	cnt := 500000
	for (bcm2835.EMCC.Status.Get()&mask) != 0 && (bcm2835.EMCC.Interrupt.Get()&bcm2835.InterruptErrorMask) == 0 && cnt > 0 {
		rt.WaitMuSec(1)
		cnt--
	}
	if cnt <= 0 || (bcm2835.EMCC.Interrupt.Get()&bcm2835.InterruptErrorMask) > 0 {
		return bcm2835.SDError
	}
	return bcm2835.SDOk
}

func sdSendCommand(code uint32, arg uint32) int {
	r := int(0)
	sd_err = bcm2835.SDOk

	//do we need to force the command app command first?
	if code&bcm2835.CommandNeedApp > 0 {
		rca := 0
		if sd_rca > 0 {
			rca = bcm2835.CommandResponse48
		}
		r = sdSendCommand(bcm2835.CommandAppCommand|uint32(rca), uint32(sd_rca))

		if sd_rca > 0 && r == 0 {
			rt.MiniUART.WriteString("ERROR: failed to send SD APP command\n")
			sd_err = bcm2835.SDErrorUnsigned //uint64(int64(bcm2835.SDError))
			return 0
		}
		code = code & (^uint32(bcm2835.CommandNeedApp))
	}

	if sdStatus(bcm2835.SRCommandInhibit) > 0 {
		rt.MiniUART.WriteString("ERROR: EMMC Busy\n")
		sd_err = bcm2835.SDTimeoutUnsigned //uint64(int64(bcm2835.SDTimeout))
		return 0
	}
	if showCommands {
		rt.MiniUART.WriteString("sending command ")
		rt.MiniUART.Hex32string(code)
		rt.MiniUART.WriteString(" arg ")
		rt.MiniUART.Hex32string(arg)
		rt.MiniUART.WriteString("\n")
	}

	//if(sd_status(SR_CMD_INHIBIT)) { uart_puts("ERROR: EMMC busy\n"); sd_err= SD_TIMEOUT;return 0;}
	//uart_puts("EMMC: Sending command ");uart_hex(code);uart_puts(" arg ");uart_hex(arg);uart_puts("\n");

	bcm2835.EMCC.Interrupt.Set(bcm2835.EMCC.Interrupt.Get()) //???
	bcm2835.EMCC.Arg1.Set(arg)
	bcm2835.EMCC.CommandTransferMode.Set(code)

	if code == bcm2835.CommandSendOpCond {
		rt.WaitMuSec(1000) //up to one milli?
	} else if code == bcm2835.CommandSendIfCond || code == bcm2835.CommandAppCommand {
		rt.WaitMuSec(100)
	}

	r = sdWaitForInterrupt(bcm2835.InterruptCommandDone)
	if r != 0 {
		rt.MiniUART.WriteString("failed to send EMCC command\n")
		sd_err = uint64(r)
		return 0
	}

	r = int(bcm2835.EMCC.Response0.Get())
	if code == bcm2835.CommandGoIdle || code == bcm2835.CommandAppCommand {
		return 0
	} else if code == (bcm2835.CommandAppCommand | bcm2835.CommandResponse48) {
		return r & bcm2835.SRAppCommand
	} else if code == bcm2835.CommandSendOpCond {
		return r
	} else if code == bcm2835.CommandSendIfCond {
		if r == int(arg) {
			return bcm2835.SDOk
		}
		return bcm2835.SDError
	} else if code == bcm2835.CommandAllSendCID {
		r = r | int(bcm2835.EMCC.Response3.Get())
		r = r | int(bcm2835.EMCC.Response2.Get())
		r = r | int(bcm2835.EMCC.Response1.Get())
		return r
	} else if code == bcm2835.CommandSendRelAddr {
		right := int((r & 0x1fff) | ((r & 0x2000) << 6))
		left := int(((r & 0x4000) << 8) | ((r & 0x8000) << 8))
		sd_err = uint64((left | right) & bcm2835.CommandErrorsMask)
		return r & bcm2835.CommandRCAMask
	}
	return r & bcm2835.CommandErrorsMask
}

func sdInit() error {
	var r, ccs, cnt int
	r = int(bcm2835.GPIO.FuncSelect[4].Get())

	comp := ^(int(7) << int((7 * 3)))
	r = r & comp //clearing the pin seven entries?
	bcm2835.GPIO.FuncSelect[4].Set(uint32(r))
	waitOnPullUps(1 << 15)

	// GPIO_CLK
	r = int(bcm2835.GPIO.FuncSelect[4].Get())
	r = r | ((int(7) << (8 * 3)) | (int(7 << (9 * 3))))
	bcm2835.GPIO.FuncSelect[4].Set(uint32(r))
	waitOnPullUps((1 << 16) | (1 << 17))

	r = int(bcm2835.GPIO.FuncSelect[5].Get())
	r = r | ((int(7 << (0 * 3))) | (int(7 << (1 * 3))) | (int(7 << (2 * 3))) | (int(7 << (3 * 3))))
	bcm2835.GPIO.FuncSelect[5].Set(uint32(r))
	waitOnPullUps((1 << 18) | (1 << 19) | (1 << 20) | (1 << 21))

	sdHardwareVersion := (bcm2835.EMCC.SlotInterruptStatus.Get() & bcm2835.HostSpecNum) >> bcm2835.HostSpecNumShift

	rt.MiniUART.WriteString("EMMC: GPIO set up\n")

	// Reset the card.
	bcm2835.EMCC.Control0.Set(0)

	bcm2835.EMCC.Control1.SetBits(bcm2835.C1ResetHost)
	cnt = 10000
	for (bcm2835.EMCC.Control1.Get()&uint32(bcm2835.C1ResetHost) != 0) && cnt > 0 {
		rt.WaitMuSec(10)
		cnt--
	}

	if cnt <= 0 {
		rt.MiniUART.WriteString("Failed to reset EMCC\n")
		return bcm2835.NewSDInitFailure("Unable to reset EMCC")
	}

	//setup clocks
	bcm2835.EMCC.Control1.SetBits(bcm2835.C1ClockEnableInternal | bcm2835.C1_TOUNIT_MAX)
	rt.WaitMuSec(10)
	// Set clock to setup frequency.
	err := sdSetClockToFreq(400000, sdHardwareVersion)
	if err != nil {
		return err
	}
	bcm2835.EMCC.InterruptEnable.Set(0xffffffff)
	bcm2835.EMCC.InterruptMask.Set(0xffffffff)
	sd_scr[0] = 0
	sd_scr[1] = 0
	sd_rca = 0
	sd_err = 0

	sdSendCommand(bcm2835.CommandGoIdle, 0)
	if sd_err != 0 {
		return bcm2835.NewSDInitFailure("unable to get card to go idle")
	}

	sdSendCommand(bcm2835.CommandSendIfCond, 0x000001AA)
	if sd_err != 0 {
		return bcm2835.NewSDInitFailure("unable to send command if cond")
	}

	cnt = 6
	r = 0
	for (r&bcm2835.ACMD41_CMD_COMPLETE) == 0 && cnt > 0 {
		cnt--
		waitCycles(400)
		r = sdSendCommand(bcm2835.CommandSendOpCond, bcm2835.ACMD41_ARG_HC)
		rt.MiniUART.WriteString("EMMC: CMD_SEND_OP_COND returned ")
		if r&bcm2835.ACMD41_CMD_COMPLETE > 0 {
			rt.MiniUART.WriteString("COMPLETE ")
		}
		if r&bcm2835.ACMD41_VOLTAGE > 0 {
			rt.MiniUART.WriteString("VOLTAGE ")
		}
		if r&bcm2835.ACMD41_CMD_CCS > 0 {
			rt.MiniUART.WriteString("CSS ")
		}
		rt.MiniUART.WriteCR()
		if sd_err != bcm2835.SDTimeoutUnsigned && sd_err != bcm2835.SDOk {
			return bcm2835.NewSDInitFailure("EMMC ACMD41 returned error ")
		}
	}
	if r&bcm2835.ACMD41_CMD_COMPLETE == 0 || cnt == 0 {
		return bcm2835.NewSDInitFailure("EMMC ACMD41: Timeout ")
	}
	if r&bcm2835.ACMD41_VOLTAGE == 0 || cnt == 0 {
		return bcm2835.NewSDInitFailure("EMMC ACMD41: Error ")
	}
	if r&bcm2835.ACMD41_CMD_CCS != 0 {
		ccs = bcm2835.SCR_SUPP_CCS
	}

	sdSendCommand(bcm2835.CommandAllSendCID, 0)
	sd_rca = uint64(sdSendCommand(bcm2835.CommandSendRelAddr, 0))
	if sd_err != 0 {
		return bcm2835.NewSDInitFailure("Command Send Relative Addr Failed")
	}

	if sdSetClockToFreq(25000000, sdHardwareVersion) != nil {
		return bcm2835.NewSDInitFailure("Could not set clock speed to 25000000")
	}

	sdSendCommand(bcm2835.CommandCardSelect, uint32(sd_rca))
	if sd_err != 0 {
		return bcm2835.NewSDInitFailure("Could not set select SD card")
	}

	if sdStatus(bcm2835.SRDataInhibit) != 0 {
		return bcm2835.NewSDInitFailure("timeout initializing card")
	}

	bcm2835.EMCC.BlockSizAndCount.Set((1 << 16) | 8)

	sdSendCommand(bcm2835.CommandSendSCR, 0)
	if sd_err != 0 {
		return bcm2835.NewSDInitFailure("Unable to use SendSCR command")
	}

	if sdWaitForInterrupt(bcm2835.InterruptReadReady) != 0 {
		return bcm2835.NewSDInitFailure("Timed out waiting for read ready")
	}

	r = 0
	cnt = 100000
	for r < 2 && cnt > 0 {
		if bcm2835.EMCC.Status.Get()&bcm2835.SRReadAvailable != 0 {
			sd_scr[r] = uint64(bcm2835.EMCC.Data.Get())
			r++
		} else {
			rt.WaitMuSec(1)
		}
	}

	if r != 2 {
		return bcm2835.NewSDInitFailure("unable to retreive data for scr register")
	}
	if sd_scr[0]&bcm2835.SCR_SD_BUS_WIDTH_4 != 0 {
		sdSendCommand(bcm2835.CommandSetBusWidth, uint32(sd_rca)|2)
		if sd_err != 0 {
			return bcm2835.NewSDInitFailure("Unable to set bus width")
		}
		r = int(bcm2835.EMCC.Control0.Get())
		r = r | bcm2835.Control0HCTLDataWidth
		bcm2835.EMCC.Control0.Set(uint32(r))
	}

	// add software flag
	rt.MiniUART.WriteString("EMMC: supports ")
	if sd_scr[0]&bcm2835.SCR_SUPP_SET_BLKCNT != 0 {
		rt.MiniUART.WriteString("SET_BLKCNT ")
	}
	if ccs != 0 {
		rt.MiniUART.WriteString("CCS ")
	}
	rt.MiniUART.WriteString("\n")
	r = int(sd_scr[0])
	comp = ^(bcm2835.SCR_SUPP_CCS)
	r = r & comp
	sd_scr[0] = uint64(r)
	sd_scr[0] = sd_scr[0] | uint64(ccs)

	return nil

}

func sdSetClockToFreq(f uint32, hwVersion uint32) error {
	var d, c, x, s, h uint32
	c = 41666666 / f
	s = 32
	h = 0
	cnt := 100000
	for (bcm2835.EMCC.Status.Get()&(bcm2835.SRCommandInhibit|bcm2835.SRDataInhibit)) != 0 && cnt > 0 {
		rt.WaitMuSec(1)
		cnt--
	}
	if cnt <= 0 {
		rt.MiniUART.WriteString("timeout waiting for inihibt flag\n")
		return bcm2835.NewSDInitFailure("timeout waiting for inihibit flag\n")
	}

	c1 := bcm2835.EMCC.Control1.Get()
	comp := ^(uint32(bcm2835.C1ClockEnable))
	c1 = c1 & comp
	bcm2835.EMCC.Control1.Set(c1)

	rt.WaitMuSec(10)

	//freq control
	x = c - 1
	if x == 0 {
		s = 0
	} else {
		if (x & uint32(0xffff0000)) == 0 {
			x <<= 16
			s -= 16
		}
		if (x & uint32(0xff000000)) == 0 {
			x <<= 8
			s -= 8
		}
		if (x & uint32(0xf0000000)) == 0 {
			x <<= 4
			s -= 4
		}
		if (x & uint32(0xc0000000)) == 0 {
			x <<= 2
			s -= 2
		}
		if (x & uint32(0x80000000)) == 0 {
			x <<= 1
			s -= 1
		}
		if s > 0 {
			s--
		}
		if s > 7 {
			s = 7
		}
	}
	if hwVersion > bcm2835.HostSpecV2 {
		d = c
	} else {
		d = 1 << s
	}
	if d <= 2 {
		d = 2
		s = 0
	}
	_ = rt.MiniUART.WriteString("sd clock divisor ")
	rt.MiniUART.Hex32string(d)
	_ = rt.MiniUART.WriteString("shift ")
	rt.MiniUART.Hex32string(s)
	_ = rt.MiniUART.WriteByte('\n')

	if hwVersion > bcm2835.HostSpecV2 {
		h = (d & 0x300) >> 2
	}
	d = (((d & 0x0ff) << 8) | h)

	r := bcm2835.EMCC.Control1.Get()
	r = (r & 0xffff003f) | d
	bcm2835.EMCC.Control1.Set(r)
	rt.WaitMuSec(10)
	r = r | bcm2835.C1ClockEnable
	bcm2835.EMCC.Control1.Set(r)
	rt.WaitMuSec(10)

	cnt = 10000
	for bcm2835.EMCC.Control1.Get()&bcm2835.C1ClockStable == 0 && cnt > 0 {
		cnt--
	}
	if cnt < 0 {
		_ = rt.MiniUART.WriteString("timeout waiting for stable clock\n")
		return bcm2835.NewSDInitFailure("timeout waiting for stable clock\n")
	}
	return nil
}

func waitOnPullUps(valueToSet uint32) {
	bcm2835.GPIO.PullUpDownEnable.Set(2)
	waitCycles(150)
	bcm2835.GPIO.PullUpDownEnableClock1.Set(valueToSet)
	waitCycles(150)
	bcm2835.GPIO.PullUpDownEnable.Set(0)
	bcm2835.GPIO.PullUpDownEnableClock1.Set(0)

}

func waitCycles(n int) {
	for n > 0 {
		arm.Asm("nop")
		n--
	}
}

func sdReadblock(lba uint32, num uint32) (int, []byte) {
	var r, c, d int
	c = 0
	if num < 1 {
		num = 1
	}
	buffer := make([]byte, sectorSize*num)
	//infoMessage("--------> start reading n blocks, first block @: ", num, lba)
	if sdStatus(bcm2835.SRDataInhibit) != 0 {
		sd_err = bcm2835.SDTimeoutUnsigned
		return 0, nil
	}
	buf := (*uint32)(unsafe.Pointer(&buffer[0]))
	if sd_scr[0]&bcm2835.SCR_SUPP_CCS != 0 {
		if num > 1 && (sd_scr[0]&bcm2835.SCR_SUPP_SET_BLKCNT != 0) {
			sdSendCommand(bcm2835.CommandSetBlockcount, num)
			if sd_err != 0 {
				return 0, nil
			}
		}
		bcm2835.EMCC.BlockSizAndCount.Set(uint32((num << 16) | 512))
		if num == 1 {
			sdSendCommand(bcm2835.CommandReadSingle, lba)
		} else {
			sdSendCommand(bcm2835.CommandReadMulti, lba)
		}
		if sd_err != 0 {
			return 0, nil
		}
	} else {
		bcm2835.EMCC.BlockSizAndCount.Set((1 << 16) | 512)
	}

	for c < int(num) {
		if sd_scr[0]&bcm2835.SCR_SUPP_CCS == 0 {
			sdSendCommand(bcm2835.CommandReadSingle, (lba+uint32(c))*sectorSize)
			if sd_err != 0 {
				return 0, nil
			}
		}
		r = sdWaitForInterrupt(bcm2835.InterruptReadReady)
		if r != 0 {
			rt.MiniUART.WriteString("ERROR: Timeout waiting for ready to read\n")
			sd_err = uint64(r)
			return 0, nil
		}
		for d = 0; d < sectorSize/4; d++ {
			*buf = bcm2835.EMCC.Data.Get()
			buf = (*uint32)(unsafe.Pointer(uintptr(unsafe.Pointer(buf)) + 4))
		}
		c++ //yuck!
		//you might think we should update buf here but it
		//was already updated to be at the beginning of the next
		//sector because it is pointing to a contiguous memory
		//blob
	}
	if num > 1 && sd_scr[0]&bcm2835.SCR_SUPP_SET_BLKCNT == 0 && sd_scr[0]&bcm2835.SCR_SUPP_CCS != 0 {
		sdSendCommand(bcm2835.CommandStopTrans, 0)
	}
	//did it blow up?
	if sd_err != bcm2835.SDOk {
		return int(sd_err), nil
	}
	//did we read the right amt?
	if c != int(num) {
		return 0, nil
	}
	//infoMessage("--------> done reading n bytes: ", uint32(len(buffer)))
	return int(num) * sectorSize, buffer
}
