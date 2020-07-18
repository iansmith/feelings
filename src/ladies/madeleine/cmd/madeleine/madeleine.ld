/*
 * This is for building one of the kernel ladies:
 * It needs to be loaded into a very different place than the usual spot for
 * bare metal.  It will be loaded into RAM and then jumped to by the
 * bootloader.
 */
ENTRY(start)

SECTIONS
{
	/* this starts ONE PAGE from the zero address so that
	  we can make sure a zero dereference as an address
	  produces a page fault */
	. = 0xfffffc0000010000;
    .exc :  { /* exception vector*/
        KEEP(*(.exc))
    }
	_start = . ;
    .text :  {  /* code can be placed with the exception vector */
        KEEP(*(.text* .text.* .gnu.linkonce.t*));
    }
    .rodata  . : ALIGN(0x10000) {
        *(.rodata .rodata.* .gnu.linkonce.r*)
    }
    .data . : ALIGN(0x10000) {
        _sdata = .;
        *(.data)
        *(.data*)
        . = ALIGN(16);
        _edata = .;
	}
    .bss . (NOLOAD) : ALIGN(0x10000) {
        . = ALIGN(16);
        _sbss = .;
        *(.bss .bss.*)
        *(COMMON)
		. = ALIGN(16);
        _ebss = .;
    }
	_end = .;

   /DISCARD/ : { *(.comment) *(.gnu*) *(.note*) *(.eh_frame*)}

	. = ALIGN(8);
}

/* For the memory allocator. */
_globals_start = _sdata;
_globals_end = _ebss;
/*_stack_top = 0xA0000;*/
_heap_start = _end;
_heap_end = _heap_start + 0x8;
