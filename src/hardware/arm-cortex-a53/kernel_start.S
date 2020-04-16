.section ".exc_vector","ax",@progbits
.align 7

.extern raw_exception_handler

/*
 * Interrupt vectors that are jumped to by the hardware when we get
 * an exception. There are 4 states and 4 types of exceptions so
 * we have 0x10 entries, and each entry cannot exceed 0x80
 */
#define S_FRAME_SIZE			256 		// size of all saved registers

#define SYNC_EL1t		0
#define IRQ_EL1t		1
#define FIQ_EL1t		2
#define ERROR_EL1t		3

#define SYNC_EL1h		4
#define IRQ_EL1h		5
#define FIQ_EL1h		6
#define ERROR_EL1h		7

#define SYNC_EL0_64	    	8
#define IRQ_EL0_64	    	9
#define FIQ_EL0_64		10
#define ERROR_EL0_64		11

#define SYNC_EL0_32		12
#define IRQ_EL0_32		13
#define FIQ_EL0_32		14
#define ERROR_EL0_32		15

	.macro interrupt_entry type
	kernel_entry
	mov	x0, #\type
	mrs	x1, esr_el1
	mrs	x2, elr_el1
	bl raw_exception_handler
	kernel_exit
	.endm

	.macro	ventry	label
	.align	7
	b	\label
	.endm

	.macro	kernel_entry
	sub	sp, sp, #S_FRAME_SIZE
	stp	x0, x1, [sp, #16 * 0]
	stp	x2, x3, [sp, #16 * 1]
	stp	x4, x5, [sp, #16 * 2]
	stp	x6, x7, [sp, #16 * 3]
	stp	x8, x9, [sp, #16 * 4]
	stp	x10, x11, [sp, #16 * 5]
	stp	x12, x13, [sp, #16 * 6]
	stp	x14, x15, [sp, #16 * 7]
	stp	x16, x17, [sp, #16 * 8]
	stp	x18, x19, [sp, #16 * 9]
	stp	x20, x21, [sp, #16 * 10]
	stp	x22, x23, [sp, #16 * 11]
	stp	x24, x25, [sp, #16 * 12]
	stp	x26, x27, [sp, #16 * 13]
	stp	x28, x29, [sp, #16 * 14]
	str	x30, [sp, #16 * 15]
	.endm

	.macro	kernel_exit
	ldp	x0, x1, [sp, #16 * 0]
	ldp	x2, x3, [sp, #16 * 1]
	ldp	x4, x5, [sp, #16 * 2]
	ldp	x6, x7, [sp, #16 * 3]
	ldp	x8, x9, [sp, #16 * 4]
	ldp	x10, x11, [sp, #16 * 5]
	ldp	x12, x13, [sp, #16 * 6]
	ldp	x14, x15, [sp, #16 * 7]
	ldp	x16, x17, [sp, #16 * 8]
	ldp	x18, x19, [sp, #16 * 9]
	ldp	x20, x21, [sp, #16 * 10]
	ldp	x22, x23, [sp, #16 * 11]
	ldp	x24, x25, [sp, #16 * 12]
	ldp	x26, x27, [sp, #16 * 13]
	ldp	x28, x29, [sp, #16 * 14]
	ldr	x30, [sp, #16 * 15]
	add	sp, sp, #S_FRAME_SIZE
	eret
	.endm

.align 11
.globl vectors
vectors:
	ventry	sync_el1t			// Synchronous EL1t
	ventry	irq_el1t			// IRQ EL1t
	ventry	fiq_el1t			// FIQ EL1t
	ventry	error_el1t			// Error EL1t

	ventry	sync_el1h			// Synchronous EL1h
	ventry	irq_el1h		    // IRQ EL1h -- big dog
	ventry	fiq_el1h			// FIQ EL1h
	ventry	error_el1h			// Error EL1h

	ventry	sync_el0_64			// Synchronous 64-bit EL0
	ventry	irq_el0_64			// IRQ 64-bit EL0
	ventry	fiq_el0_64			// FIQ 64-bit EL0
	ventry	error_el0_64		// Error 64-bit EL0

	ventry	sync_el0_32			// Synchronous 32-bit EL0
	ventry	irq_el0_32			// IRQ 32-bit EL0
	ventry	fiq_el0_32			// FIQ 32-bit EL0
	ventry	error_el0_32		// Error 32-bit EL0


// these functions need to be broken out so each macro instation can jump to a different point.
.globl sync_el1t
sync_el1t:
	interrupt_entry  SYNC_EL1t

.globl irq_el1t
irq_el1t:
	interrupt_entry  IRQ_EL1t

.globl fiq_el1t
fiq_el1t:
	interrupt_entry  FIQ_EL1t

.globl error_el1t
error_el1t:
	interrupt_entry  ERROR_EL1t

.globl sync_el1h
sync_el1h:
	interrupt_entry  SYNC_EL1h

.globl irq_el1h
irq_el1h:
	interrupt_entry IRQ_EL1h

.globl fiq_el1h
fiq_el1h:
	interrupt_entry  FIQ_EL1h

.globl error_el1h
error_el1h:
	interrupt_entry  ERROR_EL1h

.globl sync_el0_64
sync_el0_64:
	interrupt_entry  SYNC_EL0_64

.globl irq_el0_64
irq_el0_64:
	interrupt_entry  IRQ_EL0_64

.globl fiq_el0_64
fiq_el0_64:
	interrupt_entry  FIQ_EL0_64

.globl error_el0_64
error_el0_64:
	interrupt_entry  ERROR_EL0_64

.globl sync_el0_32
sync_el0_32:
	interrupt_entry  SYNC_EL0_32

.globl irq_el0_32
irq_el0_32:
	interrupt_entry  IRQ_EL0_32

.globl fiq_el0_32
fiq_el0_32:
	interrupt_entry  FIQ_EL0_32

.globl error_el0_32
error_el0_32:
	interrupt_entry  ERROR_EL0_32


.section ".text.boot"
.globl _start
_start:
	// retset stack before our code (we are in el1)
	// the stack will end up overwriting the bootloader when we reach 0x80000
	mov     x1, #0x100000
	mov     sp,x1
	mov     x30,xzr

    //jump to main
	adr x8, main
	blr x8

	adr x8, err_hang
	br x8

.globl semihosting_param_block
semihosting_param_block:
.space 16

.align(11)
err_hang: b err_hang

.globl getSystemStackPointer
getSystemStackPointer:
	mov sp,x0
