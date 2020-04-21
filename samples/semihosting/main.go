package main

import (
	"feelings/src/lib/semihosting"
	rt "feelings/src/tinygo_runtime"
)

func main() {
	rt.MiniUART = rt.NewUART()
	_ = rt.MiniUART.Configure(rt.UARTConfig{RXInterrupt: true})

	rt.MiniUART.WriteString("hello, world\n")
	x := semihosting.Clock()
	rt.MiniUART.Hex64string(x)
	semihosting.Exit(27)
}
