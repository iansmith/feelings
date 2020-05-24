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
		"BasicPending": {
			Description: `Basic Pending: The basic pending register shows which 
interrupt are pending. To speed up interrupts processing, a number of 'normal' 
interrupt status bits have been added to this register. This makes the 'IRQ 
pending base' register different from the other 'base' interrupt registers.`,
			AddressOffset: 0x00,
			Access:        sysdec.Access("r"),
			Size:          21,
			Field: map[string]*sysdec.FieldDef{
				"ARMTimer": {
					Description: `ARM Timer is the per-core timer.
https://www.raspberrypi.org/documentation/hardware/raspberrypi/bcm2836/QA7_rev3.4.pdf
Note that this is sometimes called the "local" timer to distinguish from the
chapter 14 and chapter 12 timers.`,
					BitRange: sysdec.BitRange(0, 0),
				},
				"ARMMailbox": {
					Description: ``,
					BitRange:    sysdec.BitRange(1, 1),
				},
				"ARMDoorbell0": {
					Description: ``,
					BitRange:    sysdec.BitRange(2, 2),
				},
				"ARMDoorbell1": {
					Description: ``,
					BitRange:    sysdec.BitRange(3, 3),
				},
				"GPU0Halted": {
					Description: `(Or GPU1 halted if bit 10 of control 
register 1 is set)`,
					BitRange: sysdec.BitRange(4, 4),
				},
				"GPU1Halted": {
					Description: ``,
					BitRange:    sysdec.BitRange(5, 5),
				},
				"IllegalAccessType1": {
					Description: ``,
					BitRange:    sysdec.BitRange(6, 6),
				},
				"IllegalAccessType0": {
					Description: ``,
					BitRange:    sysdec.BitRange(7, 7),
				},
				"MoreBitsSetInPending1": {
					Description: ``,
					BitRange:    sysdec.BitRange(8, 8),
				},
				"MoreBitsSetInPending2": {
					Description: ``,
					BitRange:    sysdec.BitRange(9, 9),
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
			Access:        sysdec.Access("r"),
			Field: map[string]*sysdec.FieldDef{
				"Aux": {
					Description: `One of the three auxiliary devices has a 
pending interrupt.`,
					BitRange: sysdec.BitRange(29, 29),
				},
			},
		},
		"Pending2": {
			Description: `GPU Pending 2: This register holds ALL interrupts 
32..63 from the GPU side. Some of these interrupts are also connected to the 
basic pending register. Any interrupt status bit in here which is NOT 
connected to the basic pending will also cause bit 9 of the basic pending 
register to be set. That is all bits except . register bits 21..25, 30 
(Interrupts 53..57,62). `,
			Size:          32,
			AddressOffset: 0x08,
			Access:        sysdec.Access("r"),
			Field: map[string]*sysdec.FieldDef{
				"GPIO0": {
					Description: ``,
					BitRange:    sysdec.BitRange(17, 17),
				},
				"GPIO1": {
					Description: ``,
					BitRange:    sysdec.BitRange(18, 18),
				},
				"GPIO2": {
					Description: ``,
					BitRange:    sysdec.BitRange(19, 19),
				},
				"GPIO3": {
					Description: ``,
					BitRange:    sysdec.BitRange(20, 20),
				},
				"I2C": {
					Description: ``,
					BitRange:    sysdec.BitRange(21, 21),
				},
				"SPI": {
					Description: ``,
					BitRange:    sysdec.BitRange(22, 22),
				},
				"PCM": {
					Description: ``,
					BitRange:    sysdec.BitRange(22, 22),
				},
				"UART": {
					Description: `This is not the mini UART, it's UART0.'`,
					BitRange:    sysdec.BitRange(24, 24),
				},
			},
		},
		"FIQSource": {
			Description: `The FIQ register control which interrupt source 
can generate a FIQ to the ARM. Only a single interrupt can be selected.`,
			Size:          8,
			AddressOffset: 0x0C,
			Field: map[string]*sysdec.FieldDef{
				"FIQEnable": {
					Description: `FIQ enable. Set this bit to 1 to enable 
FIQ generation. If set to 0 bits 6:0 are don't care.`,
					BitRange: sysdec.BitRange(7, 7),
					Access:   sysdec.Access("rw"),
				},
				"FIQSource": {
					Description: `FIQ enable. Set this bit to 1 to enable 
FIQ generation. If set to 0 bits 6:0 are don't care.  First 64 values
correspond to the normal GPU interrupts.`,
					BitRange: sysdec.BitRange(6, 0),
					Access:   sysdec.Access("rw"),
					EnumeratedValue: map[string]*sysdec.EnumeratedValueDef{
						"ARMTimer":           {Value: 64},
						"ARMMailbox":         {Value: 65},
						"ARMDoorbell0":       {Value: 66},
						"ARMDoorbell1":       {Value: 67},
						"GPU0Halted":         {Value: 68},
						"GPU1Halted":         {Value: 69},
						"IllegalAccessType1": {Value: 70},
						"IllegalAccessType0": {Value: 71},
					},
				},
			},
		},
		"Enable1": {
			Description: `Enable GPU interrupts from Group 1 (0-31): 
Writing a 1 to a bit will set the corresponding IRQ 
enable bit. All other IRQ enable bits are unaffected. Only bits which are 
enabled can be seen in the interrupt pending registers. There is no provision 
here to see if there are interrupts which are pending but not enabled.`,
			Size:          32,
			AddressOffset: 0x10,
			Access:        sysdec.Access("w"),
			Field: map[string]*sysdec.FieldDef{
				"Aux": {
					Description: "Enable the interrupt from the three auxiliary devices",
					BitRange:    sysdec.BitRange(29, 29),
					Access:      sysdec.Access("w"),
				},
			},
		},
		"Enable2": {
			Description: `Enable GPU interrupts from Group 2 (32-63): 
Writing a 1 to a bit will set the corresponding IRQ 
enable bit. All other IRQ enable bits are unaffected. Only bits which are 
enabled can be seen in the interrupt pending registers. There is no provision 
here to see if there are interrupts which are pending but not enabled.`,
			Size:          32,
			AddressOffset: 0x14,
			Access:        sysdec.Access("w"),
			Field: map[string]*sysdec.FieldDef{
				"GPIO0": {
					Description: ``,
					BitRange:    sysdec.BitRange(17, 17),
					Access:      sysdec.Access("r"),
				},
				"GPIO1": {
					Description: ``,
					BitRange:    sysdec.BitRange(18, 18),
					Access:      sysdec.Access("r"),
				},
				"GPIO2": {
					Description: ``,
					BitRange:    sysdec.BitRange(19, 19),
					Access:      sysdec.Access("r"),
				},
				"GPIO3": {
					Description: ``,
					BitRange:    sysdec.BitRange(20, 20),
					Access:      sysdec.Access("r"),
				},
				"I2C": {
					Description: ``,
					BitRange:    sysdec.BitRange(21, 21),
					Access:      sysdec.Access("r"),
				},
				"SPI": {
					Description: ``,
					BitRange:    sysdec.BitRange(22, 22),
					Access:      sysdec.Access("r"),
				},
				"PCM": {
					Description: ``,
					BitRange:    sysdec.BitRange(22, 22),
					Access:      sysdec.Access("r"),
				},
				"UART": {
					Description: `This is not the mini UART, it's UART0.'`,
					BitRange:    sysdec.BitRange(24, 24),
					Access:      sysdec.Access("r"),
				},
			},
		},
		"EnableBasic": {
			Description: `Writing a 1 to a bit will set the corresponding 
IRQ enable bit. All other IRQ enable bits are unaffected. Again only bits 
which are enabled can be seen in the basic pending register. There is no 
provision here to see if there are interrupts which are pending but not 
enabled.`,
			Size:          8,
			AddressOffset: 0x18,
			Access:        sysdec.Access("w"),
			Field: map[string]*sysdec.FieldDef{
				"ARMTimer": {
					Description: `ARM Timer is the per-core timer.
https://www.raspberrypi.org/documentation/hardware/raspberrypi/bcm2836/QA7_rev3.4.pdf
Note that this is sometimes called the "local" timer to distinguish from the
chapter 14 and chapter 12 timers.`,
					BitRange: sysdec.BitRange(0, 0),
				},
				"ARMMailbox": {
					Description: ``,
					BitRange:    sysdec.BitRange(1, 1),
				},
				"ARMDoorbell0": {
					Description: ``,
					BitRange:    sysdec.BitRange(2, 2),
				},
				"ARMDoorbell1": {
					Description: ``,
					BitRange:    sysdec.BitRange(3, 3),
				},
				"GPU0Halted": {
					Description: `(Or GPU1 halted if bit 10 of control 
register 1 is set)`,
					BitRange: sysdec.BitRange(4, 4),
				},
				"GPU1Halted": {
					Description: ``,
					BitRange:    sysdec.BitRange(5, 5),
				},
				"IllegalAccessType1": {
					Description: ``,
					BitRange:    sysdec.BitRange(6, 6),
				},
				"IllegalAccessType0": {
					Description: ``,
					BitRange:    sysdec.BitRange(7, 7),
				},
				"MoreBitsSetInPending1": {
					Description: ``,
					BitRange:    sysdec.BitRange(8, 8),
				},
				"MoreBitsSetInPending2": {
					Description: ``,
					BitRange:    sysdec.BitRange(9, 9),
				},
			},
		},
		"Disable1": {
			Description: `Disable GPU interrupts from Group 1 (0-31): 
Writing a 1 to a bit will clear the corresponding IRQ enable bit. 
All other IRQ enable bits are unaffected.`,
			Size:          32,
			AddressOffset: 0x1C,
			Access:        sysdec.Access("w"),
			Field: map[string]*sysdec.FieldDef{
				"Aux": {
					Description: "Disable the interrupt from the three auxiliary devices",
					BitRange:    sysdec.BitRange(29, 29),
					Access:      sysdec.Access("w"),
				},
			},
		},
		"Disable2": {
			Description: `Disable GPU interrupts from Group 2 (32-63): 
Writing a 1 to a bit will clear the corresponding IRQ enable bit. 
All other IRQ enable bits are unaffected.`,
			Size:          32,
			AddressOffset: 0x20,
			Access:        sysdec.Access("w"),
			Field: map[string]*sysdec.FieldDef{
				"GPIO0": {
					Description: ``,
					BitRange:    sysdec.BitRange(17, 17),
					Access:      sysdec.Access("r"),
				},
				"GPIO1": {
					Description: ``,
					BitRange:    sysdec.BitRange(18, 18),
					Access:      sysdec.Access("r"),
				},
				"GPIO2": {
					Description: ``,
					BitRange:    sysdec.BitRange(19, 19),
					Access:      sysdec.Access("r"),
				},
				"GPIO3": {
					Description: ``,
					BitRange:    sysdec.BitRange(20, 20),
					Access:      sysdec.Access("r"),
				},
				"I2C": {
					Description: ``,
					BitRange:    sysdec.BitRange(21, 21),
					Access:      sysdec.Access("r"),
				},
				"SPI": {
					Description: ``,
					BitRange:    sysdec.BitRange(22, 22),
					Access:      sysdec.Access("r"),
				},
				"PCM": {
					Description: ``,
					BitRange:    sysdec.BitRange(22, 22),
					Access:      sysdec.Access("r"),
				},
				"UART": {
					Description: `This is not the mini UART, it's UART0.'`,
					BitRange:    sysdec.BitRange(24, 24),
					Access:      sysdec.Access("r"),
				},
			},
		},
		"DisableBasic": {
			Description: `Writing a 1 to a bit will clear the 
corresponding IRQ enable bit. All other IRQ enable bits are unaffected.`,
			Size:          8,
			AddressOffset: 0x24,
			Access:        sysdec.Access("w"),
			Field: map[string]*sysdec.FieldDef{
				"ARMTimer": {
					Description: `ARM Timer is the per-core timer.
https://www.raspberrypi.org/documentation/hardware/raspberrypi/bcm2836/QA7_rev3.4.pdf
Note that this is sometimes called the "local" timer to distinguish from the
chapter 14 and chapter 12 timers.`,
					BitRange: sysdec.BitRange(0, 0),
				},
				"ARMMailbox": {
					Description: ``,
					BitRange:    sysdec.BitRange(1, 1),
				},
				"ARMDoorbell0": {
					Description: ``,
					BitRange:    sysdec.BitRange(2, 2),
				},
				"ARMDoorbell1": {
					Description: ``,
					BitRange:    sysdec.BitRange(3, 3),
				},
				"GPU0Halted": {
					Description: `(Or GPU1 halted if bit 10 of control 
register 1 is set)`,
					BitRange: sysdec.BitRange(4, 4),
				},
				"GPU1Halted": {
					Description: ``,
					BitRange:    sysdec.BitRange(5, 5),
				},
				"IllegalAccessType1": {
					Description: ``,
					BitRange:    sysdec.BitRange(6, 6),
				},
				"IllegalAccessType0": {
					Description: ``,
					BitRange:    sysdec.BitRange(7, 7),
				},
				"MoreBitsSetInPending1": {
					Description: ``,
					BitRange:    sysdec.BitRange(8, 8),
				},
				"MoreBitsSetInPending2": {
					Description: ``,
					BitRange:    sysdec.BitRange(9, 9),
				},
			},
		},
	},
}
