/*
 * This is for the kernel loading via the bootloader.
 * This alignment on sections is to make it possible to use intel's hex format
 * for sending the binary.
 */

SECTIONS
{
    . = 0x1000;
    .exc  ALIGN(0x1000): {
        KEEP(*(.exc_vector))
    }

    . = 0x100000;
    .text : ALIGN(0x10000) {
                KEEP(*(.text))
                KEEP(*(.text.*))
                KEEP(*(.text.boot))
                *(.gnu.linkonce.t*)
    }

    .rodata ALIGN(0x10000): {
        *(.rodata .rodata.* .gnu.linkonce.r*)
    }

	/* Globals with initial value */
    .data ALIGN(0x10000): {
        *(.data)
        *(.data*)
    }

	/* zero initialized data is initialized by preinit */
    .bss ALIGN(0x10000) (NOLOAD): {
        _sbss = .;
        *(.bss .bss.*)
        *(COMMON)
        _ebss = .;
    }

   /DISCARD/ : { *(.comment) *(.gnu*) *(.note*) *(.eh_frame*)  }
}

_heap_start = 0x400000;
_heap_end = 0x1000000;