package main

import (
	"joy"
	"machine"

	"lib/trust"
)

var boot0 uint64
var boot1 uint64
var boot2 uint64

// this function is never called because start calls kernel_main... but
// this has to be here to avoid linker complaints.
func main() {
	trust.Infof("should never be called...")
	joy.KernelMain() //required, or entire kernel gets optimized away
	machine.Abort()
}
