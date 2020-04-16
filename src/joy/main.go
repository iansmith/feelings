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

	rt.MiniUART.WriteString("hello, world.\n")
	semihosting.Exit(37)
}
