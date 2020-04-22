package main

import (
	"feelings/src/hardware/videocore"
	rt "feelings/src/tinygo_runtime"
)

//export raw_exception_handler
func raw_exception_handler() {

}

func main() {
	rt.MiniUART = rt.NewUART()
	_ = rt.MiniUART.Configure(rt.UARTConfig{ /*no interrupt*/ })

	id, ok := videocore.BoardID()
	if ok == false {
		rt.MiniUART.WriteString("err getting board id\n")
		return
	}
	rt.MiniUART.Hex64string(id)

	v, ok := videocore.FirmwareVersion()
	if ok == false {
		rt.MiniUART.WriteString("err getting firmware version id\n")
		return
	}
	rt.MiniUART.Hex32string(v)

	rev, ok := videocore.BoardRevision()
	if ok == false {
		rt.MiniUART.WriteString("err getting board revision id\n")
		return
	}
	rt.MiniUART.Hex32string(rev)

	cr, ok := videocore.GetClockRate()
	if ok == false {
		rt.MiniUART.WriteString("err getting clock rate\n")
		return
	}
	rt.MiniUART.Hex32string(cr)

}
