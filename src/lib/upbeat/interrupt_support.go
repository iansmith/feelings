package upbeat

import (
	"device/arm"
	"lib/trust"
)

func BoardRevisionDecode(s string) string {
	switch s {
	case "9020e0":
		return "3A+, Revision 1.0, 512MB, Sony UK"
	case "a02082":
		return "3B, Revision 1.2, 1GB, Sony UK"
	case "a020d3":
		return "3B+, Revision 1.3, 1GB, Sony UK"
	case "a22082":
		return "3B, Revision 1.2, 1GB, Embest"
	case "a220a0":
		return "CM3, Revision 1.0, 1GB, Embest"
	case "a32082":
		return "3B, Revision 1.2, 1GB, Sony Japan"
	case "a52082":
		return "3B, Revision 1.2, 1GB, Stadium"
	case "a22083":
		return "3B, Revision 1.3, 1GB, Embest"
	case "a02100":
		return "CM3+, Revision 1.0, 1GB, Sony UK"
	case "a03111":
		return "4B, Revision 1.1, 2GB, Sony UK"
	case "b03111":
		return "4B, Revision 1.2, 2GB, Sony UK"
	case "b03112":
		return "4B, Revision 1.2, 2GB, Sony UK"
	case "c03111":
		return "4B, Revision 1.1, 4GB, Sony UK"
	case "c03112":
		return "4B, Revision 1.2, 4GB, Sony UK"
	}
	return "unknown board"
}

func PrintoutException(esr uint64, c trust.Logger) {
	exceptionClass := esr >> 26
	switch exceptionClass {
	case 0:
		c.Errorf("unknown exception (%d)", exceptionClass)
	case 1:
		c.Errorf("trapped WFE or WFI instruction")
	case 2, 8, 9, 10, 11, 15, 16, 18, 19, 20, 22, 23, 26, 27, 28, 29, 30, 31, 35, 38, 39, 41, 42, 43, 45, 46, 54, 55, 57, 58, 59, 61, 62, 63:
		c.Errorf("unused exception code, should never happen (%d", exceptionClass)
	case 3:
		c.Errorf("trapped MRRC or MCRR access")
	case 4:
		c.Errorf("trapped MRRC or MCRR access")
	case 5:
		c.Errorf("trapped MRC or MCR access")
	case 6:
		c.Errorf("trapped LDC or STC access")
	case 7:
		c.Errorf("access to SVE, advanced SIMD or FP functionality")
	case 12:
		c.Errorf("trapped to MRRC access")
	case 13:
		c.Errorf("branch target exception")
	case 14:
		c.Errorf("illegal execution state")
	case 17:
		c.Errorf("SVC instruction in AARCH32")
		c.Errorf("[", esr&0xffff, "]")
	case 21:
		c.Errorf("SVC instruction in AARCH64")
		c.Errorf("[", esr&0xffff, "]")
	case 24:
		c.Errorf("trapped MRS, MSR or System instruction in AARCH64")
	case 25:
		c.Errorf("access to SVE functionality")
	case 32:
		c.Errorf("instruction abort from lower exception level")
	case 33:
		c.Errorf("instruction abort from same exception level")
	case 34:
		c.Errorf("PC alignment fault")
	case 36:
		c.Errorf("data abort from lower exception level")
	case 37:
		c.Errorf("data abort from same exception level")
	case 40:
		c.Errorf("trapped floating point exception from AARCH32")
	case 44:
		c.Errorf("trapped floating point exception from AARCH64")
	case 47:
		c.Errorf("SError exception")
	case 48:
		c.Errorf("Breakpoint from lower exception level")
	case 49:
		c.Errorf("Breakpoint from same exception level")
	case 50:
		c.Errorf("Software step from lower exception level")
	case 51:
		c.Errorf("Software step from same exception level")
	case 52:
		c.Errorf("Watchpoint from lower exception level")
	case 53:
		c.Errorf("Watchpoint from same exception level")
	case 56:
		c.Errorf("BKPT from AARCH32")
	case 60:
		c.Errorf("BRK from AARCH64")
	}

}

// MaskDAIF sets the value of the four D-A-I-F interupt masking on the ARM
func MaskDAIF() {
	arm.Asm("msr    daifset, #0x3") // IRQ + FIQ
}

// UnmaskDAIF sets the value of the four D-A-I-F interupt masking on the ARM
func UnmaskDAIF() {
	arm.Asm("msr    daifclr, #0x3") // IRQ + FIQ
}
