package main

import (
	"fmt"
	"unsafe"

	"boot/anticipation"
)

const resetIncrement = 0xff00 //a little less than 64K

///////////////////////////////////////////////////////////////////////////////
// emitter can take a blob of data and emit the necessary commands to transmit
// that memory to the other side. It uses a ioProto to do the actual IO work
// but it computes the lines and addresses to send.
///////////////////////////////////////////////////////////////////////////////
type emitter interface {
	line() (string, error)
	moreLines() bool //true if no more lines
	reset()          //return to beginning of sect
	sectionName() string
	read([]uint8) (string, error)
	receiver() ioProto
	nextAddr() uint64
}

// loadableSectionEmitter works from the blob of data in a loadable section of
// an elf format binary
type loadableSectionEmitter struct {
	lsl    *loadableSectionListener
	name   string
	oh     ioProto
	buffer []uint8
	next   uint64
}

// type emitterState int
//
// const (
// 	swStart         emitterState = 0
// 	swEntryPoint    emitterState = 1
// 	swAddr          emitterState = 2
// 	swData          emitterState = 3
// 	swBigEntryPoint emitterState = 4
// 	swBigAddr       emitterState = 5
// )
//
// type constantWriterState int
//
// const (
// 	cwStart   constantWriterState = 0
// 	cwBigAddr constantWriterState = 1
// 	cwAddr    constantWriterState = 2
// 	cwData    constantWriterState = 3
// )

func newSectionEmitter(lsl *loadableSectionListener, name string, oh ioProto) emitter {
	return &loadableSectionEmitter{
		name:   name,
		next:   0,
		oh:     oh,
		buffer: make([]uint8, anticipation.FileXFerDataLineSize+1),
		lsl:    lsl,
	}
}

func (s *loadableSectionEmitter) receiver() ioProto {
	return s.oh
}

func (s *loadableSectionEmitter) sectionName() string {
	return s.name
}

func (s *loadableSectionEmitter) nextAddr() uint64 {
	return s.next
}

func (s *loadableSectionEmitter) moreLines() bool {
	sectionLast := s.lsl.MustSectionLast(s.name)
	addr, err := s.lsl.SectionAddr(s.name)
	if err != nil {
		panic("unable to read section")
	}
	if (s.next + addr) > sectionLast {
		panic(fmt.Sprintf("read too far into the file! should never happen: %d", (s.next+addr+1)-sectionLast))
	}
	return s.next+addr != sectionLast
}

//string return value here is of limited value, it's already been transmitted
func (s *loadableSectionEmitter) line() (string, error) {
	payloadSize := uint64(0x30)
	sectionLast := s.lsl.MustSectionLast(s.name)
	var result string

	//clip payload size to not request more than the section has
	if payloadSize+s.next > sectionLast {
		payloadSize -= ((payloadSize + s.next) - sectionLast)
	}
	if s.lsl.MustIsInflate(s.name) {
		for i := 0; i < int(payloadSize); i++ {
			s.buffer[i] = 0
		}
	} else {
		r, err := s.lsl.ReadAt(s.name, s.buffer[:payloadSize], int64(s.next))
		if r != int(payloadSize) {
			//short read from reader, so likely at end of section
			payloadSize = uint64(r)
		} else if err != nil {
			return "bad read", err
		}
	}
	addr, err := s.lsl.SectionAddr(s.name)
	if err != nil {
		// log.Fatalf("unable to find section %s when looking for address",s.name)
	}
	result = anticipation.EncodeDataBytes(s.buffer[:payloadSize],
		addr+s.next)
	err = s.oh.Data(result, s.buffer[:payloadSize])
	if err != nil {
		return "bad output", err
	}
	s.next += payloadSize
	return result, nil
}

// normally, you want to call moreLines() immediately after this
func (s *loadableSectionEmitter) reset() {
	s.next = 0
}

func (s *loadableSectionEmitter) read(buffer []uint8) (string, error) {
	return s.oh.Read(buffer)
}

//
// Constant section writer utility implementation
//
type constantEmitter struct {
	io   ioProto
	done bool
	addr uint64
}

func (c *constantEmitter) moreLines() bool {
	return !c.done
}

func (c *constantEmitter) reset() {
	c.done = false
}

func (c *constantEmitter) read(buffer []uint8) (string, error) {
	return c.io.Read(buffer)
}
func (c *constantEmitter) receiver() ioProto {
	return c.io
}
func (c *constantEmitter) nextAddr() uint64 {
	return c.addr
}

//
// Constant section writer for sending the bootloader params
//
type constantParamsEmitter struct {
	*constantEmitter
	params *BootloaderParamsDef
}

func newContstantParamsEmitter(addr uint64, params *BootloaderParamsDef, io ioProto) emitter {
	return &constantParamsEmitter{
		constantEmitter: &constantEmitter{addr: addr, io: io, done: false},
		params:          params,
	}
}
func (c *constantParamsEmitter) sectionName() string {
	return "boot parameters"
}
func (c *constantParamsEmitter) line() (string, error) {
	if c.done {
		panic("should never call line() on a params emitter that is done!")
	}
	//6 uint64s
	payloadSize := uint64(unsafe.Sizeof(BootloaderParamsDef{}))
	rawData := make([]byte, payloadSize)
	for i := 0; i < int(payloadSize); i++ {
		ptr := (*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(&bootloaderParamsCopy)) + uintptr(i)))
		rawData[i] = *ptr
	}
	result := anticipation.EncodeDataBytes(rawData, c.addr)
	err := c.io.Data(result, rawData)
	if err != nil {
		return "data for params", err
	}
	c.done = true
	return result, nil
}

//
// ENTRY POINT EMITTER
//

type constantEntryPointEmitter struct {
	*constantEmitter
}

func newConstantEntryPointEmitter(addr uint64, io ioProto) emitter {
	if addr == 0 {
		panic("no entry point set for constant entry point emitter")
	}
	return &constantEntryPointEmitter{
		constantEmitter: &constantEmitter{addr: addr, io: io, done: false},
	}
}

func (c *constantEntryPointEmitter) sectionName() string {
	return "entry point"
}
func (c *constantEntryPointEmitter) line() (string, error) {
	if c.done {
		panic("should never call line() on a params emitter that is done!")
	}

	if c.addr == 0 {
		panic("no entry point set for constant entry point emitter")
	}
	result := anticipation.EncodeStartLinearAddress(c.addr)
	err := c.io.Data(result, nil)
	if err != nil {
		return "data for params", err
	}

	c.done = true
	return result, nil
}
