package sys

import "tools/sysdec"

//
// This is the parent of the main peripherals, but he doesn't have
// anything that is directly addressable.
//

var BCM2837 = &sysdec.PeripheralDef{}
