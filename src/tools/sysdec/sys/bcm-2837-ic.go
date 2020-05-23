package sys

import "tools/sysdec"

var IC = &sysdec.PeripheralDef{
	Version: 1,
	Description: `Interrupt Controller: Broadcom implementation of the ARM GIC.

The ARM has two types of interrupt sources:
1. Interrupts coming from the GPU peripherals.
2. Interrupts coming from local ARM control peripherals.

ProTip: To route anything from this interrupt controller to a core, you
need to tell that core that its local routing, either IRQ or FIQ,
should be from the GPU.

The ARM processor gets three types of interrupts:
1. Interrupts from ARM specific peripherals.
2. Interrupts from GPU peripherals.
3. Special events interrupts.

ProTip: Most of the interesting peripherals are attached to this
InterruptController.  The primary reason to use ARM specific peripherals
is access to additional timers (including in QEMU) and to communicate
between cores.`,
	AddressBlock: sysdec.AddressBlockDef{
		BaseAddress: 0xB200,
		Size:        0x28,
	},
	Register: map[string]*sysdec.RegisterDef{
		"Basic": {
			Description: `Basic Pending: The basic pending register shows which 
interrupt are pending. To speed up interrupts processing, a number of 'normal' 
interrupt status bits have been added to this register. This makes the 'IRQ 
pending base' register different from the other 'base' interrupt registers.`,
			AddressOffset: 0x00,
			Access:        sysdec.Access("rw"),
			Size:          32,
			Field: map[string]*sysdec.FieldDef{
				"ARMTimer": {
					Name: "ARM Timer is the per-core timer: " +
						"https://www.raspberrypi.org/documentation/hardware/" +
						"raspberrypi/bcm2836/QA7_rev3.4.pdf",
					BitRange: sysdec.BitRange(0, 0),
				},
			},
		},
		"Pending1": {
			Description: `GPU Pending 1: This register holds ALL interrupts 0..31 
from the GPU side. Some of these interrupts are also connected to the basic 
pending register. Any interrupt status bit in here which is NOT connected to 
the basic pending will also cause bit 8 of the basic pending register to be set. 
That is all bits except 7, 9, 10, 18, 19.`,
			Size:          32,
			AddressOffset: 0x04,
			Field: map[string]*sysdec.FieldDef{
				"Aux": {
					Description: "One of the three auxiliary devices has a " +
						"pending interrupt",
					BitRange: sysdec.BitRange(29, 29),
					Access:   sysdec.Access("r"),
				},
			},
		},
		"Enable1": {
			Description: `Enable interrupts from Group 1 (0-31): 
Writing a 1 to a bit will set the corresponding IRQ 
enable bit. All other IRQ enable bits are unaffected. Only bits which are 
enabled can be seen in the interrupt pending registers. There is no provision 
here to see if there are interrupts which are pending but not enabled.`,
			Size:          32,
			AddressOffset: 0x10,
			Field: map[string]*sysdec.FieldDef{
				"Aux": {
					Name:     "Enable the interrupt from the three auxiliary devices",
					BitRange: sysdec.BitRange(29, 29),
					Access:   sysdec.Access("w"),
				},
				"TestWidth": {
					Description: "Test with 2 bit values",
					BitRange:    sysdec.BitRange(27, 26),
					Access:      sysdec.Access("w"),
					EnumeratedValue: map[string]*sysdec.EnumeratedValueDef{
						"Core0": {Description: "Route To Core 0", Value: 0},
						"Core1": {Description: "Route To Core 1", Value: 1},
						"Core2": {Description: "Route To Core 2", Value: 2},
						"Core3": {Description: "Route To Core 2", Value: 3},
					},
				},
				"TestLarger": {
					Description: "Test with 8 bit values",
					BitRange:    sysdec.BitRange(7, 0),
					Access:      sysdec.Access("r"),
				},
				"TestReadWrite": {
					Description: "Test with 3 bit values",
					BitRange:    sysdec.BitRange(10, 8),
					Access:      sysdec.Access("rw"),
					EnumeratedValue: map[string]*sysdec.EnumeratedValueDef{
						"Bit0of3": {Value: 1},
						"Bit1of3": {Value: 2},
						"Bit2of3": {Value: 4},
					},
				},
			},
		},
		"ArrayTestPerCore[%s]": {
			Description:   `test that we handle arrays right`,
			Size:          32,
			AddressOffset: 0x14,
			Dim:           4,
			DimIncrement:  4, //32bits
			DimIndices: map[string]int{
				"ATestFoo": 0,
				"BTestFoo": 1,
				"TestBaz":  2,
				"Fleazil":  3,
			},
		},
	},
}
