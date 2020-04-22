package main

import (
	"feelings/src/hardware/bcm2835"
	rt "feelings/src/tinygo_runtime"

	"github.com/tinygo-org/tinygo/src/device/arm"
)

//export raw_exception_handler
func raw_exception_handler() {
	rt.MiniUART.WriteString("TRAPPED INTR\n")
}

var sd_ocr, sd_rca, sd_err, sd_hv uint64
var sd_scr [2]uint64

func main() {
	rt.MiniUART = rt.NewUART()
	_ = rt.MiniUART.Configure(rt.UARTConfig{ /*no interrupt*/ })

	//u := unsafe.Offsetof(bcm2835.GPIO.PullUpDownEnableClock1)
	//rt.MiniUART.Hex64string(uint64(u))

	if err := sdInit(); err == nil {
		_ = rt.MiniUART.WriteString("initialized the card ok\n")
	} else {
		_ = rt.MiniUART.WriteString("unable to init card: ")
		rt.MiniUART.WriteString(err.Error())

	}
	rt.MiniUART.WriteCR()
	for {
		arm.Asm("nop")
	}

}
func sdWaitForInterrupt(mask uint32) int {
	var r int
	var m = uint32(mask | bcm2835.InterruptErrorMask)
	cnt := 1000000
	for (bcm2835.EMCC.Interrupt.Get()&m == 0) && cnt > 0 {
		rt.WaitMuSec(1)
		cnt--
		if cnt%1000 == 0 {
			rt.MiniUART.WriteByte('.')
		}
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

//unsigned int r, m=mask | INT_ERROR_MASK;
//int cnt = 1000000; while(!(*EMMC_INTERRUPT & m) && cnt--) wait_msec(1);
//r=*EMMC_INTERRUPT;
//if(cnt<=0 || (r & INT_CMD_TIMEOUT) || (r & INT_DATA_TIMEOUT) ) { *EMMC_INTERRUPT=r; return SD_TIMEOUT; } else
//if(r & INT_ERROR_MASK) { *EMMC_INTERRUPT=r; return SD_ERROR; }
//*EMMC_INTERRUPT=mask;
//return 0;

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

	//int sd_cmd(unsigned int code, unsigned int arg)
	//{
	//int r=0;
	//sd_err=SD_OK;
	//if(code&CMD_NEED_APP) {
	//r=sd_cmd(CMD_APP_CMD|(sd_rca?CMD_RSPNS_48:0),sd_rca);
	//if(sd_rca && !r) { uart_puts("ERROR: failed to send SD APP command\n"); sd_err=SD_ERROR;return 0;}
	//code &= ~CMD_NEED_APP;
	//}
	if sdStatus(bcm2835.SRCommandInhibit) > 0 {
		rt.MiniUART.WriteString("ERROR: EMMC Busy\n")
		sd_err = bcm2835.SDTimeoutUnsigned //uint64(int64(bcm2835.SDTimeout))
		return 0
	}
	rt.MiniUART.WriteString("sending command ")
	rt.MiniUART.Hex32string(code)
	rt.MiniUART.WriteString(" arg ")
	rt.MiniUART.Hex32string(arg)
	rt.MiniUART.WriteString("\n")

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
	//*EMMC_INTERRUPT=*EMMC_INTERRUPT; *EMMC_ARG1=arg; *EMMC_CMDTM=code;
	//if(code==CMD_SEND_OP_COND) wait_msec(1000); else
	//if(code==CMD_SEND_IF_COND || code==CMD_APP_CMD) wait_msec(100);
	//if((r=sd_int(INT_CMD_DONE))) {uart_puts("ERROR: failed to send EMMC command\n");sd_err=r;return 0;}

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

//r=*EMMC_RESP0;
//if(code==CMD_GO_IDLE || code==CMD_APP_CMD) return 0; else
//if(code==(CMD_APP_CMD|CMD_RSPNS_48)) return r&SR_APP_CMD; else
//if(code==CMD_SEND_OP_COND) return r; else
//if(code==CMD_SEND_IF_COND) return r==arg? SD_OK : SD_ERROR; else
//if(code==CMD_ALL_SEND_CID) {r|=*EMMC_RESP3; r|=*EMMC_RESP2; r|=*EMMC_RESP1; return r; } else
//if(code==CMD_SEND_REL_ADDR) {
//sd_err=(((r&0x1fff))|((r&0x2000)<<6)|((r&0x4000)<<8)|((r&0x8000)<<8))&CMD_ERRORS_MASK;
//return r&CMD_RCA_MASK;
//}
//return r&CMD_ERRORS_MASK;
//// make gcc happy
//return 0;
//}

func sdInit() error {
	//bcm2835.GPIOSetup(7, bcm2835.GPIOAltFunc4 /*entry 3*/)
	//waitOnPullUps(1 << 15)
	//bcm2835.GPIO.HighDetectEnable1.SetBits(1 << 15)
	//rt.MiniUART.Hex32string(0xff)
	var r, ccs, cnt int
	r = int(bcm2835.GPIO.FuncSelect[4].Get())

	rt.MiniUART.WriteString("sdinit, func select 4, pin 7 ")
	rt.MiniUART.Hex64string(uint64(r))
	comp := ^(int(7) << int((7 * 3)))
	r = r & comp //clearing the pin seven entries?
	rt.MiniUART.Hex64string(uint64(r))
	rt.MiniUART.WriteCR()
	bcm2835.GPIO.FuncSelect[4].Set(uint32(r))
	waitOnPullUps(1 << 15)

	//// GPIO_CD
	//r=*GPFSEL4; r&=~(7<<(7*3)); *GPFSEL4=r;
	//*GPPUD=2; wait_cycles(150); *GPPUDCLK1=(1<<15); wait_cycles(150); *GPPUD=0; *GPPUDCLK1=0;
	//r=*GPHEN1; r|=1<<15; *GPHEN1=r;

	// GPIO_CLK
	r = int(bcm2835.GPIO.FuncSelect[4].Get())
	rt.MiniUART.WriteString("sdinit, func select 4, pins 8 and 9 ")
	rt.MiniUART.Hex64string(uint64(r))
	r = r | ((int(7) << (8 * 3)) | (int(7 << (9 * 3))))
	rt.MiniUART.Hex64string(uint64(r))
	rt.MiniUART.WriteCR()
	bcm2835.GPIO.FuncSelect[4].Set(uint32(r))

	//bcm2835.GPIOSetup(8, bcm2835.GPIOAltFunc4 /*entry 3*/)
	//bcm2835.GPIOSetup(9, bcm2835.GPIOAltFunc4 /*entry 3*/)
	waitOnPullUps((1 << 16) | (1 << 17))
	rt.MiniUART.Hex32string(0x00)

	// GPIO_CLK, GPIO_CMD
	//r = *GPFSEL4
	//r |= (7 << (8 * 3)) | (7 << (9 * 3))
	//*GPFSEL4 = r
	//*GPPUD = 2
	//wait_cycles(150)
	//*GPPUDCLK1 = (1 << 16) | (1 << 17)
	//wait_cycles(150)
	//*GPPUD = 0
	//*GPPUDCLK1 = 0

	// GPIO_DAT0, GPIO_DAT1, GPIO_DAT2, GPIO_DAT3
	//bcm2835.GPIOSetup(0, bcm2835.GPIOAltFunc5)
	//bcm2835.GPIOSetup(1, bcm2835.GPIOAltFunc5)
	//bcm2835.GPIOSetup(2, bcm2835.GPIOAltFunc5)
	//bcm2835.GPIOSetup(3, bcm2835.GPIOAltFunc5 /*entry 4*/)
	r = int(bcm2835.GPIO.FuncSelect[5].Get())
	rt.MiniUART.WriteString("sdinit, func select 5, pin 0-3 ")
	rt.MiniUART.Hex64string(uint64(r))
	r = r | ((int(7 << (0 * 3))) | (int(7 << (1 * 3))) | (int(7 << (2 * 3))) | (int(7 << (3 * 3))))
	rt.MiniUART.Hex64string(uint64(r))
	bcm2835.GPIO.FuncSelect[5].Set(uint32(r))
	rt.MiniUART.WriteCR()
	waitOnPullUps((1 << 18) | (1 << 19) | (1 << 20) | (1 << 21))

	// GPIO_DAT0, GPIO_DAT1, GPIO_DAT2, GPIO_DAT3
	//r = *GPFSEL5
	//r |= (7 << (0 * 3)) | (7 << (1 * 3)) | (7 << (2 * 3)) | (7 << (3 * 3))
	//*GPFSEL5 = r
	//*GPPUD = 2
	//wait_cycles(150)
	//*GPPUDCLK1 = (1 << 18) | (1 << 19) | (1 << 20) | (1 << 21)
	//wait_cycles(150)
	//*GPPUD = 0
	//*GPPUDCLK1 = 0

	sdHardwareVersion := (bcm2835.EMCC.SlotInterruptStatus.Get() & bcm2835.HostSpecNum) >> bcm2835.HostSpecNumShift
	rt.MiniUART.WriteString("EMMC: GPIO set up\n")
	rt.MiniUART.Hex32string(sdHardwareVersion)
	// Reset the card.
	bcm2835.EMCC.Control0.Set(0)
	bcm2835.EMCC.Control1.SetBits(bcm2835.C1ResetHost)
	cnt = 10000
	for (bcm2835.EMCC.Control1.Get()&uint32(bcm2835.C1ResetHost) != 0) && cnt > 0 {
		rt.WaitMuSec(10)
		cnt--
	}
	rt.MiniUART.WriteString("control1 and reset host\n")
	rt.MiniUART.Hex32string(bcm2835.EMCC.Control1.Get() & uint32(bcm2835.C1ResetHost))
	rt.MiniUART.WriteCR()

	if cnt <= 0 {
		rt.MiniUART.WriteString("Failed to reset EMCC\n")
		return bcm2835.NewSDInitFailure("Unable to reset EMCC")
	}
	//*EMMC_CONTROL0 = 0; *EMMC_CONTROL1 |= C1_SRST_HC;
	//cnt=10000; do{wait_msec(10);} while( (*EMMC_CONTROL1 & C1_SRST_HC) && cnt-- );
	//if(cnt<=0) {
	//	uart_puts("ERROR: failed to reset EMMC\n");
	//	return SD_ERROR;
	//}

	//setup clocks
	bcm2835.EMCC.Control1.SetBits(bcm2835.C1ClockEnableInternal | bcm2835.C1_TOUNIT_MAX)
	rt.MiniUART.WriteString("fleazil ")
	rt.WaitMuSec(10)
	rt.MiniUART.WriteByte('?')
	rt.MiniUART.Hex32string(bcm2835.EMCC.Control1.Get())

	//*EMMC_CONTROL1 |= C1_CLK_INTLEN | C1_TOUNIT_MAX;
	rt.WaitMuSec(10)
	rt.MiniUART.WriteString("blargh ")
	// Set clock to setup frequency.
	err := sdSetClockToFreq(400000, sdHardwareVersion)
	if err != nil {
		return err
	}
	//if((r=)) return r;
	bcm2835.EMCC.InterruptEnable.Set(0xffffffff)
	bcm2835.EMCC.InterruptMask.Set(0xffffffff)
	sd_scr[0] = 0
	sd_scr[1] = 0
	sd_rca = 0
	sd_err = 0
	//*EMMC_INT_EN   = 0xffffffff;
	//*EMMC_INT_MASK = 0xffffffff;
	//sd_scr[0]=sd_scr[1]=sd_rca=sd_err=0;

	sdSendCommand(bcm2835.CommandGoIdle, 0)
	if sd_err != 0 {
		return bcm2835.NewSDInitFailure("unable to get card to go idle")
	}

	sdSendCommand(bcm2835.CommandSendIfCond, 0x000001AA)
	if sd_err != 0 {
		return bcm2835.NewSDInitFailure("unable to send command if cond")
	}

	rt.MiniUART.WriteString("6 tries at cmd complete\n ")
	cnt = 6
	r = 0
	for (r&bcm2835.ACMD41_CMD_COMPLETE) == 0 && cnt > 0 {
		rt.MiniUART.WriteString("r for mask test ")
		rt.MiniUART.Hex64string(uint64(r))
		rt.MiniUART.Hex64string(bcm2835.ACMD41_CMD_COMPLETE)
		rt.MiniUART.WriteCR()
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
		rt.MiniUART.WriteString("r is 64bit ")
		rt.MiniUART.Hex64string(uint64(r))
		rt.MiniUART.WriteCR()
		if sd_err != bcm2835.SDTimeoutUnsigned && sd_err != bcm2835.SDOk {
			rt.MiniUART.WriteString("ERROR: EMMC ACMD41 returned error")
			rt.MiniUART.Hex64string(sd_err)
			rt.MiniUART.WriteCR()
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

	rt.MiniUART.WriteString("ccs is ")
	rt.MiniUART.Hex64string(uint64(ccs))
	rt.MiniUART.WriteCR()

	sdSendCommand(bcm2835.CommandAllSendCID, 0)
	sd_rca = uint64(sdSendCommand(bcm2835.CommandSendRelAddr, 0))
	rt.MiniUART.WriteString("EMMC: CMD_SEND_REL_ADDR returned ")
	rt.MiniUART.Hex64string(sd_rca)
	rt.MiniUART.WriteCR()
	if sd_err != 0 {
		return bcm2835.NewSDInitFailure("Command Send Relative Addr Failed")
		//return sd_err;
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
	//while((*EMMC_STATUS & (SR_CMD_INHIBIT|SR_DAT_INHIBIT)) && cnt--) wait_msec(1);
	//if(cnt<=0) {
	//	uart_puts("ERROR: timeout waiting for inhibit flag\n");
	//	return SD_ERROR;
	//}

	c1 := bcm2835.EMCC.Control1.Get()
	comp := ^(uint32(bcm2835.C1ClockEnable))
	c1 = c1 & comp
	bcm2835.EMCC.Control1.Set(c1)

	//bcm2835.EMCC.Control1.ClearBits(bcm2835.C1ClockEnable)
	rt.WaitMuSec(10)
	//*EMMC_CONTROL1 &= ~C1_CLK_EN; wait_msec(10);

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
	//x=c-1; if(!x) s=0; else {
	//	if(!(x & 0xffff0000u)) { x <<= 16; s -= 16; }
	//	if(!(x & 0xff000000u)) { x <<= 8;  s -= 8; }
	//	if(!(x & 0xf0000000u)) { x <<= 4;  s -= 4; }
	//	if(!(x & 0xc0000000u)) { x <<= 2;  s -= 2; }
	//	if(!(x & 0x80000000u)) { x <<= 1;  s -= 1; }
	//	if(s>0) s--;
	//	if(s>7) s=7;
	//}

	if hwVersion > bcm2835.HostSpecV2 {
		d = c
	} else {
		d = 1 << s
	}
	if d <= 2 {
		d = 2
		s = 0
	}
	//if(sd_hv>HOST_SPEC_V2) d=c; else d=(1<<s);
	//if(d<=2) {d=2;s=0;}
	_ = rt.MiniUART.WriteString("sd clock divisor ")
	rt.MiniUART.Hex32string(d)
	_ = rt.MiniUART.WriteString("shift ")
	rt.MiniUART.Hex32string(s)
	_ = rt.MiniUART.WriteByte('\n')
	//	uart_puts("sd_clk divisor ");uart_hex(d);uart_puts(", shift ");uart_hex(s);uart_puts("\n");

	if hwVersion > bcm2835.HostSpecV2 {
		h = (d & 0x300) >> 2
	}
	d = (((d & 0x0ff) << 8) | h)
	//if(sd_hv>HOST_SPEC_V2) h=(d&0x300)>>2;
	//d=(((d&0x0ff)<<8)|h);

	r := bcm2835.EMCC.Control1.Get()
	r = (r & 0xffff003f) | d
	bcm2835.EMCC.Control1.Set(r)
	//bcm2835.EMCC.Control1.ClearBits(0xffC0)
	rt.WaitMuSec(10)
	r = r | bcm2835.C1ClockEnable
	bcm2835.EMCC.Control1.Set(r)
	rt.WaitMuSec(10)

	rt.MiniUART.WriteString("control 1 after clock")
	rt.MiniUART.Hex32string(bcm2835.EMCC.Control1.Get())
	rt.MiniUART.WriteCR()

	cnt = 10000
	for bcm2835.EMCC.Control1.Get()&bcm2835.C1ClockStable == 0 && cnt > 0 {
		cnt--
	}
	if cnt < 0 {
		_ = rt.MiniUART.WriteString("timeout waiting for stable clock\n")
		return bcm2835.NewSDInitFailure("timeout waiting for stable clock\n")
	}
	return nil
	//*EMMC_CONTROL1=(*EMMC_CONTROL1&0xffff003f)|d; wait_msec(10);
	//*EMMC_CONTROL1 |= C1_CLK_EN; wait_msec(10);
	//cnt=10000; while(!(*EMMC_CONTROL1 & C1_CLK_STABLE) && cnt--) wait_msec(10);
	//if(cnt<=0) {
	//	uart_puts("ERROR: failed to get stable clock\n");
	//	return SD_ERROR;
	//}
	//return SD_OK;

}

func waitOnPullUps(valueToSet uint32) {
	bcm2835.GPIO.PullUpDownEnable.Set(2)
	for i := 0; i < 150; i++ {
		arm.Asm("nop")
	}
	bcm2835.GPIO.PullUpDownEnableClock1.Set(valueToSet)
	for i := 0; i < 150; i++ {
		arm.Asm("nop")
	}
	bcm2835.GPIO.PullUpDownEnable.Set(0)
	bcm2835.GPIO.PullUpDownEnableClock1.Set(0)

}

/*
// GPIO_CD
r=*GPFSEL4; r&=~(7<<(7*3)); *GPFSEL4=r;
*GPPUD=2; wait_cycles(150); *GPPUDCLK1=(1<<15); wait_cycles(150); *GPPUD=0; *GPPUDCLK1=0;
r=*GPHEN1; r|=1<<15; *GPHEN1=r;

// GPIO_CLK, GPIO_CMD
r=*GPFSEL4; r|=(7<<(8*3))|(7<<(9*3)); *GPFSEL4=r;
*GPPUD=2; wait_cycles(150); *GPPUDCLK1=(1<<16)|(1<<17); wait_cycles(150); *GPPUD=0; *GPPUDCLK1=0;

// GPIO_DAT0, GPIO_DAT1, GPIO_DAT2, GPIO_DAT3
r=*GPFSEL5; r|=(7<<(0*3)) | (7<<(1*3)) | (7<<(2*3)) | (7<<(3*3)); *GPFSEL5=r;
*GPPUD=2; wait_cycles(150);
*GPPUDCLK1=(1<<18) | (1<<19) | (1<<20) | (1<<21);
wait_cycles(150); *GPPUD=0; *GPPUDCLK1=0;

sd_hv = (*EMMC_SLOTISR_VER & HOST_SPEC_NUM) >> HOST_SPEC_NUM_SHIFT;
uart_puts("EMMC: GPIO set up\n");
// Reset the card.
*EMMC_CONTROL0 = 0; *EMMC_CONTROL1 |= C1_SRST_HC;
cnt=10000; do{wait_msec(10);} while( (*EMMC_CONTROL1 & C1_SRST_HC) && cnt-- );
if(cnt<=0) {
uart_puts("ERROR: failed to reset EMMC\n");
return SD_ERROR;
}
uart_puts("EMMC: reset OK\n");
*EMMC_CONTROL1 |= C1_CLK_INTLEN | C1_TOUNIT_MAX;
wait_msec(10);
// Set clock to setup frequency.
if((r=sd_clk(400000))) return r;
*EMMC_INT_EN   = 0xffffffff;
*EMMC_INT_MASK = 0xffffffff;
sd_scr[0]=sd_scr[1]=sd_rca=sd_err=0;
sd_cmd(CMD_GO_IDLE,0);
if(sd_err) return sd_err;

sd_cmd(CMD_SEND_IF_COND,0x000001AA);
if(sd_err) return sd_err;
cnt=6; r=0; while(!(r&ACMD41_CMD_COMPLETE) && cnt--) {
wait_cycles(400);
r=sd_cmd(CMD_SEND_OP_COND,ACMD41_ARG_HC);
uart_puts("EMMC: CMD_SEND_OP_COND returned ");
if(r&ACMD41_CMD_COMPLETE)
uart_puts("COMPLETE ");
if(r&ACMD41_VOLTAGE)
uart_puts("VOLTAGE ");
if(r&ACMD41_CMD_CCS)
uart_puts("CCS ");
uart_hex(r>>32);
uart_hex(r);
uart_puts("\n");
if(sd_err!=SD_TIMEOUT && sd_err!=SD_OK ) {
uart_puts("ERROR: EMMC ACMD41 returned error\n");
return sd_err;
}
}
if(!(r&ACMD41_CMD_COMPLETE) || !cnt ) return SD_TIMEOUT;
if(!(r&ACMD41_VOLTAGE)) return SD_ERROR;
if(r&ACMD41_CMD_CCS) ccs=SCR_SUPP_CCS;

sd_cmd(CMD_ALL_SEND_CID,0);

sd_rca = sd_cmd(CMD_SEND_REL_ADDR,0);
uart_puts("EMMC: CMD_SEND_REL_ADDR returned ");
uart_hex(sd_rca>>32);
uart_hex(sd_rca);
uart_puts("\n");
if(sd_err) return sd_err;

if((r=sd_clk(25000000))) return r;

sd_cmd(CMD_CARD_SELECT,sd_rca);
if(sd_err) return sd_err;

if(sd_status(SR_DAT_INHIBIT)) return SD_TIMEOUT;
*EMMC_BLKSIZECNT = (1<<16) | 8;
sd_cmd(CMD_SEND_SCR,0);
if(sd_err) return sd_err;
if(sd_int(INT_READ_RDY)) return SD_TIMEOUT;

r=0; cnt=100000; while(r<2 && cnt) {
if( *EMMC_STATUS & SR_READ_AVAILABLE )
sd_scr[r++] = *EMMC_DATA;
else
wait_msec(1);
}
if(r!=2) return SD_TIMEOUT;
if(sd_scr[0] & SCR_SD_BUS_WIDTH_4) {
sd_cmd(CMD_SET_BUS_WIDTH,sd_rca|2);
if(sd_err) return sd_err;
*EMMC_CONTROL0 |= C0_HCTL_DWITDH;
}
// add software flag
uart_puts("EMMC: supports ");
if(sd_scr[0] & SCR_SUPP_SET_BLKCNT)
uart_puts("SET_BLKCNT ");
if(ccs)
uart_puts("CCS ");
uart_puts("\n");
sd_scr[0]&=~SCR_SUPP_CCS;
sd_scr[0]|=ccs;
return SD_OK;
}
*/

func waitCycles(n int) {
	for n > 0 {
		arm.Asm("nop")
		n--
	}
}
