/*
 * This is for building the kernel so it needs to be loaded into a very different
 * place than the usual spot for bare metal.
 */
ENTRY(start)

SECTIONS
{
	. = 0xfffffc0030000000;
    .exc : { KEEP(*(.exc)) }
	_start = . ;
    .text : { KEEP(*(.text* .text.* .gnu.linkonce.t*)) }
    .rodata : { *(.rodata .rodata.* .gnu.linkonce.r*) }
    .data :
    {
        . = ALIGN(16);
        _sdata = .;
        *(.data)
        *(.data*)
        . = ALIGN(16);
        _edata = .;
    }
    .bss (NOLOAD) : {
        . = ALIGN(16);
        _sbss = .;
        *(.bss .bss.*)
        *(COMMON)
		. = ALIGN(16);
        _ebss = .;
    }
	_end = .;

   /DISCARD/ : { *(.comment) *(.gnu*) *(.note*) *(.eh_frame*)}
}

/* For the memory allocator. */
_globals_start = _sdata;
_globals_end = _ebss;
_stack_top = 0xA0000;
_heap_start = _end;
_heap_end = _heap_start + 0x200000; /* 2Mb*/
