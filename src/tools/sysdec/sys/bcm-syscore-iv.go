package sys

import "tools/sysdec"

var GPUMailbox = &sysdec.PeripheralDef{
	Version: 1,
	Description: `
This peripheral really is running the show. It's running its own OS and bosses
the ARM around.

https://github.com/raspberrypi/firmware/wiki/Mailbox-property-interface
`,
	AddressBlock: sysdec.AddressBlockDef{BaseAddress: 0xB880, Size: 0x20},
	Register: map[string]*sysdec.RegisterDef{
		"Receive": {
			Description:   ``,
			Access:        sysdec.Access("r"),
			AddressOffset: 0x0,
			Size:          32,
		},
		"Poll": {
			Description:   ``,
			Access:        sysdec.Access("r"),
			AddressOffset: 0x10,
			Size:          32,
		},
		"Sender": {
			Description:   ``,
			Access:        sysdec.Access("r"),
			AddressOffset: 0x14,
			Size:          32,
		},
		"Status": {
			Description:   ``,
			Access:        sysdec.Access("r"),
			AddressOffset: 0x18,
			Size:          32,
			Field: map[string]*sysdec.FieldDef{
				"Full": {
					Description: `Is the mailbox already full?`,
					BitRange:    sysdec.BitRange(31, 31),
				},
				"Empty": {
					Description: `Is the mailbox empty?`,
					BitRange:    sysdec.BitRange(30, 30),
				},
			},
		},
		"Config": {
			Description:   ``,
			AddressOffset: 0x1C,
			Size:          32,
			Access:        sysdec.Access("rw"),
		},
		"Write": {
			Description:   ``,
			AddressOffset: 0x20,
			Size:          32,
			Access:        sysdec.Access("w"),
		},
	},
}
