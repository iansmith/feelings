.balign	4
.globl enable_mmu_tables;
.type enable_mmu_tables, %function

//x0 is first param MAIRVAL
//x1 is 2nd param TCR VAL
//x2 is 3rd param SCTLR VAL
//x3 is 4th param, ttbr0
//x4 is 5th param, ttbr1
enable_mmu_tables:
	dsb sy

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
	ldr x1, x2 //we've already use x1, can destry
	orr x0, x0, x1
	msr sctlr_el1, x0

	ret
