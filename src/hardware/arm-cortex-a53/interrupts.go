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

// InitInterrupts is called by postinit (before main) to make sure all the interrupt
// machinery is in the right startup state.
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

//go:noinline
func unexpectedException(t uint64, esr uint64, addr uint64) {
	bcm2835.MiniUART.WriteString("Unexpected Exception: ")
	bcm2835.MiniUART.WriteString(entryErrorMessages[t])
	bcm2835.MiniUART.WriteString(", ESR 0x")
	bcm2835.MiniUART.Hex64string(esr)
	bcm2835.MiniUART.WriteString(", ADDR 0x")
	bcm2835.MiniUART.Hex64string(addr)
	bcm2835.MiniUART.WriteCR()
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
