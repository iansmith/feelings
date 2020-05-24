package sys

import "tools/sysdec"

var GPIO = &sysdec.PeripheralDef{
	Version: 1,
	Description: `There are 54 general-purpose I/O (GPIO) lines split into 
two banks. All GPIO pins have at least two alternative functions within BCM. 
The alternate functions are usually peripheral IO and a single peripheral may 
appear in each bank to allow flexibility on the choice of IO voltage.

Note: Most users will want to use the function GPIOSetup rather than setting
or clearing the function select registers and then manipulating the Pull-Up/Down
Register and the associated clocks. GPIOSetup allows you to choose the
function for a particular pin and it handles these operations for you.
`,
	AddressBlock: sysdec.AddressBlockDef{BaseAddress: 0x20_0000, Size: 0x9C},
	Register: map[string]*sysdec.RegisterDef{
		"FSel[%s]": {
			Description: `The function select registers are used to define 
the operation of the general-purpose I/O pins. Each of the 54 GPIO pins has 
at least two alternative functions as defined in section 16.2. The FSEL{n} 
field determines the functionality of the nth GPIO pin. All unused alternative 
function lines are tied to ground and will output a “0” if selected. All 
pins reset to normal GPIO input operation.`,
			Dim:           6,
			DimIncrement:  4,
			AddressOffset: 0x0,
			Size:          32,
			Access:        sysdec.Access("rw"),
		},
		"GPSet[%s]": {
			Description: `The output set registers are used to set a GPIO pin. 
The SET{n} field defines the respective GPIO pin to set, writing a “0” to the 
field has no effect. If the GPIO pin is being used as in input (by default) 
then the value in the SET{n} field is ignored. However, if the pin is 
subsequently defined as an output then the bit will be set according to the 
last set/clear operation. Separating the set and clear functions removes the 
need for read-modify-write operations.`,
			Dim:           2,
			DimIncrement:  4,
			Size:          32,
			AddressOffset: 0x1C,
			Access:        sysdec.Access("w"),
		},
		"GPClr[%s]": {
			Description: `The output clear registers are used to clear a GPIO 
pin. The CLR{n} field defines the respective GPIO pin to clear, writing a 
“0” to the field has no effect. If the GPIO pin is being used as in input 
(by default) then the value in the CLR{n} field is ignored. However, if the 
pin is subsequently defined as an output then the bit will be set 
according to the last set/clear operation. Separating the set and clear 
functions removes the need for read-modify-write operations.`,
			Dim:           2,
			DimIncrement:  4,
			Size:          32,
			AddressOffset: 0x28,
			Access:        sysdec.Access("w"),
		},
		"GPLev[%s]": {
			Description: `The pin level registers return the actual 
value of the pin. The LEV{n} field gives the value of the respective GPIO
pin.`,
			Dim:           2,
			DimIncrement:  4,
			Size:          32,
			AddressOffset: 0x34,
			Access:        sysdec.Access("r"),
		},
		"GPPED[%s]": {
			Description: `The event detect status registers are used to record
level and edge events on the GPIO pins. The relevant bit in the event
detect status registers is set whenever: 1) an edge is detected that matches
the type of edge programmed in the rising/falling edge detect enable registers,
or 2) a level is detected that matches the type of level programmed in the
high/low level detect enable registers. The bit is cleared by writing a “1”
to the relevant bit.

The interrupt controller can be programmed to interrupt the processor when
any of the status bits are set. The GPIO peripheral has three dedicated
interrupt lines. Each GPIO bank can generate an independent interrupt. The
third line generates a single interrupt whenever any bit is set.`,
			Dim:           2,
			DimIncrement:  4,
			Size:          32,
			AddressOffset: 0x40,
			Access:        sysdec.Access("rw"),
		},
		"GPRE[%s]": {
			Description: `The rising edge detect enable registers define 
the pins for which a rising edge transition sets a bit in the event detect 
status registers (GPEDSn). When the relevant bits are set in both the GPRENn 
and GPFENn registers, any transition (1 to 0 and 0 to 1) will set a bit in 
the GPEDSn registers. The GPRENn registers use synchronous edge detection. 
This means the input signal is sampled using the system clock and then it 
is looking for a “011” pattern on the sampled signal. This has the effect 
of suppressing glitches.`,
			Dim:           2,
			DimIncrement:  4,
			Size:          32,
			AddressOffset: 0x4C,
			Access:        sysdec.Access("rw"),
		},
		"GPFE[%s]": {
			Description: `The falling edge detect enable registers define 
the pins for which a falling edge transition sets a bit in the event detect 
status registers (GPEDSn). When the relevant bits are set in both the GPRENn 
and GPFENn registers, any transition (1 to 0 and 0 to 1) will set a bit in 
the GPEDSn registers. The GPFENn registers use synchronous edge detection. 
This means the input signal is sampled using the system clock and then it is 
looking for a “100” pattern on the sampled signal. This has the effect of 
suppressing glitches.`,
			Dim:           2,
			DimIncrement:  4,
			Size:          32,
			AddressOffset: 0x58,
			Access:        sysdec.Access("rw"),
		},
		"GPHE[%s]": {
			Description: `The high level detect enable registers define 
the pins for which a high level sets a bit in the event detect status register 
(GPEDSn). If the pin is still high when an attempt is made to clear the status 
bit in GPEDSn then the status bit will remain set.`,
			Dim:           2,
			Size:          32,
			DimIncrement:  4,
			AddressOffset: 0x64,
			Access:        sysdec.Access("rw"),
		},
		"GPLEn[%s]": {
			Description: `The low level detect enable registers define 
the pins for which a low level sets a bit in the event detect status 
register (GPEDSn). If the pin is still low when an attempt is made to 
clear the status bit in GPEDSn then the status bit will remain set.`,
			Dim:           2,
			DimIncrement:  4,
			Size:          32,
			AddressOffset: 0x70,
			Access:        sysdec.Access("rw"),
		},
		"GPARE[%s]": {
			Description: `The asynchronous rising edge detect enable 
registers define the pins for which a asynchronous rising edge transition 
sets a bit in the event detect status registers (GPEDSn).

Asynchronous means the incoming signal is not sampled by the system clock. 
As such rising edges of very short duration can be detected.`,
			Dim:           2,
			DimIncrement:  4,
			Size:          32,
			AddressOffset: 0x7C,
			Access:        sysdec.Access("rw"),
		},
		"GPAFE[%s]": {
			Description: `The asynchronous falling edge detect enable 
registers define the pins for which a asynchronous falling edge transition 
sets a bit in the event detect status registers (GPEDSn). Asynchronous 
means the incoming signal is not sampled by the system clock. As such falling 
edges of very short duration can be detected.`,
			Dim:           2,
			DimIncrement:  4,
			Size:          32,
			AddressOffset: 0x88,
			Access:        sysdec.Access("rw"),
		},
		"GPPUD": {
			Description: `The GPIO Pull-up/down Register controls the 
actuation of the internal pull-up/down control line to ALL the GPIO pins. 
This register must be used in conjunction with the 2 GPPUDCLKn registers.

Note that it is not possible to read back the current Pull-up/down settings 
and so it is the users’ responsibility to ‘remember’ which pull-up/downs are 
active. The reason for this is that GPIO pull-ups are maintained even in 
power-down mode when the core is off, when all register contents is lost.

The Alternate function table also has the pull state which is applied after
a power down.`,
			Size:          32,
			AddressOffset: 0x94,
			Access:        sysdec.Access("rw"),
		},
		"GPUDClk[%s]": {
			Description: `The GPIO Pull-up/down Clock Registers 
control the actuation of internal pull-downs on the respective GPIO pins. 
These registers must be used in conjunction with the GPPUD register to effect 
GPIO Pull-up/down changes. The following sequence of events is required:

1. Write to GPPUD to set the required control signal (i.e. Pull-up or 
Pull-Down or neither to remove the current Pull-up/down)

2. Wait 150 cycles – this provides the required set-up time for the 
control signal

3. Write to GPPUDCLK0/1 to clock the control signal into the GPIO pads 
you wish to modify – NOTE only the pads which receive a clock will be modified, 
all others will retain their previous state.

4. Wait 150 cycles – this provides the required hold time for the 
control signal

5. Write to GPPUD to remove the control signal

6. Write to GPPUDCLK0/1 to remove the clock`,
			Dim:           2,
			DimIncrement:  4,
			Size:          32,
			AddressOffset: 0x98,
			Access:        sysdec.Access("rw"),
		},
	},
}
