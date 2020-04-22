/*
 * This file assumes that you are running with the Raspberry PI 3/4 firmware
 * that expects bare metal to be linked for a boot address of 0x80000.
 *
 * You may want to consult https://www.raspberrypi.org/documentation/configuration/config-txt/boot.md
 * if you want to see how to force this boot address in your config.txt.
 */

SECTIONS
{
    . = 0x80000;
    anticipation_addr = .;
    .text : ALIGN(0x1000) {
                KEEP(*(.text.boot))
                KEEP(*(.text))
                KEEP(*(.text.*))
                *(.gnu.linkonce.t*)
    }

    .rodata ALIGN(0x10): {
        *(.rodata .rodata.* .gnu.linkonce.r*)
    }

	/* Globals with initial value */
    .data ALIGN(0x10): {
        *(.data)
        *(.data*)
    }

	/* zero initialized data is initialized by preinit */
    .bss ALIGN(0x1000) (NOLOAD): {
        _sbss = .;
        *(.bss .bss.*)
        *(COMMON)
        . = ALIGN(0x10);
        _ebss = .;
    }

    . = 0x96000;
   .exc  ALIGN(0x1000): {
        KEEP(*(.exc_vector))
     }
   /DISCARD/ : { *(.comment) *(.gnu*) *(.note*) *(.eh_frame*)  }
}

_heap_start = 0x1f000000;
_heap_end = 0x40000000;