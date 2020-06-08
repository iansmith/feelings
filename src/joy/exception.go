package joy

import (
	"device/arm"
	"fmt"
)

//export raw_exception_handler
func rawExceptionHandler(t uint64, esr uint64, addr uint64, el uint64, procId uint64) {
	if t != 5 {
		// this is in case we get some OTHER kind of exception
		fmt.Printf("raw exception handler:exception type %d and "+
			"esr %x with addr %x and EL=%d, ProcID=%x\n",
			t, esr, addr, el, procId)
	} else {
		fmt.Printf("interrupt type 5 received (likely a peripheral)!\n")
	}
	fmt.Printf("DEADLOOP\n")
	for {
		arm.Asm("nop")
	}

}
