.balign	4
.globl _enable_mmu_tables;
.type _enable_mmu_tables, %function

//x0 is first param MAIRVAL
//x1 is 2nd param TCR VAL
//x2 is 3rd param SCTLR VAL
//x3 is 4th param, ttbr0
//x4 is 5th param, ttbr1
_enable_mmu_tables:

	//Set the memattrs values into mair_el1
    msr mair_el1, x0

	// Bring both tables online and execute memory barrier
	msr ttbr0_el1, x3
	msr ttbr1_el1, x4
	isb

    //set a zillion translation params
	msr tcr_el1, x1
	isb

	mrs x0, sctlr_el1
	mov x1, x2 //we've already use x1, can destry
	orr x0, x0, x2
	msr sctlr_el1, x0

	isb
	ret

//tricky: this has to be here so the linker will not complain when you compile
//tricky: the antc command, but it will never be used because in fact the definition
//tricky: is going to in the *kernel* address space
.globl bootloader_params
bootloader_params:
.space 256

kernel_boot_params:
.dword 0xfffffc0030001100

kernel_heap_start:
.dword 0xfffffc003000de10

kernel_heap_end:
.dword 0xfffffc003000de18

.globl jump_to_kernel
jump_to_kernel:
	ldr x19,kernel_boot_params
	str x0, [x19]
	add x19,x19,8
	str x1, [x19]
	add x19,x19,8
	str x2, [x19]
	add x19,x19,8
	str x3, [x19]

	//need to jump to kernel, it's entry was passed as first
	mov x19,x0
	//adjust stack to point to right place
	mov sp,x3
	br x19

