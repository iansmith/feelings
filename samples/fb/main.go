package main

import (
	"feelings/src/golang/fmt"
	"feelings/src/hardware/videocore"
	"feelings/src/lib/trust"
	rt "feelings/src/tinygo_runtime"
	"unsafe"
)

//export raw_exception_handler
func raw_exception_handler() {
}

//go:extern _binary_font_psf_start
var binary_font_psf_start [0]byte

/* PC Screen Font as used by Linux Console */
type PCScreenFont struct {
	Magic         uint32
	Version       uint32
	Headersize    uint32
	Flags         uint32
	NumGlyphs     uint32
	BytesPerGlyph uint32
	Height        uint32
	Width         uint32
}

func main() {
	rt.MiniUART = rt.NewUART()
	_ = rt.MiniUART.Configure(rt.UARTConfig{ /*no interrupt*/ })
	trust.Errorf("hello bold big %s\n", "universe")

	var size, base uint32

	//info := videocore.SetFramebufferRes1920x1200()
	//if info == nil {
	//	rt.Abort("giving up")
	//}
	info := videocore.SetFramebufferRes1024x768()
	if info == nil {
		rt.Abort("giving up")
	}

	console := NewFBConsole(info, (*PCScreenFont)(unsafe.Pointer(&binary_font_psf_start)))
	logger := trust.NewLogger(console)

	id, ok := videocore.BoardID()
	if ok == false {
		trust.Errorf("can't get board id\n")
		return
	}
	logger.Infof("board id         : %016x\n", id)

	v, ok := videocore.FirmwareVersion()
	if ok == false {
		trust.Errorf("can't get firmware version id\n")
		return
	}
	logger.Infof("firmware version : %08x\n", v)

	rev, ok := videocore.BoardRevision()
	if ok == false {
		trust.Errorf("can't get board revision id\n")
		return
	}
	logger.Infof("board revision   : %08x %s\n", rev, rt.BoardRevisionDecode(fmt.Sprintf("%x", rev)))

	cr, ok := videocore.GetClockRate()
	if ok == false {
		rt.MiniUART.WriteString("can't get clock rate\n")
		return
	}
	logger.Infof("clock rate       : %d hz\n", cr)

	base, size, ok = videocore.GetARMMemoryAndBase()
	if ok == false {
		rt.MiniUART.WriteString("can't get arm memory\n")
		return
	}
	logger.Infof("ARM Memory       : 0x%x bytes @ 0x%x\n", size, base)

	base, size, ok = videocore.GetVCMemoryAndBase()
	if ok == false {
		rt.MiniUART.WriteString("can't get vc memory\n")
		return
	}
	logger.Infof("VidCore IV Memory: 0x%x bytes @ 0x%x\n", size, base)

}

//
//type bar interface {
//	baz(format string, params ...interface{})
//}
//
//type defaultBar struct {
//}
//
//func (d *defaultBar) baz(format string, params ...interface{}) {
//	fmt.Printf(format, params...)
//}
//
//type foo struct {
//	myBar bar
//}
//
//var defaultFoo = foo{&defaultBar{}}
//
//func (f *foo) fleazil() {
//	f.myBar.baz("testing %d %d %d...\n", 1, 2, 3)
//}
//
//func main() {
//	rt.MiniUART = rt.NewUART()
//	_ = rt.MiniUART.Configure(rt.UARTConfig{ /*no interrupt*/ })
//
//	fmt.Printf("START\n")
//	defaultFoo.fleazil()
//	fmt.Printf("END\n")
//}
