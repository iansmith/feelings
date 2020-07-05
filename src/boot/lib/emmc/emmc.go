package main

import (
	"fmt"
	"unsafe"

	"device/arm"
	"lib/trust"
	"machine"
	"runtime/volatile"
)

// adapted to go on july 4 weekend of quarantine, 2020

/*
 * bcm2835 external mass media controller (mmc / sd host interface)
 *
 * Copyright © 2012 Richard Miller <r.miller@acm.org>
 */

/*
	Not officially documented: emmc can be connected to different gpio pins
		48-53 (SD card)
		22-27 (P1 header)
		34-39 (wifi - pi3 only)
	using ALT3 function to activate the required routing
*/
const VIRTIO = 0x3f00_0000
const EMMCREGS = (VIRTIO + 0x300000)

const Mhz = 1000000

// i don't know what the notation HZ means in plan9, looks like it might
// mean "clockspeed" in ticks
const HZ = 1000000

const Extfreq = 100 * Mhz /* guess external clock frequency if */
/* not available from vcore */
const Initfreq = 400000   /* initialisation frequency for MMC */
const SDfreq = 25 * Mhz   /* standard SD frequency */
const SDfreqhs = 50 * Mhz /* high speed frequency */
const DTO = 14            /* data timeout exponent (guesswork) */

const GoIdle = 0         /* mmc/sdio go idle state */
const MMCSelect = 7      /* mmc/sd card select command */
const Setbuswidth = 6    /* mmc/sd set bus width command */
const Switchfunc = 6     /* mmc/sd switch function command */
const Voltageswitch = 11 /* md/sdio switch to 1.8V */
const IORWdirect = 52    /* sdio read/write direct command */
const IORWextended = 53  /* sdio read/write extended command */
const Appcmd = 55        /* mmc/sd application command prefix */

const ReadSingle = 14
const ReadMulti = 15

const Arg2 = 0x00 >> 2
const Blksizecnt = 0x04 >> 2
const Arg1 = 0x08 >> 2
const Cmdtm = 0x0c >> 2
const Resp0 = 0x10 >> 2
const Resp1 = 0x14 >> 2
const Resp2 = 0x18 >> 2
const Resp3 = 0x1c >> 2
const Data = 0x20 >> 2
const Status = 0x24 >> 2
const Control0 = 0x28 >> 2
const Control1 = 0x2c >> 2
const Interrupt = 0x30 >> 2
const Irptmask = 0x34 >> 2
const Irpten = 0x38 >> 2
const Control2 = 0x3c >> 2
const Forceirpt = 0x50 >> 2
const Boottimeout = 0x70 >> 2
const Dbgsel = 0x74 >> 2
const Exrdfifocfg = 0x80 >> 2
const Exrdfifoen = 0x84 >> 2
const Tunestep = 0x88 >> 2
const Tunestepsstd = 0x8c >> 2
const Tunestepsddr = 0x90 >> 2
const Spiintspt = 0xf0 >> 2
const Slotisrver = 0xfc >> 2

/* Control0 */
const Hispeed = 1 << 2
const Dwidth4 = 1 << 1
const Dwidth1 = 0 << 1

/* Control1 */
const Srstdata = 1 << 26 /* reset data circuit */
const Srstcmd = 1 << 25  /* reset command circuit */
const Srsthc = 1 << 24   /* reset complete host controller */
const Datatoshift = 16   /* data timeout unit exponent */
const Datatomask = 0xF0000
const Clkfreq8shift = 8 /* SD clock base divider LSBs */
const Clkfreq8mask = 0xFF00
const Clkfreqms2shift = 6 /* SD clock base divider MSBs */
const Clkfreqms2mask = 0xC0
const Clkgendiv = 0 << 5  /* SD clock divided */
const Clkgenprog = 1 << 5 /* SD clock programmable */
const Clken = 1 << 2      /* SD clock enable */
const Clkstable = 1 << 1
const Clkintlen = 1 << 0 /* enable internal EMMC clocks */

