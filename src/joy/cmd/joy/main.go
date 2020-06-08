package main

import (
	"machine"

	"joy"
	"lib/trust"
)

var boot0 uint64
var boot1 uint64
var boot2 uint64

func main() {
	trust.Infof("control transferred from bootloader to kernel...")
	joy.KernelMain()
	trust.Fatalf(1, "kernel returned from KernelMain()")
	machine.Abort()
}
