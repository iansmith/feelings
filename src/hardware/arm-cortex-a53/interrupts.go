package arm_cortex_a53

import (
	"device/arm"
)

//////////////////////////////////////////////////////////////////
// ARM64 Exception Handlers
//////////////////////////////////////////////////////////////////
type exceptionHandler func(uint64, uint64, uint64)

//list of exceptions
var excptrs [16]exceptionHandler

// MaskDAIF sets the value of the four D-A-I-F interupt masking on the ARM
func MaskDAIF() {
	arm.Asm("msr    daifset, #0x2")
}

// UnmaskDAIF sets the value of the four D-A-I-F interupt masking on the ARM
func UnmaskDAIF() {
	arm.Asm("msr    daifclr, #0x2")
}

//go:extern vectors
var vectors uint64

//go:extern proc_hang
func proc_hang()

// Called to make sure all the interrupt machinery is in the right startup state.
//go:noinline
// func InitExceptionVector() {
// 	for i := 0; i < len(excptrs); i++ {
// 		excptrs[i] = unexpectedException
// 	}
// 	arm.Asm("adr    x0, vectors") // load VBAR_EL1 with exc vector
// 	arm.Asm("msr    vbar_el1, x0")
// 	MaskDAIF()
// 	bcm2835.InterruptController.DisableIRQs1.SetBits(bcm2835.AuxInterrupt | bcm2835.SystemTimerIRQ1)
// }
//
// func SetExceptionHandlerEl1hInterrupts(h exceptionHandler) {
// 	excptrs[5] = h
// }
//
// func SetExceptionHandlerEl1hSynchronous(h exceptionHandler) {
// 	excptrs[4] = h
// }
//
// // when an interrupt falls in the woods and nobody is around to hear it
// //go:noinline
// //go:export raw_exception_handler
// func rawExceptionHandler(t uint64, esr uint64, addr uint64) {
// 	excptrs[t](t, esr, addr)
// }
//
// //go:noinline
// func unexpectedException(t uint64, esr uint64, addr uint64) {
// 	print("Unexpected Exception: ")
// 	print(entryErrorMessages[t])
// 	print(", ESR ")
// 	print(esr)
// 	reason := esr >> 26
// 	reason &= 0x3f
// 	print(top6ToESRReason(reason))
// 	reason = esr >> 25
// 	reason &= 0x3f
// 	print(top6ToESRReason(reason))
//
// 	print(", ADDR ")
// 	print(addr)
// 	print("\n")
// 	for {
// 		arm.Asm("nop")
// 	}
// }

func top6ToESRReason(shifted uint64) string {
	switch shifted {
	case 0b000000:
		return ("[unknown reason]")
	case 0b000001:
		return ("[Trapped WFE or WFI execution]")
	case 0b000011:
		return ("[Trapped MCR or MRC access]") //coproc difference?
	case 0b000100:
		return ("[Trapped MCRR or MRRC access]")
	case 0b000101:
		return ("[Trapped MCR or MRC access]") //coproc difference?
	case 0b000110:
		return ("[Read or Write to debug register DBGDTRRXint/DBGDTRTXint]")
	case 0b000111:
		return ("[Access to SVE, Advanced SIMD or FP trapped]")
	case 0b001100:
		return ("[Trapped MRRC access]") //coproc?
	case 0b001101:
		return ("[Branch Target Exception]")
	case 0b001110:
		return ("[Illegal Execution State]")
	case 0b010001:
		return ("[SVC in AARCH32 State]")
	case 0b010101:
		return ("[SVC in AARCH64 State]")
	case 0b011000:
		return ("[Trapped MSR or MRS in AARCH64 State]")
	case 0b011001:
		return ("[Access to SVE functionality trapped]")
	case 0b100000:
		return ("[Instruction Abort from lower exception level]")
	case 0b100001:
		return ("[Instruction Abort from same exception level]")
	case 0b100010:
		return ("[PC Alignment fault]")
	case 0b100100:
		return ("[Data abort from lower exception level]")
	case 0b100101:
		return ("[Data abort from same exception level]")
	case 0b100110:
		return ("[SP Alignment fault]")
	case 0b101000:
		return ("[Trapped floating point exception from AARCH32]")
	case 0b101100:
		return ("[Trapped floating point exception from AARCH64]")
	case 0b101111:
		return ("[SError interrupt]")
	case 0b110000:
		return ("[Breakpoint exception from lower exception level]")
	case 0b110001:
		return ("[Breakpoint exception from same exception level]")
	case 0b110010:
		return ("[Software step exception from lower exception level]")
	case 0b110011:
		return ("[Software step exception from same exception level]")
	case 0b110100:
		return ("[Watchpoint exception from lower exception level]")
	case 0b110101:
		return ("[Watchpoint exception from same exception level]")
	case 0b111000:
		return ("[BKPT exception in AARCH32]")
	case 0b111100:
		return ("[BRK exception in AARCH64]")
	default:
		return ("[should never happen, unused code]")
	}
}

var entryErrorMessages = []string{
	"SYNC_INVALID_EL1t",
	"IRQ_INVALID_EL1t",
	"FIQ_INVALID_EL1t",
	"ERROR_INVALID_EL1T",

	"SYNC_INVALID_EL1h",
	"IRQ_INVALID_EL1h",
	"FIQ_INVALID_EL1h",
	"ERROR_INVALID_EL1h",

	"SYNC_INVALID_EL0_64",
	"IRQ_INVALID_EL0_64",
	"FIQ_INVALID_EL0_64",
	"ERROR_INVALID_EL0_64",

	"SYNC_INVALID_EL0_32",
	"IRQ_INVALID_EL0_32",
	"FIQ_INVALID_EL0_32",
	"ERROR_INVALID_EL0_32",
}
