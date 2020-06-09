package joy

import (
	"device/arm"
	"machine"

	"lib/trust"
)

//export raw_exception_handler
func rawExceptionHandler(t uint64, esr uint64, addr uint64, el uint64, procId uint64) {
	if t == 5 {
		if machine.QA7.LocalTimerControl.InterruptPendingIsSet() {
			trust.Debugf("No interrupt pending line is set for timer, exiting")
			return
		}
		machine.QA7.LocalTimerClearReload.SetClear() //ugh, nomenclature
		machine.QA7.LocalTimerClearReload.SetReload()
		timerTick()
		return
	}
	trust.Infof("raw exception handler:exception type %d and "+
		"esr %x with addr %x and EL=%d, ProcID=%x\n",
		t, esr, addr, el, procId)
	trust.Errorf("DEADLOOP!")
	for {
		arm.Asm("nop")
	}
}
