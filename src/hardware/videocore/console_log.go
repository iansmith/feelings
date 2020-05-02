package videocore

import (
	"feelings/src/golang/fmt"
	"feelings/src/lib/trust"
	rt "feelings/src/tinygo_runtime"
	"unsafe"
)

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

func NewConsoleLogger() *trust.Logger {

	//info := SetFramebufferRes1920x1200()
	//if info == nil {
	//	rt.Abort("giving up")
	//}
	info := SetFramebufferRes1024x768()
	if info == nil {
		rt.Abort("can't set the screen resolution")
	}

	console := NewFBConsole(info, (*PCScreenFont)(unsafe.Pointer(&binary_font_psf_start)))
	return trust.NewLogger(console)

}

type FBConsole struct {
	info     *FrameBufferInfo
	font     *PCScreenFont
	virtualY uint32
	currentX uint32
	currentY uint32
	maxX     uint32
	maxY     uint32
}

// display a string on screen
func (f *FBConsole) print(s string) {
	start := (*uint8)(unsafe.Pointer(&binary_font_psf_start))
	data := uintptr(unsafe.Pointer(start)) + uintptr(f.font.Headersize)
	// get our font

	for _, c := range s {
		if c > 127 {
			c = 0 //should we warn?
		}
		offset := uintptr(f.font.BytesPerGlyph * uint32(c))
		glyphAddr := data + offset
		//deal with the x,y location
		glyphPtr := ((*uint8)(unsafe.Pointer(glyphAddr)))
		// calculate the offset on screen
		offs := (f.currentY * f.font.Height * f.info.Pitch) + (f.currentX * (f.font.Width + 1) * 4)
		// variables
		bytesPerLine := (f.font.Width + 7) / 8
		// handle carrige return
		if c == '\r' {
			//ignored
		} else
		// new line
		if c == '\n' {
			for clr := f.currentX; clr < f.maxX; clr++ {
				offs := (f.currentY * f.font.Height * f.info.Pitch) + (clr * (f.font.Width + 1) * 4)
				for j := uint32(0); j < f.font.Height; j++ {
					line := offs
					for i := uint32(0); i < f.font.Width; i++ {
						ptr := (*uint32)(unsafe.Pointer(uintptr(unsafe.Pointer(f.info.Buffer)) + uintptr(line)))
						*ptr = 0
						line += 4
					}
					offs += f.info.Pitch
				}
			}
			f.currentX = 0
			f.incrementY()
		} else {
			if f.currentX < f.maxX { //don't bother drawing stuff too far right
				for j := uint32(0); j < f.font.Height; j++ {
					line := offs
					mask := uint32(1 << (f.font.Width - 1))
					var color uint32
					for i := uint32(0); i < f.font.Width; i++ {
						ptr := (*uint32)(unsafe.Pointer(uintptr(unsafe.Pointer(f.info.Buffer)) + uintptr(line)))
						if uint32(*glyphPtr)&mask != 0 {
							color = 0xFFFFFF //r, g, and b
						} else {
							color = 0
						}
						*ptr = color
						mask >>= 1
						line += 4
					}
					// adjust to next line
					glyphAddr += uintptr(bytesPerLine)
					glyphPtr = ((*uint8)(unsafe.Pointer(glyphAddr)))
					offs += f.info.Pitch
				}
			}
			f.currentX++
		}
	}
}

// NewFBConsole allows you to init the framebuffer, but it's not probably what you want.
// NewConsoleLogger is probably better.
func NewFBConsole(data *FrameBufferInfo, font *PCScreenFont) *FBConsole {
	return &FBConsole{info: data, font: font,
		maxX: data.Width / (font.Width + 1),
		maxY: (data.Height / font.Height),
	}
}

func (f *FBConsole) Printf(format string, params ...interface{}) {
	f.print(fmt.Sprintf(format, params...))
}
func (f *FBConsole) incrementY() {
	f.currentY++
	if f.currentY == f.maxY {
		//if !SetVirtualOffset(0, 100*(f.virtualY+f.font.Height)) {
		//	panic("cant scroll")
		//}
		for i := uint32(1); i < f.maxY; i++ {
			for j := uint32(0); j < f.maxX; j++ {
				//// calculate the offset on screen
				dest := ((i - 1) * f.font.Height * f.info.Pitch) + (j * (f.font.Width + 1) * 4)
				src := (i * f.font.Height * f.info.Pitch) + (j * (f.font.Width + 1) * 4)
				for k := uint32(0); k < f.font.Height; k++ {
					for l := uint32(0); l < f.font.Width; l++ {
						destPtr := (*uint32)(unsafe.Pointer(uintptr(unsafe.Pointer(f.info.Buffer)) + uintptr(dest+(l*4))))
						srcPtr := (*uint32)(unsafe.Pointer(uintptr(unsafe.Pointer(f.info.Buffer)) + uintptr(src+(l*4))))
						*destPtr = 0
						*destPtr = *srcPtr
						if i == f.maxY-1 {
							*srcPtr = 0
						}
					}
					// adjust to next line
					dest += f.info.Pitch
					src += f.info.Pitch
				}
			}
		}
		f.currentY--
		//fmt.Printf("new scroll value %d plus %d\n", f.virtualY, f.currentY)
	}
	if f.currentY == f.maxY {
		panic("at end")
		// lift the screen
	}
}
