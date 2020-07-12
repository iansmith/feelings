package madeleine

import "lib/trust"

//go:external init_exception_vector
func initExceptionVector()

// true entry point
//go:export kernel_main
func KernelMain() {
	initExceptionVector()

	trust.Debugf("kernelMain1")

}

//go:export permit_preemption
func permitPreemption() {
	panic("permitPreemption")
}
