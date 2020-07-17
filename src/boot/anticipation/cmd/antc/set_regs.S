.balign	4

// xxx shared with the other (boot) binary and we COPY this value there, ugh
.equ spin_locations_base, 0xd8              // offset to make math work

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

.global "boot/anticipation/cmd/antc.wait10000"
"boot/anticipation/cmd/antc.wait10000":
	mov x0,#10000
iter:
	sub x0,x0,#1
	cbnz x0, iter
	ret

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
	add x5, x5, x0                 // x5 points to the first slot for that core now
wait_for_clear:
	ldr x6, [x5]
	cbnz x6,wait_for_clear
    str x2, [x5]                   // store entry point for proper core

    //now we copy the boot parameters from x3 to x4 but using the physical address
    //because we are on the wrong ttbr1 value and we are on core 0.. x3 is the bootloader
    //addr space, x4 is in the kernel proc but this is ok because we are el1 and
    //we are using the physical addrs (or really identity mapped virtual addrs on core 0)
    mov x0, x3  //prep
    mov x1, x4  //prep

    //we do the copy with a moving x0 and a fixed x1
    ldr x2, [x0, #0]
    str x2,[x1,#0]

    ldr x2, [x0, #8]
    str x2,[x1,#8]

    ldr x2, [x0, #16]
    str x2,[x1,#16]

    ldr x2,[x0,#24]
    str x2,[x1,#24]

    ldr x2,[x0,#32]
    str x2,[x1,#32]

    ldr x2,[x0,#40]
    str x2,[x1,#40]

    ldr x2,[x0,#40]
    str x2,[x1,#48]

    ldr x2,[x0,#56]
    str x2,[x1,#56]

    ldr x2,[x0, #64]
    str x2,[x1,#64]

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