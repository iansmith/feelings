.balign 4

// xxx shqared with the other (boot) binary and we COPY this value there, ugh
.equ spin_locations_base, 0xd8              // offset to make math work

.globl _enable_mmu_tables;
.type _enable_mmu_tables, %function

.globl jump_to_kernel_proc
jump_to_kernel_proc:
	//this toggles the switch that frees a core and sends that core to
	//entry point where it can complete the setup and start running
	//
	// x0: which core
	// x1: ttbr1 value to insert
	// x2: location that starts the kernel proc
	// x3: addr of the boot params in bootloader (source)
	// x4: addr of the boot params in the kernel proc (dest)

	mov x5, #spin_locations_base   // this is a bogus address to make the math work out
	lsl x0,x0,#3                   // mult by 8
	add x5, x5, x0                 // x5 points to the entry point slot for that core now
	                               // we hold it x5 while we copy the boot params

    //now we copy the boot parameters from x3 to x4 but using the physical address
    //because we are on the wrong ttbr1 value and we are on core 0.. x3 is the bootloader
    //addr space, x4 is in the kernel proc but this is ok because we are el1 and
    //we are using the physical addrs (or really identity mapped virtual addrs on core 0)

    //we do the copy with and indexed addressing mode
    mov x7, #72  // one past the end, because we subtract immediately
copy_param:
	sub x7,x7,#8
	ldr x0, [x3, x7]
	str x0, [x4, x7]
	cbnz x7,copy_param

	// want to store the correct addr (x2) into the location pointed to by x5
	// so the other core will see it and jump to x2
wait_for_clear:
	ldr x6, [x5]
	cbnz x6,wait_for_clear
    str x2, [x5]                   // store entry point for proper core

	//make sure all cores is aware
    dsb sy
	//ok, hit it
    sev                           // wakes other core from park
	ret

// this takes the params provided by go, hides the args so the other core can find
// them, and then asks it to run enable_mmu_tables above.
// in addition to all the parameters of the above, we also take the core number in x5
.globl _enable_mmu_tables_other_core
_enable_mmu_tables_other_core:
	mov x7, #0x80000                   // this is the load addr of the boot loader
	                                   // we are going to hide all these params _under_ this addr
	str x0, [x7,#-8]
	str x1, [x7,#-16]
	str x2, [x7,#-24]
	str x3, [x7,#-32]
	str x4, [x7,#-40]

    mov x6, #spin_locations_base            // Load address of spins
	ldr x4, [x6, x5, lsl #3]                // Fetch address that has been set (mpy core # by 8)
	cbnz x4,error_exit
	adr x4, _enable_mmu_tables_preamble
	str x4, [x6, x5, lsl #3]                // set the other core's target
	dsb sy
	sev                                     // send the signal to wake him up
	mov x0,#0
	ret
error_exit:
	mov x0,#1
	ret


// this retreives the params hidden by the bootloader and then calls the
// real function, enable_mmu_tables. runs on diff core than enable_mmu_tables_other_core
.globl _enable_mmu_tables_preamble
_enable_mmu_tables_preamble:
	mov x7, #0x80000                   // this is the load addr of the boot loader
                                       // we hid the values under the bootloader
    ldr x0, [x7,#-8]
    ldr x1, [x7,#-16]
    ldr x2, [x7,#-24]
    ldr x3, [x7,#-32]
    ldr x4, [x7,#-40]
    // we don't save x30 on the stack, so this must be B not BL
	b _enable_mmu_tables
	ret

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

