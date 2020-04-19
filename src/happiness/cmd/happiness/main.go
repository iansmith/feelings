package main

import (
	"feelings/src/happiness"
	arm64 "feelings/src/hardware/arm-cortex-a53"
	"feelings/src/hardware/videocore"
	"feelings/src/lib/semihosting"
	rt "feelings/src/tinygo_runtime"
	"strings"
)

func main() {
	rt.MiniUART = rt.NewUART()
	rt.MiniUART.Configure(rt.UARTConfig{})

	var exceptionLevelName string
	happiness.Console.Logf("# Stage 1 kernel is running: Happiness")
	happiness.Console.Logf("# %16s : %d ", "Unix time", rt.BootArg0)
	happiness.Console.Logf("# %16s : 0x%08x ", "Boot Address", rt.BootArg1)
	switch rt.BootArg2 {
	case 0:
		exceptionLevelName = "(User)"
	case 1:
		exceptionLevelName = "(Kernel)"
	case 2:
		exceptionLevelName = "(Hypervisor)"
	case 3:
		exceptionLevelName = "(Secure Kernel)"
	}
	happiness.Console.Logf("# %16s : %d %s ", "Exception level", rt.BootArg2, exceptionLevelName)
	bid, ok := videocore.BoardID()
	if !ok {
		happiness.Console.Logf("!   Failed To read Board ID!")
	} else {
		happiness.Console.Logf("# %16s : %08x%08x ", "Board Id", bid>>32, bid&(0xffffffff<<32))
	}
	//for i := 0; i < 100000000; i++ {
	//	arm.Asm("nop")
	//}
	firmware, ok := videocore.FirmwareVersion()
	if !ok {
		happiness.Console.Logf("!   Failed To read Firmware Version!")
	} else {
		happiness.Console.Logf("# %16s : %d ", "Firmware version", firmware)
	}
	boardModel, ok := videocore.BoardModel()
	if !ok {
		happiness.Console.Logf("!   Failed To read Board Model!")
	} else {
		happiness.Console.Logf("# %16s : %d ", "Board Model", boardModel)
	}
	boardRevision, ok := videocore.BoardRevision()
	if !ok {
		happiness.Console.Logf("!   Failed To read Board Revision!")
	} else {
		br := strings.ToLower(happiness.Console.Sprintf("%x", boardRevision))
		happiness.Console.Logf("# %16s : %s (%s)", "Board Revision", br, boardRevisionDecode(br))
	}
	addr, ok := videocore.MACAddress()
	printableMAC := happiness.Console.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
		addr>>0x30&0xff,
		addr>>0x28&0xff, addr>>0x20&0xff, addr>>0x18&0xff,
		addr>>0x10&0xff, addr>>8&0xff, addr&0xff)

	if !ok {
		happiness.Console.Logf("!   Failed to read the MAC Address!")
	} else {
		happiness.Console.Logf("# %16s : %s ", "MAC Address", printableMAC)
	}
	//interrupts start as off
	arm64.InitExceptionVector()
	happiness.TestSprintfGoodCases()
	/*
		bcm2835.InterruptController.EnableTimer()
		bcm2835.InterruptController.EnableIRQs1()
	*/

	//rt.MiniUART.WriteString("hello, world.\n")
	//Console.Logf("%%hello%% %-5s,%010d,0x%01x,0x%-06x\n", "1234", 635535, 0x01f, 4096)

	/*
		taskList := makeTaskList()          //make the task list
		taskList.AssignTask(makeInitTask()) //put the init task on it
			result:= taskList.CopyProcess(process,
			int res = copy_process((unsigned long)&process, (unsigned long)"12345");
			if (res != 0) {
				printf("error while starting process 1");
				return;
			}
			res = copy_process((unsigned long)&process, (unsigned long)"abcde");
			if (res != 0) {
				printf("error while starting process 2");
				return;
			}

			while (1){
				schedule();
			}
	*/
	semihosting.Exit(0)
}

func boardRevisionDecode(s string) string {
	switch s {
	case "9020e0":
		return "3A+, Revision 1.0, 512MB, Sony UK"
	case "a02082":
		return "3B, Revision 1.2, 1GB, Sony UK"
	case "a020d3":
		return "3B+, Revision 1.3, 1GB, Sony UK"
	case "a22082":
		return "3B, Revision 1.2, 1GB, Embest"
	case "a220a0":
		return "CM3, Revision 1.0, 1GB, Embest"
	case "a32082":
		return "3B, Revision 1.2, 1GB, Sony Japan"
	case "a52082":
		return "3B, Revision 1.2, 1GB, Stadium"
	case "a22083":
		return "3B, Revision 1.3, 1GB, Embest"
	case "a02100":
		return "CM3+, Revision 1.0, 1GB, Sony UK"
	case "a03111":
		return "4B, Revision 1.1, 2GB, Sony UK"
	case "b03111":
		return "4B, Revision 1.2, 2GB, Sony UK"
	case "b03112":
		return "4B, Revision 1.2, 2GB, Sony UK"
	case "c03111":
		return "4B, Revision 1.1, 4GB, Sony UK"
	case "c03112":
		return "4B, Revision 1.2, 4GB, Sony UK"
	}
	return "unknown board"
}