// Cmdtm
const Indexshift = 24
const Suspend = 1 << 22
const Resume = 2 << 22
const Abort = 3 << 22
const Isdata = 1 << 21
const Ixchken = 1 << 20
const Crcchken = 1 << 19
const Respmask = 3 << 16
const Respnone = 0 << 16
const Resp136 = 1 << 16
const Resp48 = 2 << 16
const Resp48busy = 3 << 16
const Multiblock = 1 << 5
const Host2card = 0 << 4
const Card2host = 1 << 4
const Autocmd12 = 1 << 2
const Autocmd23 = 2 << 2
const Blkcnten = 1 << 1

// Interrupt
const Acmderr = 1 << 24
const Denderr = 1 << 22
const Dcrcerr = 1 << 21
const Dtoerr = 1 << 20
const Cbaderr = 1 << 19
const Cenderr = 1 << 18
const Ccrcerr = 1 << 17
const Ctoerr = 1 << 16
const Err = 1 << 15
const Cardintr = 1 << 8
const Cardinsert = 1 << 6 /* not in Broadcom datasheet */
const Readrdy = 1 << 5
const Writerdy = 1 << 4
const Datadone = 1 << 1
const Cmddone = 1 << 0

// Status, which natch the documentation says to not use
const Bufread = 1 << 11  /* not in Broadcom datasheet */
const Bufwrite = 1 << 10 /* not in Broadcom datasheet */
const Readtrans = 1 << 9
const Writetrans = 1 << 8
const Datactive = 1 << 2
const Datinhibit = 1 << 1
const Cmdinhibit = 1 << 0

//send command return codes
const Eio = 1
const EBadArg = 2

const numCommands = 64 // for assertion check, len(cmdinfo) would be much less
var cmdinfo = map[uint32]int{
	0:  Ixchken,
	2:  Resp136,
	3:  Resp48 | Ixchken | Crcchken,
	5:  Resp48,
	6:  Resp48 | Ixchken | Crcchken,
	7:  Resp48busy | Ixchken | Crcchken,
	8:  Resp48 | Ixchken | Crcchken,
	9:  Resp136,
	11: Resp48 | Ixchken | Crcchken,
	12: Resp48busy | Ixchken | Crcchken,
	13: Resp48 | Ixchken | Crcchken,
	16: Resp48,
	17: Resp48 | Isdata | Card2host | Ixchken | Crcchken,
	18: Resp48 | Isdata | Card2host | Multiblock | Blkcnten | Ixchken | Crcchken,
	24: Resp48 | Isdata | Host2card | Ixchken | Crcchken,
	25: Resp48 | Isdata | Card2host | Multiblock | Blkcnten | Ixchken | Crcchken,
	41: Resp48,
	52: Resp48 | Ixchken | Crcchken,
	53: Resp48 | Ixchken | Crcchken | Isdata,
	55: Resp48 | Ixchken | Crcchken,
}

type Ctlr struct {
	//r Rendez
	//cardr Rendez
	fastclock bool
	extclk    uint64
	appcmd    bool
}

var emmc Ctlr

//func mmcinterrupt(Ureg*, void*);

func clkdiv(d uint32) uint32 {
	var v uint32
	if d >= (1 << 10) {
		panic("wrong sized value of clock divisor")
	}
	v = (d << Clkfreq8shift) & Clkfreq8mask
	v |= ((d >> 8) << Clkfreqms2shift) & Clkfreqms2mask
	return v
}

//very dumb implementation... but useful for qemu where clocks are crap
func delay(n int) uint32 {
	r := volatile.Register32{}
	for i := 0; i < n*40000; i++ {
		r.Set(r.Get() + 1)
	}
	return r.Get()
}

