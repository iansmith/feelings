.balign	4

// jump to kernel is used to jump to a kernel loaded by serial... kernel procs
// coming from disk use jump_to_kernel_proc in the loader package
.globl jump_to_kernel
jump_to_kernel:
	//need to jump to kernel, it's entry was passed as first
	mov x19,x0
	mov x0,x1   // pass a pointer to the boot parms SOURCE
	mov x1,x2   // pass a pointer to the boot params DEST

	br x19

.align 3
.global bootloader_params
bootloader_params:
	.space 256

