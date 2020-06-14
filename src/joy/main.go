package joy

import (
	"fmt"

	"device/arm"
	"machine"

	"lib/trust"
	"lib/upbeat"
)

//go:external init_exception_vector
func initExceptionVector()

func KernelMain() {
	//slightly tricky why this is necessary: yes, the uart is already configured
	//by the bootloader... however, the LINKER puts a reference to MiniUART in this
	//binary which is not initialized so if you try to call something like
	//machine.MiniUART.WriteString() you get a nil pointer exception.
	machine.MiniUART = machine.NewUART()
	machine.MiniUART.Configure(&machine.UARTConfig{})
	machine.MiniUART.WriteString("yyz 0\n")

	fmt.Printf("xxx %p\n", trust.DefaultLogger)

	machine.MiniUART.WriteString("yyz 1\n")
	//initializers definitely haven't been run... maybe zeroing has?
	trust.DefaultLogger.SetLevel(trust.ErrorMask | trust.DebugMask | trust.InfoMask | trust.WarnMask)
	machine.MiniUART.WriteString("yyz 2\n")
	initExceptionVector()
	machine.MiniUART.WriteString("yyz 3\n")
	stack, heapStart, heapEnd, err := KMemoryInit()
	machine.MiniUART.WriteString("yyz 4\n")
	if err != JoyNoError {
		panic(JoyErrorMessage(err))
	}
	machine.MiniUART.WriteString("yyz 5\n")
	InitDomains(stack, heapStart, heapEnd)
	machine.MiniUART.WriteString("yyz 6\n")
	InitGIC()
	machine.MiniUART.WriteString("yyz 7\n")
	InitSchedulingTimer()
	machine.MiniUART.WriteString("yyz 8\n")
	EnableIRQAndFIQ()
	machine.MiniUART.WriteString("yyz 9\n")

	trust.Debugf("about to copy 1")
	err = DomainCopy(displayInfo, 0)
	if err != JoyNoError {
		trust.Errorf("unable to start display info process:", JoyErrorMessage(err))
		return
	}
	trust.Debugf("about to copy 2")
	err = DomainCopy(terminalTest, 0)
	if err != JoyNoError {
		trust.Errorf("unable to start terminal test process:", JoyErrorMessage(err))
		return
	}
	for {
		schedule()
	}
}

func initVideo() *trust.Logger {
	// info := upbeat.SetFramebufferRes1920x1200()
	// if info == nil {
	// 	panic("giving up, can't set framebuffer res")
	// }

	info := upbeat.SetFramebufferRes1024x768()
	if info == nil {
		panic("can't set the framebuffer res, aborting")
		machine.Abort()
	}

	logger := upbeat.NewConsoleLogger(info)
	return logger
}

func displayInfo(_ uintptr) {
	var size, base uint32
	logger := initVideo()
	sleepForFew()

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
	sleepForFew()

	v, ok := upbeat.FirmwareVersion()
	if ok == false {
		fmt.Printf("can't get firmware version id, aborting\n")
		machine.Abort()
	}
	logger.Infof("firmware version : %08x\n", v)
	sleepForFew()

	rev, ok := upbeat.BoardRevision()
	if ok == false {
		fmt.Printf("can't get board revision id, aborting\n")
		return
	}
	logger.Infof("board revision   : %08x %s\n", rev, upbeat.BoardRevisionDecode(fmt.Sprintf("%x", rev)))
	sleepForFew()

	cr, ok := upbeat.GetClockRate()
	if ok == false {
		fmt.Printf("can't get clock rate, aborting\n")
		machine.Abort()

	}
	logger.Infof("clock rate       : %d hz\n", cr)
	sleepForFew()

	base, size, ok = upbeat.GetARMMemoryAndBase()
	if ok == false {
		fmt.Printf("can't get arm memory, aborting\n")
		machine.Abort()
	}
	logger.Infof("ARM Memory       : 0x%x bytes @ 0x%x\n", size, base)
	sleepForFew()

	base, size, ok = upbeat.GetVCMemoryAndBase()
	if ok == false {
		fmt.Printf("can't get vc memory, aborting\n")
		machine.Abort()
	}
	logger.Infof("VidCore IV Memory: 0x%x bytes @ 0x%x\n", size, base)
}

func sleepForFew() {
	for i := 0; i < 1000000; i++ {
		arm.Asm("nop")
	}
}
func terminalTest(ptr uintptr) {
	ct := 0
	for {
		fmt.Printf("terminal test: hi! #%d\n", ct)
		ct++
		sleepForFew()
	}
}
