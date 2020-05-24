package sys

import "tools/sysdec"

var QA7 = &sysdec.PeripheralDef{
	Version: 1,
	Description: `
This is a crucial "peripheral" that defines how the ARM 53A will handle
various kinds of interrupts.  You have to route things to the proper
core with this peripheral or no interrupts will arrive at your core.

https://www.raspberrypi.org/documentation/hardware/raspberrypi/bcm2836/QA7_rev3.4.pdf`,
	AddressBlock: sysdec.AddressBlockDef{BaseAddress: 0x0, Size: 0x100},
	Register: map[string]*sysdec.RegisterDef{
		"Control": {
			Description: `The control register is currently only used to 
control the 64-bit core timer.`,
			AddressOffset: 0x0,
			Size:          10,
			Field: map[string]*sysdec.FieldDef{
				"IncrementBy2": {
					Description: `Bit 9: Timer increment
This bit controls the step size of the 64-bit core timer . This may be 
important if you want the core timer to accurate represent the number of 
CPU cycles. The core timer pre-scaler is running of the APB clock. As the 
APB clock is running at half the speed of the ARM clock, you cannot get 
a timer value equal to the ARM clock cycles, even if the pre-scaler is set
to divide-by-one. This bit provides a means of getting close to the actual 
number of CPU cycles.
* If set the 64-bit core timer is incremented by 2.
* If clear the 64-bit core timer is incremented by 1.
This will still not get you the exact number of CPU cycles but will 
get you close to plus/minus one. Beware that if the core timer increment is 
set to 2 you will only get either all even or all odd values
(Depending on if the initial value you write to it is even or odd).`,
					Access:   sysdec.Access("rw"),
					BitRange: sysdec.BitRange(9, 9),
				},
				"ClockSourceAPBClock": {
					Description: `
Bit 8: Core timer clock source
This bit controls what the source clock is of the 64-bit Core timer. 
Actually it selects the source clock of the Core timer prescaler but 
that amounts to the same end-result. If set the 64-bit core timer 
pre-scaler is running of the APB clock. If clear the 64-bit core timer 
pre-scaler is running of the Crystal clock. Note that the APB clock 
is running at half the speed of the ARM clock. Thus the pre-scaler is 
only changing every second CPU clock cycle.`,
					Access:   sysdec.Access("rw"),
					BitRange: sysdec.BitRange(8, 8),
				},
			},
		},
		"CoreTimerPrescaler": {
			Description: `
timer_frequency = (2**31/prescaler) * input frequency, with
(Pre-scaler <= 2**31).

I have not found any information on how fast this timer should run. It 
seems common to run it of the processor clock. However that would not 
give a reliably timing signal when the frequency of the processor is variable. 
Therefore the source of the timer can come from either the external crystal 
or from a CPU related clock.

To give maximum flexibility to the timer speed there is a 32-bit pre-scaler. 
This prescaler can provide integer as well as fractional division ratios.

Thus setting the prescaler to 0x8000_0000 gives a divider ratio of 1. Setting 
the prescaler to 0 will stop the timer. To get a divider ratio of 19.2 use: 
2^31/19.2 = 0x06AA_AAAB. The value is rounded upwards and introduces an 
error of 8.9E-9 which is much lower than any ordinary crystal oscillator 
produces. Do not use timer values >2^31 (2147483648)`,
			AddressOffset: 0x08,
			Size:          32,
		},
		"GPUInterruptRouting": {
			Description: `This is how to connect the interrupt controller
to a core.

The GPU interrupt routing register controls where the IRQ and FIQ of the 
GPU are routed to.

The IRQ /FIQ can be connected to one processor core only. This also means that there is only one
possible GPU-IRQ/GPU-FIQ interrupt outstanding bit. `,
			Access:        sysdec.Access("rw"),
			Size:          4,
			AddressOffset: 0xC,
			Field: map[string]*sysdec.FieldDef{
				"GPUFIQRouting": {
					BitRange: sysdec.BitRange(3, 2),
					EnumeratedValue: map[string]*sysdec.EnumeratedValueDef{
						"FIQToCore0": {Value: 0b00},
						"FIQToCore1": {Value: 0b01},
						"FIQToCore2": {Value: 0b10},
						"FIQToCore3": {Value: 0b11},
					},
				},
				"GPUIRQRouting": {
					BitRange: sysdec.BitRange(2, 1),
					EnumeratedValue: map[string]*sysdec.EnumeratedValueDef{
						"IRQToCore0": {Value: 0b00},
						"IRQToCore1": {Value: 0b01},
						"IRQToCore2": {Value: 0b10},
						"IRQToCore3": {Value: 0b11},
					},
				},
			},
		},
		"Lower32": {
			Description: `64-bit core timer read/write, LS 32 bits
When reading returns the current 32 LS bit of the 64 timer and triggers 
storing a copy of the MS 32 bits.  When writing: stores a copy of the 
32 bits written. That copy is transferred to the timer when the MS 32
bits are written`,
			Access:        sysdec.Access("rw"),
			Size:          32,
			AddressOffset: 0x1C,
		},
		"Upper32": {
			Description: ` 64-bit core timer read/write, MS 32 bits
When reading returns the status of the core timer-read-hold register. That 
register is loaded when the user does a read of the LS-32 timer bits. There 
is little sense in reading this register without first doing a read from 
the LS-32 bit register. When writing the value is written to the timer, 
as well as the value previously written to the LS-32 write-holding bit 
register. There is little sense in writing this register without first doing 
a write to the LS-32 bit register.`,
			Access:        sysdec.Access("rw"),
			Size:          32,
			AddressOffset: 0x20,
		},
		"LocalInterrupt": {
			Description: `The local interrupt routing register is 
described here as the local time is the only local interrupt source
present.`,
			Access:        sysdec.Access("rw"),
			Size:          3,
			AddressOffset: 0x24,
			Field: map[string]*sysdec.FieldDef{
				"LocalTimerRoute": {
					BitRange: sysdec.BitRange(2, 0),
					EnumeratedValue: map[string]*sysdec.EnumeratedValueDef{
						"Core0IRQ": {Value: 0b000},
						"Core1IRQ": {Value: 0b001},
						"Core2IRQ": {Value: 0b010},
						"Core3IRQ": {Value: 0b011},
						"Core0FIQ": {Value: 0b100},
						"Core1FIQ": {Value: 0b101},
						"Core2FIQ": {Value: 0b110},
						"Core3FIQ": {Value: 0b111},
					},
				},
			},
		},
		"LocalTimerControl": {
			Description: `The code has a single local timer which can 
generate interrupts. The local timer ALWAYS gets its timing pulses from the 
Crystal clock. You get a 'timing pulse' every clock EDGE. Thus a 19.2 
MHz crystal gives 38.4 M pulses/second.  The local timer has a 28-bit 
programmable divider which gives a lowest frequency of 38.4/2^28 = 0.14Hz.
The local timer counts down and re-loads when it gets to zero. At the 
same time an interrupt-flag is set. The user must clear the interrupt flag. 
There is no detection if the interrupt flag is still set when the next
time the local timer re-loads. 

When disabled the local timer loads the re-load value. Bit 32 is the 
status of the interrupt flag. The interrupt flag is always set upon a 
re-load and is independent of the interrupt enable bit. An interrupt is
generated as long as the interrupt flag is set and the interrupt-enable 
bit is set.`,
			Access:        sysdec.Access("rw"),
			Size:          32,
			AddressOffset: 0x34,
			Field: map[string]*sysdec.FieldDef{
				"InterruptPending": {
					BitRange: sysdec.BitRange(31, 31),
					Access:   sysdec.Access("r"),
				},
				"InterruptEnable": {
					BitRange: sysdec.BitRange(29, 29),
					Access:   sysdec.Access("rw"),
				},
				"TimerEnable": {
					BitRange: sysdec.BitRange(28, 28),
					Access:   sysdec.Access("rw"),
				},
				"ReloadValue": {
					BitRange: sysdec.BitRange(27, 0),
					Access:   sysdec.Access("rw"),
				},
			},
		},
		"LocalTimerClearReload": {
			Description: `The interrupt flag is clear by writing bit 31 
high of the local timer IRQ clear & reload register. 

The IRQ clear & reload register has one extra bit: when writing bit 30 high, 
the local timer is immediately reloaded without generating an interrupt. 
As such it can also be used as a watchdog timer. `,
			Size:          32,
			AddressOffset: 0x38,
			Field: map[string]*sysdec.FieldDef{
				"Clear": {
					BitRange: sysdec.BitRange(31, 31),
					Access:   sysdec.Access("w"),
				},
				"Reload": {
					BitRange: sysdec.BitRange(30, 30),
					Access:   sysdec.Access("w"),
				},
			},
		},
		"TimerInterruptControl": {
			Description: `For each core, you can control how the timers
route to the FIQ and IRQ pins.  For all the fields, 1 enables an FIQ
or IRQ, 0 disables it.`,
			Access:        sysdec.Access("rw"),
			Size:          8,
			AddressOffset: 0x40,
			Dim:           4,
			DimIncrement:  4,
			Field: map[string]*sysdec.FieldDef{
				"VirtualTimerFIQ": {
					Description: `CNTVIRQ FIQ control. If set, this bit 
overrides the IRQ bit (3).`,
					BitRange: sysdec.BitRange(7, 7),
				},
				"HypervisorTimerFIQ": {
					Description: `CNTHPIRQ FIQ control. If set, this bit 
overrides the IRQ bit (2).).`,
					BitRange: sysdec.BitRange(6, 6),
				},
				"PhysicalNonSecureTimerFIQ": {
					Description: `nCNTPNSIRQ FIQ control. If set, 
this bit overrides the IRQ bit (1).`,
					BitRange: sysdec.BitRange(5, 5),
				},
				"PhysicalSecureTimerFIQ": {
					Description: `nCNTPSIRQ FIQ control. If set, 
this bit overrides the IRQ bit (0).`,
					BitRange: sysdec.BitRange(4, 4),
				},
				"VirtualTimerIRQ": {
					Description: `nCNTVIRQ IRQ control.
This bit is only valid if bit 7 is clear otherwise it is ignored`,
					BitRange: sysdec.BitRange(3, 3),
				},
				"HypervisorTimerIRQ": {
					Description: `nCNTHPIRQ IRQ control.
This bit is only valid if bit 6 is clear otherwise it is ignored.`,
					BitRange: sysdec.BitRange(2, 2),
				},
				"PhysicalNonSecureTimerIRQ": {
					Description: `nCNTPNSIRQ IRQ control.
This bit is only valid if bit 5 is clear otherwise it is ignored`,
					BitRange: sysdec.BitRange(1, 1),
				},
				"PhysicalSecureTimerIRQ": {
					Description: `nCNTPSIRQ IRQ control.
This bit is only valid if bit 4 is clear otherwise it is ignored.`,
					BitRange: sysdec.BitRange(0, 0),
				},
			},
		},
		"IRQSource": {
			Description:   ``,
			Size:          12,
			AddressOffset: 0x60,
			Dim:           4,
			DimIncrement:  4,
			Access:        sysdec.Access("rw"),
			Field: map[string]*sysdec.FieldDef{
				"LocalTimer": {
					BitRange: sysdec.BitRange(11, 11),
				},
				"GPU": {
					Description: `Can be high in one core only`,
					BitRange:    sysdec.BitRange(8, 8),
				},
				"Mailbox3": {
					BitRange: sysdec.BitRange(7, 7),
				},
				"Mailbox2": {
					BitRange: sysdec.BitRange(6, 6),
				},
				"Mailbox1": {
					BitRange: sysdec.BitRange(5, 5),
				},
				"Mailbox0": {
					BitRange: sysdec.BitRange(4, 4),
				},
				"VirtualTimer": {
					BitRange: sysdec.BitRange(3, 3),
				},
				"HypervisorTimer": {
					BitRange: sysdec.BitRange(2, 2),
				},
				"PhysicalNonSecureTimer": {
					BitRange: sysdec.BitRange(1, 1),
				},
				"PhysicalSecureTimer": {
					BitRange: sysdec.BitRange(0, 0),
				},
			},
		},
		"FIQSource": {
			Description:   ``,
			Size:          12,
			AddressOffset: 0x70,
			Dim:           4,
			DimIncrement:  4,
			Access:        sysdec.Access("rw"),
			Field: map[string]*sysdec.FieldDef{
				"LocalTimer": {
					BitRange: sysdec.BitRange(11, 11),
				},
				"GPU": {
					Description: `Can be high in one core only`,
					BitRange:    sysdec.BitRange(8, 8),
				},
				"Mailbox3": {
					BitRange: sysdec.BitRange(7, 7),
				},
				"Mailbox2": {
					BitRange: sysdec.BitRange(6, 6),
				},
				"Mailbox1": {
					BitRange: sysdec.BitRange(5, 5),
				},
				"Mailbox0": {
					BitRange: sysdec.BitRange(4, 4),
				},
				"VirtualTimer": {
					BitRange: sysdec.BitRange(3, 3),
				},
				"HypervisorTimer": {
					BitRange: sysdec.BitRange(2, 2),
				},
				"PhysicalNonSecureTimer": {
					BitRange: sysdec.BitRange(1, 1),
				},
				"PhysicalSecureTimer": {
					BitRange: sysdec.BitRange(0, 0),
				},
			},
		},
	},
}
