package main

import (
	"fmt"

	"device/arm"

	"happiness"
	"lib/upbeat"
	rt "runtime"
	"strings"
)

var boot0 uint64
var boot1 uint64
var boot2 uint64

func main() {

	var exceptionLevelName string
	fmt.Printf("# Stage 1 kernel is running: Happiness\n")
	fmt.Printf("# %16s : %d ", "Unix time", boot0)
	fmt.Printf("# %16s : 0x%08x ", "Boot Address", boot1)
	switch boot2 {
	case 0:
		exceptionLevelName = "(User)"
	case 1:
		exceptionLevelName = "(Kernel)"
	case 2:
		exceptionLevelName = "(Hypervisor)"
	case 3:
		exceptionLevelName = "(Secure Kernel)"
	default:
		exceptionLevelName = "(wtf?)"
	}
	happiness.Console.Logf("# %16s : %d %s ", "Exception level", boot2,
		exceptionLevelName)
	bid, ok := upbeat.BoardID()
	if !ok {
		happiness.Console.Logf("!   Failed To read Board ID!")
	} else {
		happiness.Console.Logf("# %16s : %08x%08x ", "Board Id",
			bid>>32, bid&(0xffffffff<<32))
	}
	//for i := 0; i < 100000000; i++ {
	//	arm.Asm("nop")
	//}
	firmware, ok := upbeat.FirmwareVersion()
	if !ok {
		happiness.Console.Logf("!   Failed To read Firmware Version!")
	} else {
		happiness.Console.Logf("# %16s : %d ", "Firmware version", firmware)
	}
	boardModel, ok := upbeat.BoardModel()
	if !ok {
		happiness.Console.Logf("!   Failed To read Board Model!")
	} else {
		happiness.Console.Logf("# %16s : %d ", "Board Model", boardModel)
	}
	boardRevision, ok := upbeat.BoardRevision()
	if !ok {
		happiness.Console.Logf("!   Failed To read Board Revision!")
	} else {
		br := strings.ToLower(happiness.Console.Sprintf("%x", boardRevision))
		happiness.Console.Logf("# %16s : %s (%s)", "Board Revision", br,
			upbeat.BoardRevisionDecode(br))
	}
	addr, ok := upbeat.MACAddress()
	printableMAC := happiness.Console.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
		addr>>0x30&0xff,
		addr>>0x28&0xff, addr>>0x20&0xff, addr>>0x18&0xff,
		addr>>0x10&0xff, addr>>8&0xff, addr&0xff)

	if !ok {
		happiness.Console.Logf("!   Failed to read the MAC Address!")
	} else {
		happiness.Console.Logf("# %16s : %s ", "MAC Address", printableMAC)
	}
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
	rt.Exit()
}

//export raw_exception_handler
func rawExceptionHandler(t uint64, esr uint64, addr uint64, el uint64, procId uint64) {
	if t != 5 {
		// this is in case we get some OTHER kind of exception
		fmt.Printf("raw exception handler:exception type %d and "+
			"esr %x with addr %x and EL=%d, ProcID=%x\n",
			t, esr, addr, el, procId)
	} else {
		fmt.Printf("should not be receiving interrupts!\n")
	}
	fmt.Printf("DEADLOOP\n")
	for {
		arm.Asm("nop")
	}

}
