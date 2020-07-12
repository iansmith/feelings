//
// kernel entry point
// we need a stack to do anything meaningful, so we get the stack as a param to start.
// all the other params go in bootloader_params and x1 is a ptr to that.
//
.global kernel_main

.global start
start:
	//stack pointer is in the structure, but put into sp and store in _stack_top
	ldr x5,[x0,#24]
	mov sp, x5
	adrp x6, _stack_top
	add x6,x6,#:lo12:_stack_top
	str x5,[x6]


	// heap start is in the struction
	adrp x6, _heap_start
	add x6,x6,#:lo12:_heap_start
	ldr x5,[x0,#32]
	str x5,[x6]

	//heap end is in the structure
	adrp x6, _heap_end
	add x6,x6,#:lo12:_heap_end
	ldr x5,[x0,#40]
	str x5,[x6]

	b kernel_main


.globl "ladies/madeleine.setHeapPointers"

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

