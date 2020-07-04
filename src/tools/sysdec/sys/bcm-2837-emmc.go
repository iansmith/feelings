package sys

import "tools/sysdec"

var EMMC = &sysdec.PeripheralDef{
	Version: 1,
	Description: `The External Mass Media Controller (EMMC) is an embedded 
MultiMedia and SD card interface provided by Arasan. It is compliant to the 
following standards:
• SDTM Host Controller Standard Specification Version 3.0 Draft 1.0
• SDIOTM card specification version 3.0
• SDTM Memory Card Specification Draft version 3.0
• SDTM Memory Card Security Specification version 1.01
• MMCTM Specification version 3.31,4.2 and 4.4

Because the EMMC module shares pins with other functionality it must be 
selected in the GPIO interface. Please refer to the GPIO section for 
further details.

The interface to the card uses its own clock clk_emmc which is provided by 
the clock manager module. The frequency of this clock should be selected 
between 50 MHz and 100 MHz. Having a separate clock allows high performance 
access to the card even if the VideoCore runs at a reduced clock frequency. 
The EMMC module contains its own internal clock divider to generate the 
card’s clock from clk_emmc.

Additionally can the sampling clock for the response and data from the 
card be delayed in up to 40 steps with a configurable delay between 
200ps to 1100ps per step typically. The delay is intended to cancel 
the internal delay inside the card (up to 14ns) when reading. The delay 
per step will vary with temperature and supply voltage. Therefore it is 
better to use a bigger delay than necessary as there is no restriction for 
the maximum delay.

The EMMC module handles the handshaking process on the command and data 
lines and all CRC processing automatically.`,
	AddressBlock: sysdec.AddressBlockDef{BaseAddress: 0x30_0000, Size: 0xfc},
	Register: map[string]*sysdec.RegisterDef{
		"Arg2": {
			Description: `This register contains the argument for the SD card 
specific command ACMD23 (SET_WR_BLK_ERASE_COUNT). ARG2 must be set before the 
ACMD23 command is issued using the CMDTM register.`,
			AddressOffset: 0x0,
			Size:          32,
			Access:        sysdec.Access("rw"),
		},
		"BlockSizeAndCount": {
			Description: `This register must not be accessed or modified while 
any data transfer between card and host is ongoing.

It contains the number and size in bytes for data blocks to be transferred. 
Please note that the EMMC module restricts the maximum block size to the 
size of the internal data FIFO which is 1k bytes.

BLKCNT is used to tell the host how many blocks of data are to be transferred. 
Once the data transfer has started and the TM_BLKCNT_EN bit in the CMDTM 
register is set the EMMC module automatically decreases the BNTCNT value as 
the data blocks are transferred and stops the transfer once BLKCNT reaches 0.`,
			AddressOffset: 0x4,
			Size:          32,
			Access:        sysdec.Access("rw"),
			Field: map[string]*sysdec.FieldDef{
				"BlkCnt": {
					Description: `Number of blocks to be transferred`,
					BitRange:    sysdec.BitRange(31, 16),
				},
				"BlkSize": {
					Description: `Block size in bytes`,
					BitRange:    sysdec.BitRange(9, 0),
				},
			},
		},
		"Argument": {
			Description: `This register contains the arguments for all 
commands except for the SD card specific command ACMD23 which uses ARG2. 
ARG1 must be set before the command is issued using the CMDTM register.`,
			AddressOffset: 0x8,
			Size:          32,
			Access:        sysdec.Access("rw"),
		},
		"CommandTransferMode": {
			Description: `This register is used to issue commands to the card. 
Besides the command it also contains flags informing the EMMC module what 
card response and type of data transfer to expect. Incorrect flags will 
result in strange behaviour.

For data transfers two modes are supported: either transferring a single 
block of data or several blocks of the same size. The SD card uses two 
different sets of commands to differentiate between them but the host needs 
to be additionally configured using TM_MULTI_BLOCK. It is important that 
this bit is set correct for the command sent to the card, i.e. 1 for CMD18 
and CMD25 and 0 for CMD17 and CMD24. Multiple block transfer gives a better 
performance.

The BLKSIZECNT register is used to configure the size and number of blocks 
to be transferred. If bit TM_BLKCNT_EN of this register is set the transfer 
stops automatically after the number of data blocks configured in the 
BLKSIZECNT register has been transferred.

The TM_AUTO_CMD_EN bits can be used to make the host to send automatically 
a command to the card telling it that the data transfer has finished once 
the BLKCNT bits in the BLKSIZECNT register are 0.`,
			AddressOffset: 0xC,
			Size:          32,
			Access:        sysdec.Access("rw"),
			Field: map[string]*sysdec.FieldDef{
				"CommandIndex": {
					Description: `Index of the command to be issued to the card`,
					BitRange:    sysdec.BitRange(29, 24),
				},
				"CommandType": {
					Description: `Type of command to be issued to the card: 
00 = normal
01 = suspend (the current data transfer) 
10 = resume (the last data transfer)
11 = abort (the current data transfer)`,
					BitRange: sysdec.BitRange(23, 22),
					EnumeratedValue: map[string]*sysdec.EnumeratedValueDef{
						"Normal":  {Value: 0b00},
						"Suspend": {Value: 0b01},
						"Resume":  {Value: 0b10},
						"Abort":   {Value: 0b11},
					},
				},
				"CommandIsData": {
					Description: `Command involves data transfer`,
					BitRange:    sysdec.BitRange(21, 21),
					Access:      sysdec.Access("rw"),
				},
				"EnableCommandIndexCheck": {
					Description: `Check that the response has the same index 
as the command`,
					BitRange: sysdec.BitRange(20, 20),
					Access:   sysdec.Access("rw"),
				},
				"EnableCommandCRCCheck": {
					Description: `Check that the response's CRC`,
					BitRange:    sysdec.BitRange(19, 19),
					Access:      sysdec.Access("rw"),
				},
				"CommandResponseType": {
					Description: `Type of expected response from card`,
					BitRange:    sysdec.BitRange(18, 17),
					Access:      sysdec.Access("rw"),
					EnumeratedValue: map[string]*sysdec.EnumeratedValueDef{
						"None":            {Value: 0b00},
						"Bits136":         {Value: 0b01},
						"Bits48":          {Value: 0b10},
						"Bits48UsingBusy": {Value: 0b11},
					},
				},
				"TransferModeIsMultiblock": {
					Description: `Type of data transfer (if 0, 
then the transfer is single block`,
					BitRange: sysdec.BitRange(5, 5),
					Access:   sysdec.Access("rw"),
				},
				"TransferModeDataDirection": {
					Description: `0 is host to card, 1 is card to host`,
					BitRange:    sysdec.BitRange(4, 4),
					Access:      sysdec.Access("rw"),
				},
				"EnableTransferModeAutoCommand": {
					Description: `Select the command to be send after 
completion of a data transfer:
00 = no command
01 = command CMD12
10 = command CMD23 
11 = reserved`,
					BitRange: sysdec.BitRange(3, 2),
					Access:   sysdec.Access("rw"),
					EnumeratedValue: map[string]*sysdec.EnumeratedValueDef{
						"NoCommand": {Value: 0b00},
						"Command12": {Value: 0b01},
						"Command23": {Value: 0b10},
					},
				},
				"EnableTransferModeBlockCounter": {
					Description: `Enable the block counter for 
multiple block transfers:`,
					BitRange: sysdec.BitRange(1, 1),
					Access:   sysdec.Access("rw"),
				},
			},
		},
		"Response0": {
			Description: `This register contains the status bits of the SD 
card s response. In case of commands CMD2 and CMD10 it contains 
CID[31:0] and in case of command CMD9 it contains CSD[31:0].

Note: this register is only valid once the last command has completed and 
no new command was issued.`,
			AddressOffset: 0x10,
			Size:          32,
			Access:        sysdec.Access("rw"),
		},
		"Response1": {
			Description: `In case of commands CMD2 and CMD10 this register 
contains CID[63:32] and in case of command CMD9 it contains CSD[63:32].

Note: this register is only valid once the last command has completed 
and no new command was issued.`,
			AddressOffset: 0x14,
			Size:          32,
			Access:        sysdec.Access("rw"),
		},
		"Response2": {
			Description: `In case of commands CMD2 and CMD10 this register 
contains CID[95:64] and in case of command CMD9 it contains CSD[95:64].

Note: this register is only valid once the last command has completed 
and no new command was issued.`,
			AddressOffset: 0x18,
			Size:          32,
			Access:        sysdec.Access("rw"),
		},
		"Response3": {
			Description: `In case of commands CMD2 and CMD10 this register 
contains CID[127:96] and in case of command CMD9 it contains CSD[127:96].

Note: this register is only valid once the last command has completed and 
no new command was issued.`,
			AddressOffset: 0x1C,
			Size:          32,
			Access:        sysdec.Access("rw"),
		},
		"Data": {
			Description: `This register is used to transfer data to/from 
the card.

Bit 1 of the INTERRUPT register can be used to check if data is available. 
For paced DMA transfers the high active signal dma_req can be used.`,
			AddressOffset: 0x20,
			Size:          32,
			Access:        sysdec.Access("rw"),
		},
		"DebugStatus": {
			Description: `This register contains information intended for 
debugging. Its values change automatically according to the hardware. As it 
involves resynchronisation between different clock domains it changes only 
after some latency and it is easy sample the values too early.

Therefore it is not recommended to use this register for polling. Instead 
use the INTERRUPT register which implements a handshake mechanism which makes 
it impossible to miss a change when polling.`,
			AddressOffset: 0x24,
			Size:          32,
			Access:        sysdec.Access("rw"),
		},
		"Control0": {
			Description: `This register is used to configure the EMMC module.
For the exact details please refer to the Arasan documentation 
SD3.0_Host_AHB_eMMC4.4_Usersguide_ver5.9_jan11_10.pdf. Bits marked as 
reserved in this document but not by the Arasan documentation refer to 
functionality which has been disabled due to the changes listed in the 
previous chapter.`,
			AddressOffset: 0x28,
			Size:          32,
			Access:        sysdec.Access("rw"),
			Field: map[string]*sysdec.FieldDef{
				"EnableAlternateBootAccess": {
					Description: `Enable alternate boot mode access`,
					BitRange:    sysdec.BitRange(22, 22),
				},
				"EnableBootModeAccess": {
					Description: `Enable boot mode access`,
					BitRange:    sysdec.BitRange(21, 21),
				},
				"EnableSPIMode": {
					Description: `Enable SPI mode on 1, otherwise normal mode`,
					BitRange:    sysdec.BitRange(20, 20),
				},
				"EnableGAPInterrupt": {
					Description: `Enable SDIO interrupt at block gap (only 
valid if the HCTL_DWIDTH bit is set)`,
					BitRange: sysdec.BitRange(19, 19),
				},
				"EnableReadWait": {
					Description: `Use DAT2 read-wait protocol for SDIO 
cards supporting this:`,
					BitRange: sysdec.BitRange(18, 18),
				},
				"GapRestart": {
					Description: `Restart a transaction which was stopped 
using the GAP_STOP bit: 0 disabled, 1 enabled`,
					BitRange: sysdec.BitRange(17, 17),
				},
				"GapStop": {
					Description: `Stop the current transaction at the next 
block gap: 0 = ignore 1 = stop`,
					BitRange: sysdec.BitRange(16, 16),
				},
				"HardwareControl8Bit": {
					Description: `Use 8 data lines`,
					BitRange:    sysdec.BitRange(5, 5),
				},
				"EnableHardwareControlHighSpeed": {
					Description: `Select high speed mode (i.e. DAT and 
CMD lines change on the rising CLK edge):`,
					BitRange: sysdec.BitRange(2, 2),
				},
				"HardwareControlDataWidth": {
					Description: `Use four data lines`,
					BitRange:    sysdec.BitRange(1, 1),
				},
			},
		},
		"Control1": {
			Description: `This register is used to configure the EMMC module.
CLK_STABLE seems contrary to its name only to indicate that there was a rising 
edge on the clk_emmc input but not that the frequency of this clock is 
actually stable.`,
			AddressOffset: 0x2C,
			Size:          32,
			Access:        sysdec.Access("rw"),
			Field: map[string]*sysdec.FieldDef{
				"ResetData": {
					Description: `Reset data handling circuit. 
1 enabled, 0 disabled.`,
					BitRange: sysdec.BitRange(26, 26),
				},
				"ResetCommand": {
					Description: `Reset the command handling circuit: 
0 = disabled 1 = enabled`,
					BitRange: sysdec.BitRange(25, 25),
				},
				"ResetHostCircuit": {
					Description: `Reset the complete host circuit: 
0 = disabled 1 = enabled`,
					BitRange: sysdec.BitRange(24, 24),
				},
				"DataTimeoutUnitExponent": {
					Description: `Data timeout unit exponent: 
1111 = disabled, x = TMCLK * 2^(x+13)`,
					BitRange: sysdec.BitRange(19, 16),
				},
				"ClockFrequencyDividerLSB": {
					Description: `SD clock base divider low 8 bits`,
					BitRange:    sysdec.BitRange(15, 8),
				},
				"ClockFrequencyDividerMSB": {
					Description: `SD clock base divider high 2 bits`,
					BitRange:    sysdec.BitRange(7, 6),
				},
				"ClockGenerationSelect": {
					Description: `Mode of clock generation: 0 = divided
1 = programmable`,
					BitRange: sysdec.BitRange(5, 5),
				},
				"EnableClock": {
					Description: `SD Clock Enable, 1 = enabled, 0 = disabled`,
					BitRange:    sysdec.BitRange(2, 2),
				},
				"ClockStable": {
					Description: `SD Clock Stable, 0 = no, 1 = yes.
Please check the documentation of this register for more info.`,
					BitRange: sysdec.BitRange(1, 1),
				},
				"ClockInternal": {
					Description: `Clock enable for internal EMMC clocks 
for power saving: 0 = disabled, 1 = enabled`,
					BitRange: sysdec.BitRange(0, 0),
				},
			},
		},
		"Interrupt": {
			Description: `This register holds the interrupt flags. Each 
flag can be disabled using the according bit in the IRPT_MASK register.
ERR is a generic flag and is set if any of the enabled error flags is set.`,
			AddressOffset: 0x30,
			Size:          25,
			Access:        sysdec.Access("rw"),
			Field: map[string]*sysdec.FieldDef{
				"AutoCommandError": {
					Description: `Auto command error: 0 = no error
1 = error`,
					BitRange: sysdec.BitRange(24, 24),
				},
				"DataEndBitNot1": {
					Description: `End bit on data line not 1: 0 = no error
1 = error`,
					BitRange: sysdec.BitRange(22, 22),
				},
				"DataCRCError": {
					Description: `Data CRC error: 0 = no error
1 = error`,
					BitRange: sysdec.BitRange(21, 21),
				},
				"DataTimeoutError": {
					Description: `Data Timeout error: 0 = no error
1 = error`,
					BitRange: sysdec.BitRange(20, 20),
				},
				"CommandBad": {
					Description: `Bad command index in response: 0 = no error
1 = error`,
					BitRange: sysdec.BitRange(19, 19),
				},
				"CommandEndBitNot1": {
					Description: `End bit on command line not 1: 0 = no error
1 = error`,
					BitRange: sysdec.BitRange(18, 18),
				},
				"CommandCRCError": {
					Description: `Command CRC error: 0 = no error
1 = error`,
					BitRange: sysdec.BitRange(17, 17),
				},
				"CommandTimeoutError": {
					Description: `Command timeout error: 0 = no error
1 = error`,
					BitRange: sysdec.BitRange(16, 16),
				},
				"Error": {
					Description: `An error has occurred [sic?]: 0 = no error
1 = error `,
					BitRange: sysdec.BitRange(15, 15),
				},
				"BootOperationTerminated": {
					Description: `A boot operation has terminated: 0 = no 
1 = yes `,
					BitRange: sysdec.BitRange(14, 14),
				},
				"BootACKReceived": {
					Description: `A boot Acknowleged has been received: 0 = no 
1 = yes `,
					BitRange: sysdec.BitRange(13, 13),
				},
				"ClockRetune": {
					Description: `A clock retune request was made: 0 = no 
1 = yes `,
					BitRange: sysdec.BitRange(12, 12),
				},
				"Card": {
					Description: `Card made interrupt request: 0 = no 
1 = yes `,
					BitRange: sysdec.BitRange(8, 8),
				},
				"ReadReady": {
					Description: `Data register contains data to be read: 
0 = no, 1 = yes `,
					BitRange: sysdec.BitRange(5, 5),
				},
				"WriteReady": {
					Description: `Data can be written to data register: 
0 = no, 1 = yes `,
					BitRange: sysdec.BitRange(5, 5),
				},
				"BlockGap": {
					Description: `Data transfer has stopped a block gap: 
0 = no, 1 = yes `,
					BitRange: sysdec.BitRange(2, 2),
				},
				"DataDone": {
					Description: `Data transfer has finished`,
					BitRange:    sysdec.BitRange(1, 1),
				},
				"CommandDone": {
					Description: `Command has finished`,
					BitRange:    sysdec.BitRange(0, 0),
				},
			},
		},
		"InterruptMask": {
			Description: `This register is used to mask the interrupt flags 
in the INTERRUPT register.`,
			AddressOffset: 0x34,
			Size:          25,
			Access:        sysdec.Access("rw"),
			Field: map[string]*sysdec.FieldDef{
				"AutoCommandError": {
					Description: `Set flag if auto command error: 0 = no 
1 = yes`,
					BitRange: sysdec.BitRange(24, 24),
				},
				"DataEndBitNot1": {
					Description: `Set flag if end bit on data line not 1: 
0 = no 1 = yes`,
					BitRange: sysdec.BitRange(22, 22),
				},
				"DataCRCError": {
					Description: `Set flag if data CRC error: 0 = no 
1 = yes`,
					BitRange: sysdec.BitRange(21, 21),
				},
				"DataTimeoutError": {
					Description: `Set flag on Data Timeout on Data line: 
0 = no 1 = yes`,
					BitRange: sysdec.BitRange(20, 20),
				},
				"CommandBad": {
					Description: `Set flag on bad command index in response: 
0 = no 1 = ye-`,
					BitRange: sysdec.BitRange(19, 19),
				},
				"CommandEndBitNot1": {
					Description: `Set flag on end bit on command line not 1: 0 = nor
1 = yes`,
					BitRange: sysdec.BitRange(18, 18),
				},
				"CommandCRCError": {
					Description: `Set flag on ommand CRC error: 0 = no 
1 = yes`,
					BitRange: sysdec.BitRange(17, 17),
				},
				"CommandTimeoutError": {
					Description: `Set flag if timeout on command line: 
0 = no  1 = yes`,
					BitRange: sysdec.BitRange(16, 16),
				},
				"BootOperationTerminated": {
					Description: `Set flag if boot operation has terminated: 
0 = no 1 = yes `,
					BitRange: sysdec.BitRange(14, 14),
				},
				"BootACKReceived": {
					Description: `Set flag if boot Acknowleged has been received: 0 = no 
1 = yes `,
					BitRange: sysdec.BitRange(13, 13),
				},
				"ClockRetune": {
					Description: `Set if a clock retune request was made: 0 = no 
1 = yes `,
					BitRange: sysdec.BitRange(12, 12),
				},
				"Card": {
					Description: `Set if card made interrupt request: 0 = no 
1 = yes `,
					BitRange: sysdec.BitRange(8, 8),
				},
				"ReadReady": {
					Description: `Data register contains data to be read: 
0 = no, 1 = yes `,
					BitRange: sysdec.BitRange(5, 5),
				},
				"WriteReady": {
					Description: `Data can be written to data register: 
0 = no, 1 = yes `,
					BitRange: sysdec.BitRange(5, 5),
				},
				"BlockGap": {
					Description: `Data transfer has stopped a block gap: 
0 = no, 1 = yes `,
					BitRange: sysdec.BitRange(2, 2),
				},
				"DataDone": {
					Description: `Data transfer has finished, 0 = no
1 = yes`,
					BitRange: sysdec.BitRange(1, 1),
				},
				"CommandDone": {
					Description: `Command has finished, 0 = no, 1 = yes`,
					BitRange:    sysdec.BitRange(0, 0),
				},
			},
		},
		"EnableInterrupt": {
			Description: `his register is used to enable the different 
interrupts in the INTERRUPT register to generate an interrupt on the i
nt_to_arm output.`,
			AddressOffset: 0x38,
			Size:          25,
			Access:        sysdec.Access("rw"),
			Field: map[string]*sysdec.FieldDef{
				"AutoCommandError": {
					Description: `Create interrupt if auto command error: 0 = no 
1 = yes`,
					BitRange: sysdec.BitRange(24, 24),
				},
				"DataEndBitNot1": {
					Description: `Create interrupt if end bit on data line 
not 1: 0 = no 1 = yes`,
					BitRange: sysdec.BitRange(22, 22),
				},
				"DataCRCError": {
					Description: `Create interrupt if data CRC error: 0 = no 
1 = yes`,
					BitRange: sysdec.BitRange(21, 21),
				},
				"DataTimeoutError": {
					Description: `Create interrupt Data Timeout on Data line: 
0 = no 1 = yes`,
					BitRange: sysdec.BitRange(20, 20),
				},
				"CommandBad": {
					Description: `Create interrupt if bad command index 
in response: 0 = no 1 = ye-`,
					BitRange: sysdec.BitRange(19, 19),
				},
				"CommandEndBitNot1": {
					Description: `Create interrupt on end bit on command 
line not 1: 0 = nor 1 = yes`,
					BitRange: sysdec.BitRange(18, 18),
				},
				"CommandCRCError": {
					Description: `Create interrupt on command CRC error: 0 = no 
1 = yes`,
					BitRange: sysdec.BitRange(17, 17),
				},
				"CommandTimeoutError": {
					Description: `Create interrupt if timeout on command line: 
0 = no  1 = yes`,
					BitRange: sysdec.BitRange(16, 16),
				},
				"BootOperationTerminated": {
					Description: `Create interrupt if boot operation 
has terminated: 0 = no 1 = yes `,
					BitRange: sysdec.BitRange(14, 14),
				},
				"BootACKReceived": {
					Description: `Create interrupt if boot Acknowleged has been 
received: 0 = no 1 = yes `,
					BitRange: sysdec.BitRange(13, 13),
				},
				"ClockRetune": {
					Description: `Create interrupt if a clock retune request 
was made: 0 = no 1 = yes `,
					BitRange: sysdec.BitRange(12, 12),
				},
				"Card": {
					Description: `Create interrupt if card made interrupt 
request: 0 = no 1 = yes `,
					BitRange: sysdec.BitRange(8, 8),
				},
				"ReadReady": {
					Description: `Create interrupt if data register contains 
data to be read: 0 = no, 1 = yes `,
					BitRange: sysdec.BitRange(5, 5),
				},
				"WriteReady": {
					Description: `Create an interrupt if data can be written 
to data register: 0 = no, 1 = yes `,
					BitRange: sysdec.BitRange(5, 5),
				},
				"BlockGap": {
					Description: `Create interrupt if data transfer has 
stopped a block gap: 0 = no, 1 = yes `,
					BitRange: sysdec.BitRange(2, 2),
				},
				"DataDone": {
					Description: `Create interrupt if data transfer has 
finished, 0 = no 1 = yes`,
					BitRange: sysdec.BitRange(1, 1),
				},
				"CommandDone": {
					Description: `Create interrupt if command has finished, 
0 = no, 1 = yes`,
					BitRange: sysdec.BitRange(0, 0),
				},
			},
		},
		"Control2": {
			Description: `This register is used to enable the different 
interrupts in the INTERRUPT register to generate an interrupt on the 
int_to_arm output.`,
			AddressOffset: 0x3C,
			Size:          24,
			Access:        sysdec.Access("r"),
			Field: map[string]*sysdec.FieldDef{
				"TunedClockIsUsed": {
					Description: `Tuned clock is used for sampling data.
0=no 1=yes`,
					BitRange: sysdec.BitRange(23, 23),
				},
				"ClockIsTuning": {
					Description: `Start tuning the SD clock: 0 not tuned,
or tuning complete, 1=tuning`,
					BitRange: sysdec.BitRange(22, 22),
				},
				"UHSMode": {
					Description: `Select the speed mode of the SD card: 
000 = SDR12
001 = SDR25
010 = SDR50
011 = SDR104 
100 = DDR50 
other = reserved`,
					BitRange: sysdec.BitRange(18, 16),
					EnumeratedValue: map[string]*sysdec.EnumeratedValueDef{
						"SDR12":  {Value: 0b000},
						"SDR25":  {Value: 0b001},
						"SDR50":  {Value: 0b010},
						"SDR104": {Value: 0b011},
						"DDR50":  {Value: 0b100},
					},
				},
				"NotC12Err": {
					Description: `Error occurred during auto cmd 12 execution
0=no error, 1=error`,
					BitRange: sysdec.BitRange(7, 7),
				},
				"ACBadErr": {
					Description: `Error occurred during auto command execution
0=no error, 1=error`,
					BitRange: sysdec.BitRange(4, 4),
				},
				"ACEndErr": {
					Description: `End bit is not 1 during auto command 
execution: 0 = no error 1 = error`,
					BitRange: sysdec.BitRange(3, 3),
				},
				"AutoCommandCRCError": {
					Description: `Command CRC error occurred during auto 
command execution: 0 = no error 1 = error`,
					BitRange: sysdec.BitRange(2, 2),
				},
				"AutoCommandTimeout": {
					Description: `Timeout occurred during auto 
command execution: 0 = no error 1 = error`,
					BitRange: sysdec.BitRange(1, 1),
				},
				"AutoCommandNotExecuted": {
					Description: `Auto command not executed due to err: 0 = no 
1 = yes`,
					BitRange: sysdec.BitRange(0, 0),
				},
			},
		},
		"ForceInterrupt": {
			Description: `This register is used to fake the different interrupt 
events for debugging.`,
			AddressOffset: 0x50,
			Size:          25,
			Access:        sysdec.Access("rw"),
			Field: map[string]*sysdec.FieldDef{
				"AutoCommandError": {
					Description: `Create auto command error: 0 = no
1 = yes`,
					BitRange: sysdec.BitRange(24, 24),
				},
				"DataEndNot1": {
					Description: `Create end bit on data line not 1: 0 = no
1 = yes`,
					BitRange: sysdec.BitRange(22, 22),
				},
				"DataCRC": {
					Description: `Create data CRC error: 0 = no
1 = yes`,
					BitRange: sysdec.BitRange(21, 21),
				},
				"DataTimeout": {
					Description: `Create timeout on data line: 0 = no
1 = yes`,
					BitRange: sysdec.BitRange(20, 20),
				},
				"CommandBad": {
					Description: `Create incorrect command index in response: 
0 = no 1 = yes`,
					BitRange: sysdec.BitRange(19, 19),
				},
				"CommandBitNotEnd1": {
					Description: `Create end bit on command line not 1: 0 = no
1 = yes`,
					BitRange: sysdec.BitRange(18, 18),
				},
				"CommandCRCError": {
					Description: `Create command CRC error: 0 = no
1 = yes`,
					BitRange: sysdec.BitRange(17, 17),
				},
				"CommandTimeout": {
					Description: `Create command timeout error: 0 = no
1 = yes`,
					BitRange: sysdec.BitRange(16, 16),
				},
				"BootOperationTerminated": {
					Description: `Create boot operation has terminated: 0 = no
1 = yes`,
					BitRange: sysdec.BitRange(14, 14),
				},
				"BootAcknowleged": {
					Description: `Create boot acknowlege has been received: 
0 = no 1 = yes`,
					BitRange: sysdec.BitRange(14, 14),
				},
				"Retune": {
					Description: `Create return request has been received: 
0 = no 1 = yes`,
					BitRange: sysdec.BitRange(12, 12),
				},
				"Card": {
					Description: `Create card has made interrupt request: 
0 = no 1 = yes`,
					BitRange: sysdec.BitRange(8, 8),
				},
				"ReadReady": {
					Description: `Create data register contains data to 
be read: 0 = no 1 = yes`,
					BitRange: sysdec.BitRange(5, 5),
				},
				"TransmitReady": {
					Description: `Create data can be written to data register: 
0 = no 1 = yes`,
					BitRange: sysdec.BitRange(4, 4),
				},
				"BlockGap": {
					Description: `Create interrupt if data transfer has
stopped a block gap: 0 = no 1 = yes`,
					BitRange: sysdec.BitRange(2, 2),
				},
				"DataDone": {
					Description: `Create data transfer has finished
0 = no 1 = yes`,
					BitRange: sysdec.BitRange(1, 1),
				},
				"CommandDone": {
					Description: `Create command has finished
0 = no 1 = yes`,
					BitRange: sysdec.BitRange(0, 0),
				},
			},
		},
		"BootTimeout": {
			Description:   `Timeout in boot mode`,
			AddressOffset: 0x70,
			Size:          32,
			Access:        sysdec.Access("rw"),
			Field: map[string]*sysdec.FieldDef{
				"Select": {
					Description: `Number of card clock cycles after which 
a timeout during boot mode is flagged`,
					Access:   sysdec.Access("rw"),
					BitRange: sysdec.BitRange(31, 0),
				},
			},
		},
		"DebugSelect": {
			Description: `This register selects which submodules are 
accessed by the debug bus.`,
			AddressOffset: 0x74,
			Size:          1,
			Access:        sysdec.Access("rw"),
			Field: map[string]*sysdec.FieldDef{
				"Select": {
					Description: `This register selects which submodules are 
accessed by the debug bus.`,
					Access:   sysdec.Access("rw"),
					BitRange: sysdec.BitRange(0, 0),
				},
			},
		},
		"EnableExentendedReadFIFO": {
			Description: `This register allows fine tuning the dma_req 
generation for paced DMA transfers when reading from the card. If the 
extension data FIFO contains less than RD_THRSH 32 bits words dma_req 
becomes inactive until the card has filled the extension data FIFO above 
threshold. This compensates the DMA latency.

When writing data to the card the extension data FIFO feeds into the EMMC 
module s FIFO and no fine tuning is required Therefore the RD_THRSH value 
is in this case ignored.`,
			AddressOffset: 0x80,
			Size:          1,
			Access:        sysdec.Access("rw"),
			Field: map[string]*sysdec.FieldDef{
				"Enable": {
					Description: `This register enables the extension data 
register. It should be enabled for paced DMA transfers and be bypassed for 
burst DMA transfers.`,
					Access:   sysdec.Access("rw"),
					BitRange: sysdec.BitRange(0, 0),
				},
			},
		},
		"EnableExtendedReadFIFO": {
			Description: `This register enables the extension data register. It 
should be enabled for paced DMA transfers and be bypassed for burst 
DMA transfers.`,
			AddressOffset: 0x84,
			Size:          1,
			Access:        sysdec.Access("rw"),
			Field: map[string]*sysdec.FieldDef{
				"Enable": {
					Description: `0=bypass extension FIFO, 1=enabled`,
					BitRange:    sysdec.BitRange(0, 0),
				},
			},
		},
		"TuneStep": {
			Description:   ``,
			AddressOffset: 0x88,
			Size:          3,
			Access:        sysdec.Access("rw"),
			Field: map[string]*sysdec.FieldDef{
				"Delay": {
					Description: `This register is used to delay the card 
clock when sampling the returning data and command response from the card.
DELAY determines by how much the sampling clock is delayed per step.`,
					BitRange: sysdec.BitRange(2, 0),
					EnumeratedValue: map[string]*sysdec.EnumeratedValueDef{
						"PS200": {
							Description: "200 picoseconds typically",
							Value:       0b000,
						},
						"PS400": {
							Description: "400 picoseconds typically",
							Value:       0b001,
						},
						"PS400A": {
							Description: "400 picoseconds typically [sic? " +
								"same as previous?]",
							Value: 0b010,
						},
						"PS600": {
							Description: "600 picoseconds typically",
							Value:       0b011,
						},
						"PS700": {
							Description: "700 picoseconds typically",
							Value:       0b100,
						},
						"PS900": {
							Description: "900 picoseconds typically",
							Value:       0b101,
						},
						"PS900A": {
							Description: "900 picoseconds typically [sic?" +
								"same as previous?]",
							Value: 0b110,
						},
						"PS1100A": {
							Description: "1100 picoseconds typically ",
							Value:       0b111,
						},
					},
				},
			},
		},
		"TuneStepsStandard": {
			Description: `This register is used to delay the card clock when 
sampling the returning data and command response from the card. It determines 
by how many steps the sampling clock is delayed in SDR mode.`,
			AddressOffset: 0x8C,
			Size:          6,
			Access:        sysdec.Access("rw"),
			Field: map[string]*sysdec.FieldDef{
				"Steps": {
					Access:      sysdec.Access("rw"),
					BitRange:    sysdec.BitRange(5, 0),
					Description: `Number of steps (0 to 40)`,
				},
			},
		},
		"TuneStepsDDR": {
			Description: `This register is used to delay the card clock
when sampling the returning data and command response from the card. It 
determines by how many steps the sampling clock is delayed in DDR mode.`,
			AddressOffset: 0x90,
			Size:          6,
			Access:        sysdec.Access("rw"),
			Field: map[string]*sysdec.FieldDef{
				"Steps": {
					Access:      sysdec.Access("rw"),
					BitRange:    sysdec.BitRange(5, 0),
					Description: `Number of steps (0 to 40)`,
				},
			},
		},
		"SPIInterrupt": {
			Description: `This register controls whether assertion of 
interrupts in SPI mode is possible independent of the card select line.`,
			AddressOffset: 0xf0,
			Size:          6,
			Access:        sysdec.Access("rw"),
			Field: map[string]*sysdec.FieldDef{
				"Select": {
					Access:      sysdec.Access("rw"),
					BitRange:    sysdec.BitRange(7, 0),
					Description: `Interrupt independent select line`,
				},
			},
		},
		"SlotISRVersion": {
			Description: `This register contains the version information 
and slot interrupt status.`,
			AddressOffset: 0xfc,
			Size:          32,
			Access:        sysdec.Access("rw"),
			Field: map[string]*sysdec.FieldDef{
				"VendorVersionNumber": {
					Access:      sysdec.Access("rw"),
					BitRange:    sysdec.BitRange(31, 24),
					Description: `Vendor version`,
				},
				"SDVersion": {
					Access:      sysdec.Access("rw"),
					BitRange:    sysdec.BitRange(23, 16),
					Description: `Host Controller specification version`,
				},
				"SlotStatus": {
					Access:   sysdec.Access("rw"),
					BitRange: sysdec.BitRange(7, 0),
					Description: `Logical OR of interrupt and wakeup signal 
for each slot`,
				},
			},
		},
	},
}
