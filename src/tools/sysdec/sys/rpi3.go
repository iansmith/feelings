// +build feelings_rpi3

package sys

import "tools/sysdec"

var RPI3 = sysdec.DeviceDef{
	Vendor:      "Raspberry Pi Foundation",
	VendorID:    "RPI",
	Name:        "rpi3b",
	Series:      "raspberry pi",
	Version:     1, //version of this doc
	Description: "Single Board Computer With An ARM A-53 Quad Core CPU in a BCM2837 SOC",
	//license for this doc
	LicenseText: `
	MIT License

	Copyright (c) 2020 Ian Smith

	Permission is hereby granted, free of charge, to any person obtaining a copy
	of this software and associated documentation files (the "Software"), to 
	deal in the Software without restriction, including without limitation the 
	rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
	sell copies of the Software, and to permit persons to whom the Software is
	furnished to do so, subject to the following conditions:

	The above copyright notice and this permission notice shall be included in 
	all copies or substantial portions of the Software.

	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
	IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
	FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
	AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
	LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
	OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
	SOFTWARE.`,
	Cpu: sysdec.CPUDef{
		Name:                "CA53", //arm name
		Description:         "ARM Cortex A-53",
		Revision:            "r3p0", //arm revision scheme
		LittleEndian:        true,
		MMUPresent:          true,
		FPUPresent:          true,
		DSPPresent:          false,
		ICachePresent:       true,
		DCachePresent:       true,
		DeviceNumInterrupts: 2 /*FIQ+IRQ*/ * (64 /*GPU*/ + 16 /*ARM Chip*/),
	},
	Peripheral: map[string]*sysdec.PeripheralDef{
		"SOC":         BCM2837, //just for completeness
		"IC":          IC,
		"Aux":         Aux,
		"QA7":         QA7,
		"GPUMailbox":  GPUMailbox,
		"GPIO":        GPIO,
		"SystemTimer": SystemTimer,
		"EMMC":        EMMC,
	},
	NumCores: 4,
	MMIOBindings: map[string]int{
		"IC":          0x3f00_0000,
		"Aux":         0x3f00_0000,
		"SystemTimer": 0x3f00_0000,
		"QA7":         0x4000_0000,
		"GPUMailbox":  0x3f00_0000,
		"GPIO":        0x3f00_0000,
		"EMMC":        0x3f00_0000,
	},
}
