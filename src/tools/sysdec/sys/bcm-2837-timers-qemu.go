package sys

import "tools/sysdec"

var SystemTimerQEMU = &sysdec.PeripheralDef{
	Version: 1,
	Description: `
A free running 64 bit timer and 2 (neé 4) match registers.  However, the QEMU
implementation does not have the 2 match registers.

The exact behavior of this "clock" is hard to understand in QEMU.

This is sometimes called the Chapter 12 timer, referring the BCM2835
ARM peripherals manual and to disambiguate from the Chapter 14 timer
and the ARM local timer.`,
	AddressBlock: sysdec.AddressBlockDef{BaseAddress: 0x3000, Size: 0x1C},
	Register: map[string]*sysdec.RegisterDef{
		"SystemTimerLower32": {
			Description:   `System Timer counter Lower 32 bits`,
			AddressOffset: 0x4,
			Size:          32,
		},
		"SystemTimerUpper32": {
			Description:   `System Timer counter Upper 32 bits`,
			AddressOffset: 0x8,
			Size:          32,
		},
	},
}
