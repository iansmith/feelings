package main

import (
	arm64 "feelings/src/hardware/arm-cortex-a53"
	"feelings/src/joy/semihosting"
	rt "feelings/src/tinygo_runtime"

	"github.com/tinygo-org/tinygo/src/runtime"
)

func main() {
	runtime.SetExternalRuntime(&rt.BaremetalRT{})
	rt.MiniUART = rt.NewUART()
	rt.MiniUART.Configure(rt.UARTConfig{RXInterrupt: true})

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
	semihosting.Exit(37)
}
