package joy

import (
	"fmt"
	tgr "runtime"

	"device/arm"
	"machine"

	"lib/trust"
	"lib/upbeat"
)

//go:external init_exception_vector
func initExceptionVector()

var terminalTestPtr FuncPtr

//go:export kernel_main
func KernelMain() {
	tgr.ReInit()
	initExceptionVector()

	err := KMemoryInit()
	if err != JoyNoError {
		panic(JoyErrorMessage(err))
	}
	InitDomains()
	InitGIC()
	InitSchedulingTimer()

	err = DomainCopy(displayInfoPtr, 0)
	if err != JoyNoError {
		trust.Errorf("unable to start display info process:", JoyErrorMessage(err))
		return
	}
	err = DomainCopy(terminalTestPtr, 0)
	if err != JoyNoError {
		trust.Errorf("unable to start terminal test process:", JoyErrorMessage(err))
		return
	}

	EnableIRQAndFIQ()
	for {
		schedule()
	}
}

func initVideo() *trust.Logger {
	fmt.Printf("init video started in D1\n")
	// info := upbeat.SetFramebufferRes1920x1200()
	// if info == nil {
	// 	panic("giving up, can't set framebuffer res")
	// }

	info := upbeat.SetFramebufferRes1024x768()
	fmt.Printf("fb set in D1\n")
	if info == nil {
		panic("can't set the framebuffer res, aborting")
		machine.Abort()
	}

	fmt.Printf("about to hit console create D1\n")
	logger := upbeat.NewConsoleLogger(info)
	return logger
}

//go:extern
var displayInfoPtr FuncPtr

//go:export joy.displayInfo
func displayInfo(_ uintptr) {
	tgr.ReInit()

	var size, base uint32
	logger := initVideo()
	sleepForFew()
	logger.Sink().(*upbeat.FBConsole).Clear()
	logger.Infof("#")
	logger.Infof("# joy")
	logger.Infof("#")
	sleepForFew()

	id, ok := upbeat.BoardID()
	if ok == false {
		fmt.Printf("can't get board id, aborting\n")
		machine.Abort()
	}
	logger.Infof("board id         : %016x\n", id)
	trust.Infof("board id         : %016x\n", id)
	sleepForFew()

	v, ok := upbeat.FirmwareVersion()
	if ok == false {
		fmt.Printf("can't get firmware version id, aborting\n")
		machine.Abort()
	}
	logger.Infof("firmware version : %08x\n", v)
	trust.Infof("firmware version : %08x\n", v)
	sleepForFew()

	rev, ok := upbeat.BoardRevision()
	if ok == false {
		fmt.Printf("can't get board revision id, aborting\n")
		return
	}
	logger.Infof("board revision   : %08x %s\n", rev, upbeat.BoardRevisionDecode(fmt.Sprintf("%x", rev)))
	trust.Infof("board revision   : %08x %s\n", rev, upbeat.BoardRevisionDecode(fmt.Sprintf("%x", rev)))
	sleepForFew()

	cr, ok := upbeat.GetClockRate()
	if ok == false {
		fmt.Printf("can't get clock rate, aborting\n")
		machine.Abort()

	}
	logger.Infof("clock rate       : %d hz\n", cr)
	trust.Infof("clock rate       : %d hz\n", cr)
	sleepForFew()

	base, size, ok = upbeat.GetARMMemoryAndBase()
	if ok == false {
		fmt.Printf("can't get arm memory, aborting\n")
		machine.Abort()
	}
	logger.Infof("ARM Memory       : 0x%x bytes @ 0x%x\n", size, base)
	trust.Infof("ARM Memory       : 0x%x bytes @ 0x%x\n", size, base)
	sleepForFew()

	base, size, ok = upbeat.GetVCMemoryAndBase()
	if ok == false {
		fmt.Printf("can't get vc memory, aborting\n")
		machine.Abort()
	}
	logger.Infof("VidCore IV Memory: 0x%x bytes @ 0x%x\n", size, base)
	trust.Infof("VidCore IV Memory: 0x%x bytes @ 0x%x\n", size, base)
}

func sleepForFew() {
	for i := 0; i < 1000000; i++ {
		arm.Asm("nop")
	}
}

//go:export joy.terminalTest
func terminalTest(ptr uintptr) {
	tgr.ReInit()
	ct := 0
	for {
		fmt.Printf("terminal test: hi! #%d\n", ct)
		ct++
		sleepForFew()
	}
}
