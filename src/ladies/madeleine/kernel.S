//
// kernel entry point
// we need a stack to do anything meaningful, so we get the stack as a param to start.
// all the other params go in bootloader_params and x1 is a ptr to that.
//
.global kernel_main

// if we get here, we are running in kernel space with an address that looks
// like 0xffff_fc00_xxxx_xxxx
.global start
start:
	// our bootloader_params is already set up by the bootloader
	adrp x1, bootloader_params
    add x1,x1,#:lo12:bootloader_params

	// initialize VBAR_EL1 with vector table
	adrp x6, VectorTable            // get addr of stack top, top bits
    add x6,x6,#:lo12:VectorTable    // get rest of addr of stack top
    msr vbar_el1,x6                 // store the addr in the place CPU can find it

	//stack pointer is in the structure, but it's the PAGE
	ldr x5,[x1,#24]   // stack BASE (not SP)
	ldr x7,[x1,#64]   // page counts
	and x7,x7,#0xff00 // 2nd byte is stack
	lsr x7,x7,#8      // have isolated the stack pages count
	sub x7,x7,#1      // num pages is 1 too many for this calculation
	lsl x7,x7,#16     // convert to pages
	add x5,x5,x7      // x5 is now the smallest addr of last page
	mov x7,#0xfff
	lsl x7,x7,#4      // x7=0xfff0, which is 16 byte aligned
	add x5,x5,x7      // x5 is absolute top of the stack (highest addr on page)

	mov sp, x5        // set stack pointer, woot!

	adrp x6, _stack_top            // get addr of stack top, top bits
	add x6,x6,#:lo12:_stack_top    // get rest of addr of stack top
	str x5,[x6]                    // store our sp value in there


	// heap start is in the structure
	ldr x5, [x1,#32]               // heap start at 32 bytes in
	adrp x6, _heap_start
	add x6,x6,#:lo12:_heap_start
	str x5,[x6]

	//get the number of pages for heap end
	ldr x7,[x1,#64]                // page counts
	mov x7, #0xff0000              // mask for number of pages

	//conversion implies right shift, then left
    lsr x7,x7,#8                  // have isolated the stack pages count
    sub x7,x7,#1                  // num pages is 1 too many for this calculation
    lsl x7,x7,#16                 // convert to pages
	add x5,x5,x7                  // x5 is now the smallest addr of last page
	mov x7,#0xfff
	lsl x7,x7,#4                  // x7=0xfff0, almost to end of page
	add x5,x5,x7                  // x5 is now last addr of last page of heap

	//store value at heap_end
	adrp x6, _heap_end            //get addr in x6
	add x6,x6,#:lo12:_heap_end
	str x5,[x6]                   //store top addr (from above) in there

	//let's go!
	bl kernel_main
hang:
	b hang   //just in case


// per family heap info
.extern _heap_start
.extern _heap_end

.extern PermitPreemption
schedule_tail:
	str x30, [sp, #-16]!                      //save the link reg so we can get back
    bl    permit_preemption
	ldr x30, [sp], #16                        //save the link reg so we can get back
    //mov x30, x19
    ret

.globl "ladies/madeleine.retFromFork"
"ladies/madeleine.retFromFork":
    bl    schedule_tail
    mov    x0, x20
    blr    x19

// where the bootloader params end up, gotta make sure this is 8 byte aligned
.align 3
.global bootloader_params
bootloader_params:
	.space 256


.global _stack_top
_stack_top:
	.dword 0

.global  tinygo_scanstack
.global  tinygo_scanCurrentStack
.type    tinygo_scanCurrentStack, %function
tinygo_scanCurrentStack:
   // Push callee-saved registers onto the stack.
   	mov	x0, sp
    stp	x29, x30, [sp, #-16]!
    stp	x27, x28, [sp, #-16]!
    stp	x25, x26, [sp, #-16]!
    stp	x23, x24, [sp, #-16]!
    stp	x21, x22, [sp, #-16]!
    stp	x19, x20, [sp, #-16]!

	//x0 already has the SP,just call it
    bl tinygo_scanstack

    // Restore return address.
    ldp	x19, x20, [sp], #16
    ldp	x21, x22, [sp], #16
    ldp	x23, x24, [sp], #16
    ldp	x25, x26, [sp], #16
    ldp	x27, x28, [sp], #16
    ldp	x29, x30, [sp], #16

	// Return to the caller.
    ret
