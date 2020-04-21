package main

import (
	"feelings/src/lib/semihosting"
	rt "feelings/src/tinygo_runtime"
)

func main() {
	rt.MiniUART = rt.NewUART()
	rt.MiniUART.Configure(rt.UARTConfig{ /*no interrupt*/ })

	//exit with code 22
	semihosting.SemihostingCall(0x16, 22)
}