func emmcclk(freq uint32) {
	var div uint32
	var i int

	div = uint32(emmc.extclk) / (freq << 1)
	if uint32(emmc.extclk)/(div<<1) > freq {
		div++
	}
	machine.EMMC.Control1.Set(clkdiv(div) | DTO<<Datatoshift | Clkgendiv | Clken | Clkintlen)
	//WR(Control1, clkdiv(div) | DTO<<Datatoshift | Clkgendiv | Clken | Clkintlen)
	for i = 0; i < 1000; i++ {
		delay(1)
		if machine.EMMC.Control1.ClockStableIsSet() {
			break
		}
	}
	if i == 1000 {
		panic(fmt.Sprintf("emmc: can't set clock to %ud\n", freq))
	}
}

func cardintready() bool {
	return machine.EMMC.Interrupt.CardIsSet()
}

func emmcinit() int {
	var clk uint64

	//clk = getclkrate(ClkEmmc); XXX fix me
	if clk == 0 {
		clk = Extfreq
		trust.Infof("emmc: assuming external clock %d Mhz\n", clk/1000000)
	}
	emmc.extclk = clk
	trust.Debugf("emmc control %08x %08x %08x\n",
		machine.EMMC.Control0.Get(),
		machine.EMMC.Control1.Get(),
		machine.EMMC.Control2.Get())
	machine.EMMC.Control1.SetResetHostCircuit()
	delay(10)
	for machine.EMMC.Control1.ResetHostCircuitIsSet() {
		delay(1)
	}
	machine.EMMC.Control1.SetResetData()
	delay(10)
	machine.EMMC.Control1.Set(0)
	return 0
}

var everythingButCardIntr = ^(uint32(Cardintr))
var everythingButDwidthAndHighspeed = ^(uint32(Dwidth4 | Hispeed))

func emmcgoidle() {
	machine.EMMC.Control0.Set(everythingButDwidthAndHighspeed)
	emmcclk(Initfreq)
	if machine.EMMC.DebugStatus.Get()&Cmdinhibit != 0 {
		trust.Errorf("unable to go idle, command inhibit")
	}
	if machine.EMMC.DebugStatus.Get()&Datinhibit != 0 {
		trust.Errorf("data inhibit high, but not used by go idle")
	}
	machine.EMMC.Argument.Set(0)
	r := machine.EMMC.Interrupt.Get() & everythingButCardIntr
	if r != 0 {
		if r != Cardinsert {
			trust.Errorf("before command, intr was %x", r)
		}
		machine.EMMC.Interrupt.Set(r)
	}
	machine.EMMC.CommandTransferMode.Set(0)
	for {
		if machine.EMMC.Interrupt.CommandDoneIsSet() {
			break
		}
		if machine.EMMC.Interrupt.Get() != 0 {
			trust.Infof("got something from intr: %x", machine.EMMC.Interrupt.Get())
			break
		}
		delay(1)
	}
}

var zero = uint32(0)

func emmcenable() {
	emmcclk(Initfreq)
	machine.EMMC.EnableInterrupt.Set(0)
	machine.EMMC.InterruptMask.Set(^zero)
	machine.EMMC.Interrupt.Set(^zero)
	//intrenable(IRQmmc, mmcinterrupt, nil, 0, "mmc");
}

//enable the card interrupt and return the rest of the interrupt value reg?
func sdiocardintr(canWait bool) int { // request from card
	var i int

	machine.EMMC.Interrupt.Set(Cardintr) // not documented
	for {
		i := machine.EMMC.Interrupt.Get()
		if i&Cardintr != 0 {
			break
		}
		if !canWait {
			return 0
		}
		machine.EMMC.EnableInterrupt.SetCard()
		// xxx sleep?
		delay(10)
	}
	machine.EMMC.Interrupt.SetCard()
	return i
}

