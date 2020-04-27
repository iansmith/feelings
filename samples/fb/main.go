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

	id, ok := videocore.BoardID()
	if ok == false {
		trust.Errorf("can't get board id\n")
		return
	}
	fmt.Printf("board id         : %016x\n", id)

	v, ok := videocore.FirmwareVersion()
	if ok == false {
		trust.Errorf("can't get firmware version id\n")
		return
	}
	fmt.Printf("firmware version : %08x\n", v)

	rev, ok := videocore.BoardRevision()
	if ok == false {
		trust.Errorf("can't get board revision id\n")
		return
	}
	fmt.Printf("board revision   : %08x %s\n", rev, rt.BoardRevisionDecode(fmt.Sprintf("%x", rev)))

	cr, ok := videocore.GetClockRate()
	if ok == false {
		rt.MiniUART.WriteString("can't get clock rate\n")
		return
	}
	fmt.Printf("clock rate       : %d hz\n", cr)

	info := videocore.SetFramebufferRes1024x768()
	if info == nil {
		rt.Abort("giving up")
	}
	lfbPrint(10, 5, "Hello World!", (*PCScreenFont)(unsafe.Pointer(&binary_font_psf_start)), info)
}

//go:extern _binary_font_psf_start
var binary_font_psf_start [0]byte

// display a string on screen
func lfbPrint(x uint32, y uint32, s string, font *PCScreenFont, info *videocore.FrameBufferInfo) {
	start := (*uint8)(unsafe.Pointer(&binary_font_psf_start))
	data := uintptr(unsafe.Pointer(start)) + uintptr(font.Headersize)
	// get our font

	for _, c := range s {
		if c > 127 {
			c = 0 //should we warn?
		}
		offset := uintptr(font.BytesPerGlyph * uint32(c))
		glyphAddr := data + offset
		//deal with the x,y location
		glyphPtr := ((*uint8)(unsafe.Pointer(glyphAddr)))
		// calculate the offset on screen
		offs := (y * font.Height * info.Pitch) + (x * (font.Width + 1) * 4)
		// variables
		bytesPerLine := (font.Width + 7) / 8
		// handle carrige return
		if c == '\r' {
			x = 0
		} else
		// new line
		if c == '\n' {
			x = 0
			y++
		} else {
			for j := uint32(0); j < font.Height; j++ {
				line := offs
				mask := uint32(1 << (font.Width - 1))

				var color uint32
				for i := uint32(0); i < font.Width; i++ {
					ptr := (*uint32)(unsafe.Pointer(uintptr(unsafe.Pointer(info.Buffer)) + uintptr(line)))
					if uint32(*glyphPtr)&mask != 0 {
						color = 0xFFFFFF //r, g, and b
					} else {
						color = 0
					}
					//fmt.Printf("line is %08x and mask is %08x\n", line, mask)
					*ptr = color
					mask >>= 1
					line += 4
				}
				// adjust to next line
				glyphAddr += uintptr(bytesPerLine)
				glyphPtr = ((*uint8)(unsafe.Pointer(glyphAddr)))
				offs += info.Pitch
			}
			x++
		}
	}
}
