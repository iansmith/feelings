.section ".exc", "ax", %progbits

//  Set vector table for EL1 for Core0 (All others cores parked)
.globl "ladies/joy.initExceptionVector"
"ladies/joy.initExceptionVector":
	ldr x0, =VectorTable
    msr vbar_el1,x0
	ret

/* macro to align handlers every 0x80 bytes */
.macro vector handler type
.balign 0x80
mov x0,\type
b \handler
.endm

.balign 0x800
.globl VectorTable
VectorTable:
    // from current EL with sp_el0
    vector ex_el1 #0       // Synchronous
    vector ex_el1 #1       // IRQ
    vector ex_el1 #2       // FIQ
    vector ex_el1 #3       // SError

    // from current EL with sp_elx, x != 0
    vector  ex_el1 #4      // Synchronous
    vector  ex_el1 #5      // IRQ
    vector  ex_el1 #6      // FIQ
    vector  ex_el1 #7      // SError

    // from lower EL, target EL minus 1 is AArch64
    vector  ex_el1 #8       // Synchronous
    vector  ex_el1 #9       // IRQ
    vector  ex_el1 #10      // FIQ
    vector  ex_el1 #11      // SError

    // from lower EL, target EL minus 1 is AArch32
    vector  ex_el1 #12      // Synchronous
    vector  ex_el1 #13      // IRQ
    vector  ex_el1 #14      // FIQ
    vector  ex_el1 #15      // SError


.balign 8
.extern raw_exception_handler

//.globl ex_el1
ex_el1:
    stp	x29, x30, [sp, #-16]!
    stp	x27, x28, [sp, #-16]!
    stp	x25, x26, [sp, #-16]!
    stp	x23, x24, [sp, #-16]!
    stp	x21, x22, [sp, #-16]!
    stp	x19, x20, [sp, #-16]!
    stp	x17, x18, [sp, #-16]!
    stp	x15, x16, [sp, #-16]!
    stp	x13, x14, [sp, #-16]!
    stp	x11, x12, [sp, #-16]!
    stp	x9, x10, [sp, #-16]!
    stp	x7, x8, [sp, #-16]!
    stp	x5, x6, [sp, #-16]!
    stp	x3, x4, [sp, #-16]!
    stp	x1, x2, [sp, #-16]!

    mrs	x22, elr_el1
    mrs	x23, spsr_el1

    stp	x0,x22, [sp, #-16]!             //to make eret work right
    str x23, [sp, #-16]!                //to make eret work right

    ldr x5, =raw_exception_handler      // Address to raw exception handler
    mrs x1, esr_el1                     // get syndrome register
    mov x2, x22                         // get link register
    mrs x3, CurrentEL
    lsr x3,x3,#2
    mrs x4, mpidr_el1                   // Fetch core Id
    and x4, x4, #0x3                    // Create 2 bit mask of core Id
    blr x5                              // Call raw exception handler

    mrs	x22, elr_el1                    //to make eret work right
    mrs	x23, spsr_el1                   //to make eret work right
    ldr	x23, [sp], #16
    ldp	x0,x22, [sp], #16

	msr	elr_el1, x22
	msr	spsr_el1, x23

    ldp	x1, x2, [sp], #16
    ldp	x3, x4, [sp], #16
    ldp	x5, x6, [sp], #16
    ldp	x7, x8, [sp], #16
    ldp	x9, x10, [sp], #16
    ldp	x11, x12, [sp], #16
    ldp	x13, x14, [sp], #16
    ldp	x15, x16, [sp], #16
    ldp	x17, x18, [sp], #16
    ldp	x19, x20, [sp], #16
    ldp	x21, x22, [sp], #16
    ldp	x23, x24, [sp], #16
    ldp	x25, x26, [sp], #16
    ldp	x27, x28, [sp], #16
    ldp	x29, x30, [sp], #16
    eret

.align 5
.globl semihosting_call                     // support for ARM v2 semihosting (qemu)
semihosting_call:
    hlt 0xf000
    ret


.align 5
.globl "ladies/joy.cpuSwitchTo"
"ladies/joy.cpuSwitchTo":
	mov	x10, x2                 // offset to RSS
	add	x8, x0, x10
	mov	x9, sp
	stp	x19, x20, [x8], #16		// store callee-saved registers
	stp	x21, x22, [x8], #16
	stp	x23, x24, [x8], #16
	stp	x25, x26, [x8], #16
	stp	x27, x28, [x8], #16
	stp	x29, x9, [x8], #16
	str	x30, [x8]
	add	x8, x1, x10
	ldp	x19, x20, [x8], #16		// restore callee-saved registers
	ldp	x21, x22, [x8], #16
	ldp	x23, x24, [x8], #16
	ldp	x25, x26, [x8], #16
	ldp	x27, x28, [x8], #16
	ldp	x29, x9, [x8], #16
	ldr	x30, [x8]
	mov	sp, x9
	ret

semihostingCharBuffer:
.dword

// special putchar with no allocs
// Call(uint64(Semihostingv2OpWriteC), uintptr(unsafe.Pointer(&c)))
.global semihosting_putchar
semihosting_putchar:
	mov x1, x0
	mov x0,#0x03 //write c
	adrp x2,semihostingCharBuffer
	str x1, [x2]
	mov x1, x2 //get addr of place we put it
	hlt 0xf000
	mov x0, xzr
	ret

