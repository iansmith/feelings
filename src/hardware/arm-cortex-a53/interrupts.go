package arm_cortex_a53

import (
	"feelings/src/hardware/bcm2835"
	"github.com/tinygo-org/tinygo/src/device/arm"
)

//////////////////////////////////////////////////////////////////
// ARM64 Exception Handlers
//////////////////////////////////////////////////////////////////
type exceptionHandler func(uint64, uint64, uint64)

//list of exceptions
var excptrs [16]exceptionHandler

// MaskDAIF sets the value of the four D-A-I-F interupt masking on the ARM
func MaskDAIF() {
	arm.Asm("msr    daifset, #0xf")
}

// UnmaskDAIF sets the value of the four D-A-I-F interupt masking on the ARM
func UnmaskDAIF() {
	arm.Asm("msr    daifclr, #0xf")
}

//go:extern vectors
var vectors uint64

//go:extern proc_hang
func proc_hang()

// Called to make sure all the interrupt machinery is in the right startup state.
func InitInterrupts() {
	for i := 0; i < len(excptrs); i++ {
		excptrs[i] = unexpectedException
	}
	arm.Asm("adr    x0, vectors") // load VBAR_EL1 with exc vector
	arm.Asm("msr    vbar_el1, x0")
	MaskDAIF()
	bcm2835.InterruptController.DisableIRQs1.SetBits(bcm2835.AuxInterrupt)
}

func SetExceptionHandlerEl1hInterrupts(h exceptionHandler) {
	excptrs[5] = h
}

func SetExceptionHandlerEl1hSynchronous(h exceptionHandler) {
	excptrs[4] = h
}

// when an interrupt falls in the woods and nobody is around to hear it
//go:export raw_exception_handler
func rawExceptionHandler(t uint64, esr uint64, addr uint64) {
	excptrs[t](t, esr, addr)
}

func unexpectedException(t uint64, esr uint64, addr uint64) {
	print("Unexpected Exception: ")
	print(entryErrorMessages[t])
	print(", ESR ")
	print(esr)
	reason := esr >> 26
	reason &= 0x3f
	switch reason {
	case 0b000000:
		print("[unknown reason]")
	case 0b000001:
		print("[Trapped WFE or WFI execution]")
	case 0b000011:
		print("[Trapped MCR or MRC access]") //coproc difference?
	case 0b000100:
		print("[Trapped MCRR or MRRC access]")
	case 0b000101:
		print("[Trapped MCR or MRC access]") //coproc difference?
	case 0b000110:
		print("[Read or Write to debug register DBGDTRRXint/DBGDTRTXint]")
	case 0b000111:
		print("[Access to SVE, Advanced SIMD or FP trapped]")
	case 0b001100:
		print("[Trapped MRRC access]") //coproc?
	case 0b001101:
		print("[Branch Target Exception]")
	case 0b001110:
		print("[Illegal Execution State]")
	case 0b010001:
		print("[SVC in AARCH32 State]")
	case 0b010101:
		print("[SVC in AARCH64 State]")
	case 0b011000:
		print("[Trapped MSR or MRS in AARCH64 State]")
	case 0b011001:
		print("[Access to SVE functionality trapped]")
	case 0b100000:
		print("[Instruction Abort from lower exception level]")
	case 0b100001:
		print("[Instruction Abort from same exception level]")
	case 0b100010:
		print("[PC Alignment fault]")
	case 0b100100:
		print("[Data abort from lower exception level]")
	case 0b100101:
		print("[Data abort from same exception level]")
	case 0b100110:
		print("[SP Alignment fault]")
	case 0b101000:
		print("[Trapped floating point exception from AARCH32]")
	case 0b101100:
		print("[Trapped floating point exception from AARCH64]")
	case 0b101111:
		print("[SError interrupt]")
	case 0b110000:
		print("[Breakpoint exception from lower exception level]")
	case 0b110001:
		print("[Breakpoint exception from same exception level]")
	case 0b110010:
		print("[Software step exception from lower exception level]")
	case 0b110011:
		print("[Software step exception from same exception level]")
	case 0b110100:
		print("[Watchpoint exception from lower exception level]")
	case 0b110101:
		print("[Watchpoint exception from same exception level]")
	case 0b111000:
		print("[BKPT exception in AARCH32]")
	case 0b111100:
		print("[BRK exception in AARCH64]")
	default:
		print("[should never happen, unused code]")
	}
	print(", ADDR ")
	print(addr)
	print("\n")
	for {
		arm.Asm("nop")
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
