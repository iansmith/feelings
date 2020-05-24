package sys

import "tools/sysdec"

var Aux = &sysdec.PeripheralDef{
	Version: 1,
	Description: `Auxiliary Peripherals: The SOC has three Auxiliary 
peripherals: One mini UART and two SPI masters. These three peripheral are 
grouped together as they share the same area in the peripheral register map 
and they share a common interrupt. Also all three are controlled by the 
auxiliary enable register.

There are two Auxiliary registers which control all three devices. One is the 
interrupt status register, the second is the Auxiliary enable register. The 
Auxiliary IRQ status register can help to hierarchically determine the source 
of an interrupt.

The mini UART is a secondary low throughput4 UART intended to be used as a 
console. It needs to be enabled before it can be used. It is also recommended 
that the correct GPIO function mode is selected before enabling the mini UART.
The mini Uart has the following features:
• 7 or 8 bit operation.
• 1 start and 1 stop bit.
• No parities.
• Break generation.
• 8 symbols deep FIFOs for receive and transmit.
• SW controlled RTS, SW readable CTS.
• Auto flow control with programmable FIFO level.
• 16550 like registers.
• Baudrate derived from system clock.
This is a mini UART and it does NOT have the following capabilities:
• Break detection
• Framing errors detection.
• Parity bit
• Receive Time-out interrupt
• DCD, DSR, DTR or RI signals.
The implemented UART is not a 16650 compatible UART However as far as possible 
the first 8 control and status registers are laid out like a 16550 UART. All 
16550 register bits which are not supported can be written but will be 
ignored and read back as 0. All control bits for simple UART receive/transmit 
operations are available.

Currently, the two SPI masters are not described in this document.`,
	AddressBlock: sysdec.AddressBlockDef{BaseAddress: 0x21_5000, Size: 0x68},
	Register: map[string]*sysdec.RegisterDef{
		"IRQ": {
			Description: `The IRQ register is used to check any pending
interrupts which may be asserted by the three Auxiliary sub blocks.`,
			AddressOffset: 0x0,
			Size:          3,
			Access:        sysdec.Access("r"),
			Field: map[string]*sysdec.FieldDef{
				"SPI2": {
					Description: `If set the SPI 2 module has an interrupt 
pending.`,
					BitRange: sysdec.BitRange(2, 2),
					Access:   sysdec.Access("r"),
				},
				"SPI1": {
					Description: `If set the SPI 1 module has an interrupt 
pending.`,
					BitRange: sysdec.BitRange(1, 1),
					Access:   sysdec.Access("r"),
				},
				"MiniUART": {
					Description: `If set the Mini UART module has an interrupt 
pending.`,
					BitRange: sysdec.BitRange(0, 0),
					Access:   sysdec.Access("r"),
				},
			},
		},
		"Enable": {
			Description: `The AUXENB register is used to enable the three modules; 
UART, SPI1, SPI2.`,
			AddressOffset: 0x4,
			Size:          3,
			Field: map[string]*sysdec.FieldDef{
				"SPI2": {
					Description: `If set the SPI 2 module is enabled.
If clear the SPI 2 module is disabled. That also disables any SPI 2 module 
register access`,
					BitRange: sysdec.BitRange(2, 2),
					Access:   sysdec.Access("rw"),
				},
				"SPI1": {
					Description: `If set the SPI 1 module is enabled.
If clear the SPI 1 module is disabled. That also disables any SPI 1 module 
register access`,
					BitRange: sysdec.BitRange(1, 1),
					Access:   sysdec.Access("rw"),
				},
				"MiniUART": {
					Description: `If set the mini UART is enabled. The UART 
will immediately start receiving data, especially if the UART1_RX [sic?] 
line is low. If clear the mini UART is disabled. That also disables any 
mini UART register access`,
					BitRange: sysdec.BitRange(0, 0),
					Access:   sysdec.Access("rw"),
				},
			},
		},
		"MUData": {
			Description: `The AUX_MU_IO_REG register is primary used to write 
data to and read data from the UART FIFOs. If the DLAB bit in the line 
control register is set this register gives access to the LS 8 bits of the 
aud rate. (Pro Tip: there is easier access to the baud rate register, so
don't bother with this DLAB bit.) Aux, MU=MinuUART, IO=Input/Output reg.`,
			AddressOffset: 0x40,
			Size:          8,
			Field: map[string]*sysdec.FieldDef{
				"Transmit": {
					Description: `Transmit data write.  Data written is put 
in the transmit FIFO (provided it is not full).`,
					BitRange: sysdec.BitRange(7, 0),
					Access:   sysdec.Access("w"),
				},
				"Receive": {
					Description: `Receive data read.  Data read is taken 
from the receive FIFO (provided it is not empty)`,
					BitRange: sysdec.BitRange(7, 0),
					Access:   sysdec.Access("r"),
				},
			},
		},
		"MUIER": {
			Description: `The AUX_MU_IER_REG register is primary used to 
enable interrupts. If the DLAB bit in the line control register is set 
this register gives access to the MS 8 bits of the baud rate.  Page 12
of the BCM2835 documentation has numerous mistakes related to this register.
(Pro Tip: there is easier access to the baud rate register, so don't bother
with this DLAB bit.) Aux, MU=MinuUART, IER=Interrupt Enable Reg.`,
			AddressOffset: 0x44,
			Size:          4,
			Field: map[string]*sysdec.FieldDef{
				"WriteErr": {
					Description: `Unclear if this is the way to get
transmission errors.`,
					BitRange: sysdec.BitRange(3, 3),
					Access:   sysdec.Access("rw"),
				},
				"ReadErr": {
					Description: `Undocumented enable for parity, framing
or overrun error.`,
					BitRange: sysdec.BitRange(2, 2),
					Access:   sysdec.Access("rw"),
				},
				"Transmit": {
					Description: `If this bit is set the interrupt line is 
asserted whenever the transmit FIFO is empty. If this bit is clear no 
transmit interrupts are generated.`,
					BitRange: sysdec.BitRange(1, 1),
					Access:   sysdec.Access("rw"),
				},
				"Receive": {
					Description: `If this bit is set the interrupt line is 
asserted whenever the receive FIFO holds at least 1 byte. If this bit is 
clear no receive interrupts are generated.`,
					BitRange: sysdec.BitRange(0, 0),
					Access:   sysdec.Access("rw"),
				},
			},
		},
		"MUIIR": {
			Description: `The AUX_MU_IIR_REG register shows the interrupt 
status.  It also has two FIFO enable status bits and (when writing) FIFO 
clear bits. Aux, MU=MinuUART, IIR=Interrupt Info Reg.`,
			AddressOffset: 0x48,
			Size:          8,
			Field: map[string]*sysdec.FieldDef{
				"FIFOEnabled": {
					Description: `Both bits always read as 1 as the FIFOs are 
always enabled`,
					BitRange: sysdec.BitRange(7, 6),
					Access:   sysdec.Access("r"),
				},
				"InterruptID": {
					Description: `On read this register shows the interrupt 
ID bit 
00 : No interrupts
01 : Transmit holding register empty
10 : Receiver holds valid byte
11 : <Not possible> `,
					BitRange: sysdec.BitRange(2, 1),
					Access:   sysdec.Access("r"),
					EnumeratedValue: map[string]*sysdec.EnumeratedValueDef{
						"NoInterrupt":   {Value: 0b00},
						"TransmitReady": {Value: 0b01},
						"ReceiverReady": {Value: 0b10},
					},
				},
				"ClearFIFO": {
					Description: `On write:
Writing with bit 1 set will clear the receive FIFO. 
Writing with bit 2 set will clear the transmit FIFO`,
					BitRange: sysdec.BitRange(2, 1),
					Access:   sysdec.Access("w"),
					EnumeratedValue: map[string]*sysdec.EnumeratedValueDef{
						"ZeroReceive":            {Value: 0b01},
						"ZeroTransmit":           {Value: 0b10},
						"ZeroTransmitAndReceive": {Value: 0b11},
					},
				},
				"InterruptPending": {
					Description: `This bit is clear whenever an interrupt 
is pending`,
					BitRange: sysdec.BitRange(0, 0),
					Access:   sysdec.Access("r"),
				},
			},
		},
		"MULCR": {
			Description: `The AUX_MU_LCR_REG register controls the line data 
format and gives access to the baudrate register. 
Aux, MU=MinuUART, LCR=Line Control Register. `,
			AddressOffset: 0x4C,
			Size:          8,
			Field: map[string]*sysdec.FieldDef{
				"Break": {
					Description: `If set high the UART1_TX [sic?] line is 
pulled low continuously. If held for at least 12 bits times that will 
indicate a break condition.`,
					BitRange: sysdec.BitRange(6, 6),
					Access:   sysdec.Access("rw"),
				},
				"DataSize": {
					Description: `If clear the UART works in 7-bit mode.
If set the UART works in 8-bit mode. The documentation in the data sheet is 
actually wrong on this one.  Used the known to work values.`,
					BitRange: sysdec.BitRange(1, 0),
					Access:   sysdec.Access("rw"),
					EnumeratedValue: map[string]*sysdec.EnumeratedValueDef{
						"SevenBit": {Value: 0b00},
						"EightBit": {Value: 0b00},
					},
				},
			},
		},
		"MUMCR": {
			Description: `The AUX_MU_MCR_REG register controls the 'modem' 
signals. Aux, MU=MinuUART, MCR=Modem Control Register.`,
			AddressOffset: 0x50,
			Size:          2,
			Field: map[string]*sysdec.FieldDef{
				"RTS": {
					Description: `If clear the UART1_RTS [sic?] line is high. 
If set the UART1_RTS [sic?] line is low.This bit is ignored if the RTS is used 
for auto-flow control. See the Mini Uart Extra Control register description)`,
					BitRange: sysdec.BitRange(1, 1),
					Access:   sysdec.Access("rw"),
				},
			},
		},
		"MULSR": {
			Description: `The AUX_MU_LSR_REG register shows the data status.
 Aux, MU=MinuUART, LSR=Line Status Register.`,
			AddressOffset: 0x54,
			Size:          7,
			Field: map[string]*sysdec.FieldDef{
				"TransmitterIdle": {
					Description: `This bit is set if the transmit FIFO is empty 
and the transmitter is idle (it has finished shifting out the last bit).`,
					BitRange: sysdec.BitRange(6, 6),
					Access:   sysdec.Access("r"),
				},
				"TransmitterEmpty": {
					Description: `This bit is set if the transmit FIFO can 
accept at least one byte.`,
					BitRange: sysdec.BitRange(5, 5),
					Access:   sysdec.Access("r"),
				},
				"ReceiverOverrun": {
					Description: `This bit is set if there was a receiver 
overrun. That is: one or more characters arrived whilst the receive FIFO 
was full. The newly arrived charters have been discarded. This bit is 
cleared each time this register is read. To do a non-destructive read 
of this overrun bit use the Mini Uart Extra Status register.`,
					BitRange: sysdec.BitRange(1, 1),
					Access:   sysdec.Access("r"),
				},
				"DataReady": {
					Description: `This bit is set if the receive FIFO holds 
at least 1 symbol.`,
					BitRange: sysdec.BitRange(0, 0),
					Access:   sysdec.Access("r"),
				},
			},
		},
		"MUMSR": {
			Description: `The AUX_MU_LSR_REG register shows the "modem" status.
 Aux, MU=MinuUART, MSR=Modem Status Register.`,
			AddressOffset: 0x58,
			Size:          6,
			Field: map[string]*sysdec.FieldDef{
				"CTS": {
					Description: `This bit is the inverse of the UART1_CTS 
[sic?] input Thus, if set the UART1_CTS [sic?] pin is low if clear the 
UART1_CTS [sic?] pin is high`,
					BitRange: sysdec.BitRange(5, 5),
					Access:   sysdec.Access("r"),
				},
			},
		},
		"AuxMUScratch": {
			Description:   ``,
			AddressOffset: 0x58,
			Size:          6,
			Field: map[string]*sysdec.FieldDef{
				"CTS": {
					Description: `AUX_MU_SCRATCH is a single byte storage.`,
					BitRange:    sysdec.BitRange(7, 0),
					Access:      sysdec.Access("rw"),
				},
			},
		},
		"MUCNTL": {
			Description: `The AUX_MU_CNTL_REG provides access to some extra 
useful and nice features not found on a normal 16550 UART.  Sometimes this 
is called the 'extra' control register. "`,
			AddressOffset: 0x60,
			Size:          8,
			Field: map[string]*sysdec.FieldDef{
				"CTSAssertLevel": {
					Description: `This bit allows one to invert the CTS auto 
flow operation polarity. If set the CTS auto flow assert level is low.
If clear the CTS auto flow assert level is high.`,
					BitRange: sysdec.BitRange(7, 7),
					Access:   sysdec.Access("rw"),
				},
				"RTSAssertLevel": {
					Description: `This bit allows one to invert the RTS 
auto flow operation polarity. If set the RTS auto flow assert level is low.
If clear the RTS auto flow assert level is high.`,
					BitRange: sysdec.BitRange(6, 6),
					Access:   sysdec.Access("rw"),
				},
				"RTSAutoFlowLevel": {
					Description: `These two bits specify at what receiver 
FIFO level the RTS line is de-asserted in auto-flow mode.
00 : De-assert RTS when the receive FIFO has 3 empty spaces left.
01 : De-assert RTS when the receive FIFO has 2 empty spaces left.
10 : De-assert RTS when the receive FIFO has 1 empty space left.
11 : De-assert RTS when the receive FIFO has 4 empty spaces left.`,
					BitRange: sysdec.BitRange(5, 4),
					Access:   sysdec.Access("rw"),
					EnumeratedValue: map[string]*sysdec.EnumeratedValueDef{
						"DeassertRTSWith3Empty": {Value: 0b00},
						"DeassertRTSWith2Empty": {Value: 0b01},
						"DeassertRTSWith1Empty": {Value: 0b10},
						"DeassertRTSWith4Empty": {Value: 0b11},
					},
				},
				"EnableTransmitAutoFlowControlUsingCTS": {
					Description: `If this bit is set the transmitter will 
stop if the CTS line is de-asserted.  If this bit is clear the transmitter 
will ignore the status of the CTS line`,
					BitRange: sysdec.BitRange(3, 3),
					Access:   sysdec.Access("rw"),
				},
				"EnableReceiveAutoFlowControlUsingRTS": {
					Description: `If this bit is set the RTS line will 
de-assert if the receive FIFO reaches it 'auto flow' level. In fact the 
RTS line will behave as an RTR (Ready To Receive) line.
If this bit is clear the RTS line is controlled by the AUX_MU_MCR_REG 
register bit 1.`,
					BitRange: sysdec.BitRange(2, 2),
					Access:   sysdec.Access("rw"),
				},
				"TransmitterEnable": {
					Description: `If this bit is set the mini UART transmitter 
is enabled. If this bit is clear the mini UART transmitter is disabled.
If this bit is set [sic?] no new symbols will be sent the transmitter. 
Any symbols in progress of transmission will be finished.`,
					BitRange: sysdec.BitRange(1, 1),
					Access:   sysdec.Access("rw"),
				},
				"ReceiverEnable": {
					Description: `If this bit is set the mini UART receiver 
is enabled. If this bit is clear the mini UART receiver is disabled.
If this bit is set [sic?] no new symbols will be accepted by the receiver. 
Any symbols in progress of reception will be finished.`,
					BitRange: sysdec.BitRange(0, 0),
					Access:   sysdec.Access("rw"),
				},
			},
		},
		"MUStat": {
			Description: `The AUX_MU_STAT_REG provides a lot of useful 
information about the internal status of the mini UART not found on a normal 
16550 UART.  This is sometimes called the "extra" status register. '`,
			AddressOffset: 0x64,
			Size:          28,
			Field: map[string]*sysdec.FieldDef{
				"TransmitFIFOFillLevel": {
					Description: `These bits shows how many symbols are stored 
in the transmit FIFO. The value is in the range 0-8`,
					BitRange: sysdec.BitRange(27, 24),
					Access:   sysdec.Access("r"),
				},
				"ReceiveFIFOFillLevel": {
					Description: `These bits shows how many symbols are stored 
in the receive FIFO. The value is in the range 0-8`,
					BitRange: sysdec.BitRange(19, 16),
					Access:   sysdec.Access("r"),
				},
				"TransmitterDone": {
					Description: `This bit is set if the transmitter is idle 
and the transmit FIFO is empty.  It is a logical AND of bits 2 and 8`,
					BitRange: sysdec.BitRange(9, 9),
					Access:   sysdec.Access("r"),
				},
				"TransmitFIFOEmpty": {
					Description: `If this bit is set the transmitter FIFO is 
empty. Thus it can accept 8 symbols.`,
					BitRange: sysdec.BitRange(8, 8),
					Access:   sysdec.Access("r"),
				},
				"CTSLine": {
					Description: `This bit shows the status of the 
UART1_CTS [sic?] line.`,
					BitRange: sysdec.BitRange(7, 7),
					Access:   sysdec.Access("r"),
				},
				"RTSLine": {
					Description: `This bit shows the status of the 
UART1_RTS [sic?] line. This bit is useful only in receive Auto flow-control 
mode as it shows the status of the RTS line.`,
					BitRange: sysdec.BitRange(6, 6),
					Access:   sysdec.Access("r"),
				},
				"TransmitFIFOFull": {
					Description: `This is the inverse of bit 1.`,
					BitRange:    sysdec.BitRange(5, 5),
					Access:      sysdec.Access("r"),
				},
				"ReceiverOverrun": {
					Description: `This bit is set if there was a receiver 
overrun. That is: one or more characters arrived whilst the receive FIFO 
was full. The newly arrived characters have been discarded. This bit is 
cleared each time the AUX_MU_LSR_REG register is read.`,
					BitRange: sysdec.BitRange(4, 4),
					Access:   sysdec.Access("r"),
				},
				"TransmitterIdle": {
					Description: `If this bit is set the transmitter is idle. 
If this bit is clear the transmitter is idle [sic].
Note that the bit will set only for a short time if the transmit FIFO 
contains data. Normally you want to use bit 9: Transmitter done.`,
					BitRange: sysdec.BitRange(3, 3),
					Access:   sysdec.Access("r"),
				},
				"ReceiverIdle": {
					Description: `If this bit is set the receiver is idle.
If this bit is clear the receiver is busy. This bit can change unless the 
receiver is disabled.

This bit is only useful if the receiver is disabled. The normal use is to 
disable the receiver. Then check (or wait) until the bit is set. Now you 
can be sure that no new symbols will arrive. (e.g. now you can change the 
baudrate...)`,
					BitRange: sysdec.BitRange(2, 2),
					Access:   sysdec.Access("r"),
				},
				"SpaceAvailable": {
					Description: `If this bit is set the mini UART transmitter 
FIFO can accept at least one more symbol. If this bit is clear the mini 
UART transmitter FIFO is full`,
					BitRange: sysdec.BitRange(1, 1),
					Access:   sysdec.Access("r"),
				},
				"SymbolAvailable": {
					Description: `If this bit is set the mini UART receive 
FIFO contains at least 1 symbol If this bit is clear the mini UART receiver 
FIFO is empty`,
					BitRange: sysdec.BitRange(0, 0),
					Access:   sysdec.Access("r"),
				},
			},
		},
		"MUBaud": {
			Description: `The AUX_MU_BAUD register allows direct access to the 
16-bit wide baudrate counter.`,
			AddressOffset: 0x68,
			Size:          16,
			Field: map[string]*sysdec.FieldDef{
				"Baudrate": {
					Description: `mini UART baudrate counter.  This is the 
same register as is accessed using the LABD bit and the first two registers, 
but much easier to access.  PRO TIP: Don't bother with the other stuff.`,
					BitRange: sysdec.BitRange(15, 0),
					Access:   sysdec.Access("rw"),
				},
			},
		},
	},
}
