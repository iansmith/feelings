package main

// adapted (a lot) from
// https://github.com/bztsrc/raspi3-tutorial/blob/master/0B_readsector/sd.c

import (
	"device/arm"
	"errors"
	"lib/trust"
	"machine"
	"unsafe"
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

type fatPartition struct {
	rootCluster         uint32 // Active partition rootCluster
	sectorsPerCluster   uint32 // Active partition sectors per cluster
	bytesPerSector      uint32 // Active partition bytes per sector
	fatOrigin           uint32 // The beginning of the 1 or more FATs (sector number)
	fatSize             uint32 // fat size in sectors, including all FATs
	dataSectors         uint32 // Active partition data sectors
	unusedSectors       uint32 // Active partition unused sectors (this is also the offset of the partition)
	reservedSectorCount uint32 // Active partition reserved sectors
	isFAT16             bool
}

//bits 15,17,18,19,20,21,22,24
//AutoCommand,DataEndBitNot1,DataCRC,DataTimeOut,CommandBad,CommandEndBitNot1
//CommandCRC,Error
const InterruptErrorMask = 0x017E8000

//command flags
const CommandNeedApp = 0x80000000
const CommandResponse48 = 0x00020000
const CommandErrorsMask = 0xfff9c004
const CommandRCAMask = 0xffff0000

// COMMANDs
const CommandGoIdle = 0x00000000
const CommandAllSendCID = 0x02010000
const CommandSendRelAddr = 0x03020000
const CommandCardSelect = 0x07030000
const CommandSendIfCond = 0x08020000
const CommandStopTrans = 0x0C030000
const CommandReadSingle = 0x11220010
const CommandReadMulti = 0x12220032
const CommandSetBlockCount = 0x17020000
const CommandAppCommand = 0x37000000
const CommandSetBusWidth = (0x06020000 | CommandNeedApp)
const CommandSendOpCond = (0x29020000 | CommandNeedApp)
const CommandSendSCR = (0x33220010 | CommandNeedApp)

// STATUS register settings...
// xxx this whole use of the DebugStatus register is dodgy
const StatusReadAvailable = 0x00000800
const StatusDataInhibit = 0x00000002
const StatusCommandInhibit = 0x00000001
const StatusAppCommand = 0x00000020

// INTERRUPT register settings
const InterruptDataTimeout = 0x00100000
const InterruptCommandTimeout = 0x00010000
const InterruptReadReady = 0x00000020

// XXX we should have a way to generate a MASK from a bitfield descriptor
// XXX this is being used as a mask: emmc.Interrupt.CommandDoneIsSet()
const InterruptCommandDone = 0x1 //bit 0

// SLOTISR_VERSION values
const HostSpecNum = 0x00ff0000
const HostSpecNumShift = 16
const HostSpecV3 = 2
const HostSpecV2 = 1
const HostSpecV1 = 0

// command 41
const ACommand41Voltage = 0x00ff8000
const ACommand41CommandComplete = 0x80000000
const ACommand41CommandCCS = 0x40000000
const ACommand41ArgHC = 0x51ff8000

// SCR Flags
const SCRSDBusWidth = 0x00000400
const SCRSupportSetBlockCount = 0x02000000

// added by BZT driver
const ScrSupportsCCS = 0x00000001

// this is the either the whole disk or the 1st partition
type sdCardInfo struct {
	// xxx add details about the card itself
	activePartition *fatPartition
}

func WaitMuSecDumb(n int) {
	for n > 0 {
		// 10,000 cycles at 1MHz should be enough?
		for i := 0; i < 10000; i++ {
			arm.Asm("nop")
		}
		n--
	}
}

func sdWaitForInterrupt(mask uint32) int {
	var r uint32
	var m = mask | InterruptErrorMask
	cnt := 1000000

	trust.Debugf("xxx sdWaitForIntr m=%x, value=%x", m, machine.EMMC.Interrupt.Get())
	WaitMuSecDumb(1)
	for (machine.EMMC.Interrupt.Get()&m == 0) && cnt > 0 {
		WaitMuSecDumb(1)
		if cnt%10000 == 0 {
			trust.Debugf("ct is %d", cnt)
		}
		cnt--
	}
	r = machine.EMMC.Interrupt.Get()
	if cnt <= 0 || machine.EMMC.Interrupt.CommandTimeoutErrorIsSet() ||
		machine.EMMC.Interrupt.DataTimeoutErrorIsSet() {
		trust.Debugf("InterruptCommandTimeout: %v, InterruptDataTimeout %v",
			machine.EMMC.Interrupt.CommandTimeoutErrorIsSet(),
			machine.EMMC.Interrupt.DataTimeoutErrorIsSet())
		machine.EMMC.Interrupt.Set(r)
		return EMMCTimeout
	}
	if (r & InterruptErrorMask) > 0 {
		machine.EMMC.Interrupt.Set(uint32(r))
		return EMMCError
	}
	machine.EMMC.Interrupt.Set(mask)
	return EMMCOk
}

func sdStatus(mask uint32) int {

	cnt := 500000
	for (machine.EMMC.DebugStatus.Get()&mask) != 0 &&
		(machine.EMMC.Interrupt.Get()&InterruptErrorMask) == 0 &&
		cnt > 0 {
		WaitMuSecDumb(1)
		cnt--
	}
	if cnt <= 0 || (machine.EMMC.Interrupt.Get()&InterruptErrorMask) > 0 {
		return EMMCError
	}
	return EMMCOk
}

func sdSendCommand(code uint32, arg uint32) int {
	r := 0
	sdErr = EMMCOk

	trust.Debugf("sdSendCommand1: %d", code)
	//do we need to force the command app command first?
	if code&CommandNeedApp > 0 {
		rca := 0
		if sdRca > 0 {
			rca = CommandResponse48
		}
		r = sdSendCommand(CommandAppCommand|uint32(rca), uint32(sdRca))

		if sdRca > 0 && r == 0 {
			trust.Errorf("failed to send SD APP command\n")
			sdErr = uint64(EMMCErrorUnsigned)
			return 0
		}
		code = code & (^uint32(CommandNeedApp))
	}
	trust.Debugf("sdSendCommand2: %d", code)

	if sdStatus(StatusCommandInhibit) > 0 {
		trust.Errorf("ERROR: EMMC Busy\n")
		sdErr = EMMCTimeoutUnsigned //uint64(int64(bcm2835.SDTimeout))
		return 0
	}
	if showCommands {
		trust.Debugf("sending command %x arg %x\n",
			code, arg)
	}

	machine.EMMC.Interrupt.Set(machine.EMMC.Interrupt.Get()) //???
	machine.EMMC.Argument.Set(arg)
	machine.EMMC.CommandTransferMode.Set(code)

	trust.Debugf("sdSendCommand3: %d", code)
	if code == CommandSendOpCond {
		WaitMuSecDumb(1000) //up to one milli?
	} else if code == CommandSendIfCond || code == CommandAppCommand {
		WaitMuSecDumb(100)
	}

	trust.Debugf("sdSendCommand4: %d", code)
	r = sdWaitForInterrupt(InterruptCommandDone)
	if r != 0 {
		trust.Errorf("failed to send EMMC command\n")
		sdErr = uint64(r)
		return 0
	}

	r = int(machine.EMMC.Response0.Get())
	trust.Debugf("sdSendCommand6: %d,%d", code, r)
	if code == CommandGoIdle || code == CommandAppCommand {
		return 0
	} else if code == (CommandAppCommand | CommandResponse48) {
		return r & StatusAppCommand
	} else if code == CommandSendOpCond {
		return r
	} else if code == CommandSendIfCond {
		if r == int(arg) {
			return EMMCOk
		}
		return EMMCError
	} else if code == CommandAllSendCID {
		r = r | int(machine.EMMC.Response3.Get())
		r = r | int(machine.EMMC.Response2.Get())
		r = r | int(machine.EMMC.Response1.Get())
		return r
	} else if code == CommandSendRelAddr {
		right := (r & 0x1fff) | ((r & 0x2000) << 6)
		left := ((r & 0x4000) << 8) | ((r & 0x8000) << 8)
		sdErr = uint64((left | right) & CommandErrorsMask)
		return r & CommandRCAMask
	}
	return r & CommandErrorsMask
}

// It's not at all clear what these are needed for
func GPIOPinsInit() int {
	var r int
	r = int(machine.GPIO.FSel[4].Get())

	comp := ^(7 << (7 * 3))
	r = r & comp //clearing the pin seven entries?
	machine.GPIO.FSel[4].Set(uint32(r))
	waitOnPullUps(1 << 15)

	// GPIO_CLK
	r = int(machine.GPIO.FSel[4].Get())
	r = r | ((7 << (8 * 3)) | (7 << (9 * 3)))
	machine.GPIO.FSel[4].Set(uint32(r))
	waitOnPullUps((1 << 16) | (1 << 17))

	r = int(machine.GPIO.FSel[5].Get())
	r = r | ((7 << (0 * 3)) | (7 << (1 * 3)) | (7 << (2 * 3)) | (7 << (3 * 3)))
	machine.GPIO.FSel[5].Set(uint32(r))
	waitOnPullUps((1 << 18) | (1 << 19) | (1 << 20) | (1 << 21))
	return EMMCOk
}

func sdInit() int {
	var r, ccs, cnt int

	if err := GPIOPinsInit(); err != EMMCOk {
		return err
	}

	sdHardwareVersion := (machine.EMMC.SlotISRVersion.SDVersion())
	//SlotInterruptStatus.Get() & bcm2835.HostSpecNum) >> bcm2835.HostSpecNumShift
	trust.Infof("EMMC: GPIO (?) set up (version %d)\n", sdHardwareVersion)

	// Reset the card.
	machine.EMMC.Control0.Set(0)

	machine.EMMC.Control1.SetResetHostCircuit()
	cnt = 10000
	for machine.EMMC.Control1.ResetHostCircuitIsSet() && cnt > 0 {
		WaitMuSecDumb(10)
		cnt--
	}

	if cnt <= 0 {
		trust.Errorf("Failed to reset EMMC\n")
		return EMMCError
	}

	//setup clocks
	machine.EMMC.Control1.SetEnableClock()
	machine.EMMC.Control1.SetDataTimeoutUnitExponent(0b1110) //max
	WaitMuSecDumb(10)
	// Set clock to setup frequency.
	err := sdSetClockToFreq(400000, sdHardwareVersion)
	if err != EMMCOk {
		return err
	}
	trust.Debugf("enabling interrupts, but masking them")
	machine.EMMC.EnableInterrupt.Set(0xffffffff) //turn them all on, then...
	machine.EMMC.InterruptMask.Set(0xffffffff)   //huh? why?
	sdScr[0] = 0
	sdScr[1] = 0
	sdRca = 0
	sdErr = 0

	trust.Debugf("sending go idle")
	sdSendCommand(CommandGoIdle, 0)
	if sdErr != 0 {
		trust.Errorf("unable to get card to go idle")
		return EMMCError
	}

	trust.Debugf("sending if cond")

	sdSendCommand(CommandSendIfCond, 0x000001AA)
	if sdErr != 0 {
		trust.Errorf("unable to send IF cond")
		return EMMCError
	}

	cnt = 6
	r = 0
	for (r&ACommand41CommandComplete) == 0 && cnt > 0 {
		cnt--
		waitCycles(400)
		trust.Debugf("sending op cond, magic number 41")
		r = sdSendCommand(CommandSendOpCond, ACommand41ArgHC)
		trust.Infof("EMMC: CMD_SEND_OP_COND returned ")
		if r&ACommand41CommandComplete > 0 {
			trust.Infof("---> COMPLETE ")
		}
		if r&ACommand41Voltage > 0 {
			trust.Infof("---> VOLTAGE ")
		}
		if r&ACommand41CommandCCS > 0 {
			trust.Infof("---> CCS ")
		}
		if sdErr != EMMCTimeoutUnsigned && sdErr != EMMCOk {
			trust.Errorf("EMMC ACMD41 returned error")
			return EMMCError
		}
	}
	if r&ACommand41CommandComplete == 0 || cnt == 0 {
		trust.Errorf("EMMC ACMD41: Timeout ")
		return EMMCTimeout
	}
	if r&ACommand41Voltage == 0 || cnt == 0 {
		trust.Errorf("EMMC ACMD41: Error ")
		return EMMCError
	}
	if r&ACommand41CommandCCS != 0 {
		ccs = ScrSupportsCCS
	}

	sdSendCommand(CommandAllSendCID, 0)
	sdRca = uint64(sdSendCommand(CommandSendRelAddr, 0))
	if sdErr != 0 {
		trust.Errorf("Command Send Relative Addr Failed")
		return EMMCError
	}

	if sdSetClockToFreq(25000000, sdHardwareVersion) != EMMCOk {
		trust.Errorf("Could not set clock speed to 25000000")
		return EMMCError
	}

	sdSendCommand(CommandCardSelect, uint32(sdRca))
	if sdErr != 0 {
		trust.Errorf("Could not set select SD card")
		return EMMCError
	}
	trust.Debugf("CommandCardSelect")

	if sdStatus(StatusDataInhibit) != 0 {
		trust.Errorf("timeout initializing card")
		return EMMCError
	}

	machine.EMMC.BlockSizeAndCount.SetBlkCnt(1)
	machine.EMMC.BlockSizeAndCount.SetBlkSize(8)

	sdSendCommand(CommandSendSCR, 0)
	if sdErr != 0 {
		trust.Errorf("Unable to use SendSCR command")
	}
	trust.Debugf("SendSCR")

	if sdWaitForInterrupt(InterruptReadReady) != 0 {
		trust.Errorf("timeout waiting for read ready")
		return EMMCError
	}

	r = 0
	cnt = 100000
	for r < 2 && cnt > 0 {
		if machine.EMMC.DebugStatus.Get()&StatusReadAvailable != 0 {
			sdScr[r] = uint64(machine.EMMC.Data.Get())
			r++
		} else {
			WaitMuSecDumb(1)
		}
	}

	if r != 2 {
		trust.Errorf("unable to retreive data for scr register")
		return EMMCError
	}
	if sdScr[0]&SCRSDBusWidth != 0 {
		sdSendCommand(CommandSetBusWidth, uint32(sdRca)|2)
		if sdErr != 0 {
			trust.Errorf("Unable to set bus width")
			return EMMCError
		}
		machine.EMMC.Control0.SetHardwareControlDataWidth()
	}

	// add software flag
	trust.Infof("EMMC: supports ")
	if sdScr[0]&SCRSupportSetBlockCount != 0 {
		trust.Infof("---> SET_BLKCNT ")
	}
	if ccs != 0 {
		trust.Infof("---> CCS ")
	}
	r = int(sdScr[0])
	comp := ^(ScrSupportsCCS)
	r = r & comp
	sdScr[0] = uint64(r)
	sdScr[0] = sdScr[0] | uint64(ccs)

	return EMMCOk
}

func sdSetClockToFreq(f uint32, hwVersion uint32) int {
	var d, c, x, s, h uint32
	c = 41666666 / f
	s = 32
	h = 0
	cnt := 100000
	//xxx these should be bit fields
	for (machine.EMMC.DebugStatus.Get()&(StatusCommandInhibit|StatusDataInhibit)) != 0 && cnt > 0 {
		WaitMuSecDumb(1)
		cnt--
	}
	if cnt <= 0 {
		trust.Infof("timeout waiting for inihibt flag\n")
		return EMMCTimeout
	}

	machine.EMMC.Control1.ClearEnableClock()
	WaitMuSecDumb(10)

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
	if hwVersion > HostSpecV2 {
		d = c
	} else {
		d = 1 << s
	}
	if d <= 2 {
		d = 2
		s = 0
	}
	trust.Infof("sd clock divisor 0x%x shift 0x%x", d, s)

	if hwVersion > HostSpecV2 {
		h = (d & 0x300) >> 2
	}
	d = ((d & 0x0ff) << 8) | h

	//xxx figure out what bitfields are being set/cleared
	r := machine.EMMC.Control1.Get()
	r = (r & 0xffff003f) | d
	machine.EMMC.Control1.Set(r)
	WaitMuSecDumb(10)
	machine.EMMC.Control1.SetEnableClock()
	WaitMuSecDumb(10)

	cnt = 10000
	for !machine.EMMC.Control1.ClockStableIsSet() && cnt > 0 {
		cnt--
	}
	if cnt < 0 {
		trust.Errorf("timeout waiting for stable clock\n")
		return EMMCTimeout
	}
	trust.Infof("clock is stable")

	return EMMCOk
}

func waitOnPullUps(valueToSet uint32) {
	machine.GPIO.GPPUD.Set(2)
	waitCycles(150)
	machine.GPIO.GPUDClk[1].Set(valueToSet)
	waitCycles(150)
	machine.GPIO.GPPUD.Set(0)
	machine.GPIO.GPUDClk[1].Set(0)

}

func waitCycles(n int) {
	for n > 0 {
		arm.Asm("nop")
		n--
	}
}

func (s *sdCardInfo) readInto(sector uint32, data unsafe.Pointer) error {
	result := sdReadblockInto(sector, 1, (*uint32)(data))
	if result == 0 {
		errors.New("should be a read error type")
	}
	return nil
}

//reads into a buffer created on the heap
func sdReadblock(lba uint32, num uint32) (int, []byte) {
	buffer := make([]byte, sectorSize*num)
	buf := (*uint32)(unsafe.Pointer(&buffer[0]))
	read := sdReadblockInto(lba, num, buf)
	return read, buffer
}

//reads num sectors starting at lba into a buffer
//provided
func sdReadblockInto(lba uint32, num uint32, buf *uint32) int {
	var r, c, d int
	c = 0
	if num < 1 {
		num = 1
	}
	//infoMessage("--------> start reading n blocks, first block @: ", num, lba)
	if sdStatus(StatusDataInhibit) != 0 {
		sdErr = EMMCTimeoutUnsigned
		return 0
	}
	if sdScr[0]&ScrSupportsCCS != 0 {
		if num > 1 && (sdScr[0]&SCRSupportSetBlockCount != 0) {
			sdSendCommand(CommandSetBlockCount, num)
			if sdErr != 0 {
				return 0
			}
		}
		machine.EMMC.BlockSizeAndCount.SetBlkCnt(num)
		machine.EMMC.BlockSizeAndCount.SetBlkSize(sectorSize)
		if num == 1 {
			sdSendCommand(CommandReadSingle, lba)
		} else {
			sdSendCommand(CommandReadMulti, lba)
		}
		if sdErr != 0 {
			return 0
		}
	} else {
		machine.EMMC.BlockSizeAndCount.SetBlkCnt(1)
		machine.EMMC.BlockSizeAndCount.SetBlkSize(sectorSize)
	}

	for c < int(num) {
		if sdScr[0]&ScrSupportsCCS == 0 {
			sdSendCommand(CommandReadSingle, (lba+uint32(c))*sectorSize)
			if sdErr != 0 {
				return 0
			}
		}
		r = sdWaitForInterrupt(InterruptReadReady)
		if r != 0 {
			trust.Errorf("Timeout waiting for ready to read\n")
			sdErr = uint64(r)
			return 0
		}
		for d = 0; d < sectorSize/4; d++ {
			*buf = machine.EMMC.Data.Get()
			buf = (*uint32)(unsafe.Pointer(uintptr(unsafe.Pointer(buf)) + 4))
		}
		c++ //yuck!
		//you might think we should update buf here but it
		//was already updated to be at the beginning of the next
		//sector because it is pointing to a contiguous memory
		//blob
	}
	if num > 1 && sdScr[0]&SCRSupportSetBlockCount == 0 && sdScr[0]&ScrSupportsCCS != 0 {
		sdSendCommand(CommandStopTrans, 0)
	}
	//did it blow up?
	if sdErr != EMMCOk {
		return int(sdErr)
	}
	//did we read the right amt?
	if c != int(num) {
		return 0
	}
	//infoMessage("--------> done reading n bytes: ", uint32(len(buffer)))
	return int(num) * sectorSize
}
