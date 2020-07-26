package madeleine

import "lib/trust"

// true entry point
//go:noinline
//go:export kernel_main
func KernelMain() {
	//we have already initialized the kernel exception vector in start()

	trust.Debugf("kernelMain1")

}

//go:export permit_preemption
func permitPreemption() {
	panic("permitPreemption")
}
