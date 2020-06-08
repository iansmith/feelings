package joy

import (
	"fmt"

	"machine"

	"lib/trust"
	"lib/upbeat"
)

func initExceptionVector()

func KernelMain() {
	initExceptionVector()
	logger := initVideo()
	displayInfo(logger)
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

func displayInfo(logger *trust.Logger) {
	var size, base uint32

	logger.Infof("#")
	logger.Infof("# joy")
	logger.Infof("#")

	id, ok := upbeat.BoardID()
	if ok == false {
		fmt.Printf("can't get board id, aborting\n")
		machine.Abort()
	}
	logger.Infof("board id         : %016x\n", id)

	v, ok := upbeat.FirmwareVersion()
	if ok == false {
		fmt.Printf("can't get firmware version id, aborting\n")
		machine.Abort()
	}
	logger.Infof("firmware version : %08x\n", v)

	rev, ok := upbeat.BoardRevision()
	if ok == false {
		fmt.Printf("can't get board revision id, aborting\n")
		return
	}
	logger.Infof("board revision   : %08x %s\n", rev, upbeat.BoardRevisionDecode(fmt.Sprintf("%x", rev)))

	cr, ok := upbeat.GetClockRate()
	if ok == false {
		fmt.Printf("can't get clock rate, aborting\n")
		machine.Abort()

	}
	logger.Infof("clock rate       : %d hz\n", cr)

	base, size, ok = upbeat.GetARMMemoryAndBase()
	if ok == false {
		fmt.Printf("can't get arm memory, aborting\n")
		machine.Abort()
	}
	logger.Infof("ARM Memory       : 0x%x bytes @ 0x%x\n", size, base)

	base, size, ok = upbeat.GetVCMemoryAndBase()
	if ok == false {
		fmt.Printf("can't get vc memory, aborting\n")
		machine.Abort()
	}
	logger.Infof("VidCore IV Memory: 0x%x bytes @ 0x%x\n", size, base)
}
