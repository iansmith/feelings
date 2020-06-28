package sys

import "tools/sysdec"

var SystemTimer = &sysdec.PeripheralDef{
	Version: 1,
	Description: `
A free running 64 bit timer and 2 (neé 4) match registers.  Only two
of these registers are actually available, so only those two have been
documented.

This is sometimes called the Chapter 12 timer, referring the BCM2835
ARM peripherals manual and to disambiguate from the Chapter 14 timer
and the ARM local timer.

The System Timer peripheral provides four 32-bit timer channels and a 
single 64-bit free running counter. Each channel has an output compare 
register, which is compared against the 32 least significant bits of the 
free running counter values. When the two values match, the system timer 
peripheral generates a signal to indicate a match for the appropriate channel. 
The match signal is then fed into the interrupt controller. The interrupt 
service routine then reads the output compare register and adds the appropriate 
offset for the next timer tick. The free running counter is driven by the 
timer clock and stopped whenever the processor is stopped in debug mode.`,
	AddressBlock: sysdec.AddressBlockDef{BaseAddress: 0x3000, Size: 0x1C},
	Register: map[string]*sysdec.RegisterDef{
		"CS": {
			Description: `System Timer Control/Status.  This register is 
used to record and clear timer channel comparator matches. The system 
timer match bits are routed to the interrupt controller where they can 
generate an interrupt. The M0-3 fields contain the free-running counter 
match status. Write a one to the relevant bit to clear the match detect 
status bit and the corresponding interrupt request line.`,
			AddressOffset: 0x0,
			Size:          4,
			Field: map[string]*sysdec.FieldDef{
				"Match3": {
					Description: `System Timer Match 3
0 = No Timer 3 match since last cleared. 
1 = Timer 3 match detected.`,
					BitRange: sysdec.BitRange(3, 3),
					Access:   sysdec.Access("rw"),
				},
				"Match1": {
					Description: `System Timer Match 1
0 = No Timer 1 match since last cleared. 
1 = Timer 1 match detected.`,
					BitRange: sysdec.BitRange(1, 1),
					Access:   sysdec.Access("rw"),
				},
			},
		},
		"LeastSignificant32": {
			Description: `System Timer counter Lower 32 bits.The system 
timer free-running counter lower register is a read-only register that 
returns the current value of the lower 32-bits of the free running counter.`,
			AddressOffset: 0x4,
			Size:          32,
			Access:        sysdec.Access("r"),
		},
		"MostSignificant32": {
			Description: `System Timer counter Upper 32 bits.The system 
timer free-running counter higher register is a read-only register that 
returns the current value of the higher 32-bits of the free running counter.`,
			AddressOffset: 0x8,
			Size:          32,
			Access:        sysdec.Access("r"),
		},
		"Compare1": {
			Description: `The system timer compare registers hold 
the compare value for each of the four timer channels. Whenever the 
lower 32-bits of the free-running counter matches one of the compare 
values the corresponding bit in the system timer control/status register 
is set.  Each timer peripheral (minirun and run) has a set of 
four compare registers.  Note: Only two of these timers are actually
usable`,
			AddressOffset: 0x10,
			Size:          32,
			Access:        sysdec.Access("rw"),
		},
		"Compare3": {
			Description: `The system timer compare registers hold 
the compare value for each of the four timer channels. Whenever the 
lower 32-bits of the free-running counter matches one of the compare 
values the corresponding bit in the system timer control/status register 
is set.  Each timer peripheral (minirun and run) has a set of 
four compare registers.  Note: Only two of these timers are actually
usable`,
			AddressOffset: 0x18,
			Size:          32,
			Access:        sysdec.Access("rw"),
		},
	},
}