var commandDoneOrError = uint32(Cmddone | Err)
var dataDoneOrError = uint32(Datadone | Err)
var everythingButCommandDoneOrError = ^commandDoneOrError
var dataReadOrWriteReady = uint32(Datadone | Readrdy | Writerdy)
var everythingButDataReadOrWriteReady = ^dataReadOrWriteReady
var everythingButDWidth4 = ^(uint32(Dwidth4))
var everythingBufLowestByte = ^(uint32(0xFF))

func emmccmd(cmd uint32, arg uint32, resp *[4]uint32) int {
	info, ok := cmdinfo[cmd]
	if cmd >= numCommands && ok {
		panic(fmt.Sprintf("command %d called but it's number is too high!", cmd))
	}
	c := (cmd << Indexshift) | uint32(info)
	//c now has all the deets

	//CMD6 may be Setbuswidth or Switchfunc depending on Appcmd prefix
	if cmd == Switchfunc && !emmc.appcmd {
		c |= Isdata | Card2host
	}
	if cmd == IORWextended {
		if arg&(1<<31) != 0 {
			c |= Host2card
		} else {
			c |= Card2host
		}
		if machine.EMMC.BlockSizeAndCount.Get()&0xFFFF0000 != 0x10000 {
			c |= Multiblock | Blkcnten
		}
	}
	//GoIdle indicates new card insertion: reset bus width & speed
	if cmd == GoIdle {
		machine.EMMC.Control0.Set(machine.EMMC.Control0.Get() & everythingButDwidthAndHighspeed)
		emmcclk(Initfreq)
	}
	// cmd inhibit
	if machine.EMMC.DebugStatus.Get()&Cmdinhibit != 0 {
		trust.Infof("emmccmd: need to reset Cmdinhibit intr %x stat %x\n",
			machine.EMMC.Interrupt.Get(), machine.EMMC.DebugStatus.Get())
		machine.EMMC.Control1.SetResetCommand()
		for machine.EMMC.DebugStatus.Get()&Cmdinhibit != 0 {
			delay(1)
		}
	}
	//data inhibit
	if ((c & Isdata) != 0) || ((c & Respmask) == Resp48busy) {
		trust.Infof("emmccmd: need to reset Datinhibit intr %x stat %x\n",
			machine.EMMC.Interrupt.Get(), machine.EMMC.DebugStatus.Get())
		machine.EMMC.Control1.SetResetData()
		for machine.EMMC.Control1.ResetDataIsSet() {
			delay(1)
		}
		for machine.EMMC.DebugStatus.Get()&Datinhibit != 0 {
			delay(1)
		}
	}
	//write arg and command
	machine.EMMC.Argument.Set(arg)
	i := machine.EMMC.Interrupt.Get()
	if i&everythingButCardIntr != 0 {
		if i != Cardinsert {
			trust.Infof("emmc: before command, intr was %x\n", i)
		}
		machine.EMMC.Interrupt.Set(i)
	}
	machine.EMMC.CommandTransferMode.Set(c)
	now := machine.SystemTime()
	//wait to see if we get a cmddone or err
	for {
		i = machine.EMMC.Interrupt.Get()
		if i&commandDoneOrError != 0 {
			break
		}
		if machine.SystemTime()-now > HZ { //wait 1 sec?
			break
		}
		trust.Debugf("waiting on cmd: %d", machine.SystemTime()-now)
	}
	//are we done?
	if i&commandDoneOrError != Cmddone {
		if i&everythingButCommandDoneOrError != Ctoerr {
			trust.Infof("emmc: cmd %x arg %x error intr %x stat %x\n",
				c, arg, i, machine.EMMC.DebugStatus.Get())
		}
		machine.EMMC.Interrupt.Set(i) //clears the interrupt
		if machine.EMMC.DebugStatus.Get()&Cmdinhibit != 0 {
			machine.EMMC.Control1.SetResetCommand()
			for machine.EMMC.Control1.ResetCommandIsSet() {
				delay(1)
			}
		}
		return Eio
	}
	if resp == nil {
		trust.Errorf("response required, but no response buffer provided!")
		return EBadArg
	}
	//clear anything that is NOT data related
	machine.EMMC.Interrupt.Set(i & everythingButCommandDoneOrError)
	//which type of resp?
	switch c & Respmask {
	case Resp136:
		if resp == nil {
			trust.Errorf("136bit response required, but no response buffer provided!")
			return EBadArg
		}
		resp[0] = machine.EMMC.Response0.Get() << 8
		resp[1] = machine.EMMC.Response0.Get()>>24 | machine.EMMC.Response1.Get()<<8
		resp[2] = machine.EMMC.Response1.Get()>>24 | machine.EMMC.Response2.Get()<<8
		resp[3] = machine.EMMC.Response2.Get()>>24 | machine.EMMC.Response3.Get()<<8
	case Resp48, Resp48busy:
		if resp == nil {
			trust.Errorf("response required, but no response buffer provided!")
			return EBadArg
		}
		resp[0] = machine.EMMC.Response0.Get()
	case Respnone:
		//nothing to do
	}
	//busy wait?
	if c&Respmask == Resp48busy {
		machine.EMMC.EnableInterrupt.Set(machine.EMMC.EnableInterrupt.Get() | dataDoneOrError)
		//xxx this should be a non-blocking call
		for j := 0; j < 30; j++ {
			if dataDone() {
				break
			}
			delay(1)
		}
		i := machine.EMMC.Interrupt.Get()
		if i&Datadone == 0 {
			trust.Errorf("emmcio: no datadone after CMD %d", cmd)
		}
		if i&Err != 0 {
			trust.Errorf("emmcio: command %d error interrupt %x", cmd,
				machine.EMMC.Interrupt.Get())
		}
		//clear interrupts
		machine.EMMC.Interrupt.Set(i)
	}
	//once card is selected, use faster clock
	if cmd == MMCSelect {
		delay(1)
		emmcclk(SDfreq)
		delay(1)
		emmc.fastclock = true
	}
	if cmd == Setbuswidth {
		if emmc.appcmd {
			// If card bus width changes, change host bus width
			switch arg {
			case 0:
				machine.EMMC.Control0.Set(machine.EMMC.Control0.Get() & everythingButDWidth4)
			case 2:
				machine.EMMC.Control0.Set(machine.EMMC.Control0.Get() | Dwidth4)
			}
		} else {
			//if card went to high speed mode, incr clock speed
			if (arg & 0x8000000F) == 0x80000001 {
				delay(1)
				emmcclk(SDfreqhs)
				delay(1)
			}
		}
		//nope, I got no idea what that constant is
	} else if cmd == IORWdirect && ((arg & everythingBufLowestByte) == (1<<31 | 0<<28 | 7<<9)) {
		switch arg & 0x3 {
		case 0:
			machine.EMMC.Control0.Set(machine.EMMC.Control0.Get() & everythingButDWidth4)
		case 2:
			machine.EMMC.Control0.Set(machine.EMMC.Control0.Get() | Dwidth4)
		}
	}
	emmc.appcmd = (cmd == Appcmd)
	return 0
}

//assume that buf is contiguous set of bcount buffers of size bsize?
func emmciosetup(buf unsafe.Pointer, bsize uint32, bcount uint32) {
	machine.EMMC.BlockSizeAndCount.Set(bcount<<16 | bsize)
}

// predicate to wait on
func dataDone() bool {
	return machine.EMMC.Interrupt.Get()&dataDoneOrError != 0
}

//xxx fix me
func mmcinterruptReceived() {
	i := machine.EMMC.Interrupt.Get()
	if i&dataDoneOrError != 0 {
		arm.Asm("nop")
		// wakeup r (rendez r)
	}
	if i&Cardintr != 0 {
		arm.Asm("nop")
		// wakeup cardr
	}
	machine.EMMC.Interrupt.Set(^i)
}
