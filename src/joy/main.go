package main

import (
	arm64 "feelings/src/hardware/arm-cortex-a53"
	"feelings/src/joy/semihosting"
	rt "feelings/src/tinygo_runtime"
)

func main() {
	rt.MiniUART = rt.NewUART()
	rt.MiniUART.Configure(rt.UARTConfig{RXInterrupt: true})

	var exceptionLevelName string
	Console.Logf("# Stage 1 kernel is running: Happiness")
	Console.Logf("#   Unix time       %12d %16s: ", rt.BootArg0, exceptionLevelName)
	Console.Logf("#   Boot address    %12x %16s: ", rt.BootArg1, exceptionLevelName)
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
	Console.Logf("#   Exception level %12d %16s: ", rt.BootArg2, exceptionLevelName)

	//interrupts start as off
	arm64.InitInterrupts()
	sprintfGoodCases()
	/*
		bcm2835.InterruptController.EnableTimer()
		bcm2835.InterruptController.EnableIRQs1()
	*/

	//rt.MiniUART.WriteString("hello, world.\n")
	Console.Logf("%%hello%% %-5s,%010d,0x%01x,0x%-06x\n", "1234", 635535, 0x01f, 4096)

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
