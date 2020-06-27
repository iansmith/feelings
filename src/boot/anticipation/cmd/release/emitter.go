package main

import (
	"boot/anticipation"
	"debug/elf"
	"io"
	"log"
	"unsafe"
)

const resetIncrement = 0xff00 //a little less than 64K

///////////////////////////////////////////////////////////////////////////////
// emitter can take a blob of data and emit the necessary commands to transmit
// that memory to the other side. It uses a ioProto to do the actual IO work
// but it computes the lines and addresses to send.
///////////////////////////////////////////////////////////////////////////////
type emitter interface {
	line() (string, error)
	next() bool //true if no more lines
	reset()     //return to beginning of sect
	name() string
	read([]uint8) (string, error)
	receiver() ioProto
	currentAddr() uint32
}

// loadableSectionEmitter works from the blob of data in a loadable section of
// an elf format binary
type loadableSectionEmitter struct {
	sect              *elf.Section
	loadable          *loadableSect
	state             emitterState
	current           uint32
	resetPoint        uint32
	oh                ioProto
	buffer            []uint8
	seeker            io.ReadSeeker
	pendingLineLength uint16
}

type emitterState int

const (
	swStart         emitterState = 0
	swEntryPoint    emitterState = 1
	swAddr          emitterState = 2
	swData          emitterState = 3
	swBigEntryPoint emitterState = 4
	swBigAddr       emitterState = 5
)

type constantWriterState int

const (
	cwStart   constantWriterState = 0
	cwBigAddr constantWriterState = 1
	cwAddr    constantWriterState = 2
	cwData    constantWriterState = 3
)

func newSectionEmitter(s *elf.Section, l *loadableSect, oh ioProto) emitter {
	if s.Size > 0xffff {
		log.Fatalf("unable to encode inflating sect (.bss) because size is greater than 0xffff (16 bits): %x", s.Size)
	}
	if l.vaddr&0xffff_ffff_0000_000 == 0 && l.vaddr&0xffff != 0 {
		if l.vaddr&0xf != 0 {
			log.Fatalf("unable to create base addr for sect %s, %x has neither lower 16 or lower 4 bits clear", l.name, l.vaddr)
		}
		if l.vaddr&0xfff00000 != 0 {
			log.Fatalf("unable to create base addr for sect %s, %x has width > 16bits that will not fit in ESA", l.name, l.vaddr)
		}
		l.addressType = anticipation.ExtendedSegmentAddress
	} else {
		l.addressType = anticipation.ExtensionBigEntryPoint
	}
	return &loadableSectionEmitter{sect: s,
		loadable:   l,
		state:      swStart,
		oh:         oh,
		buffer:     make([]uint8, anticipation.FileXFerDataLineSize+1),
		current:    0,
		resetPoint: resetIncrement, //16 bits in a data line means we need a reset before rollover
		seeker:     nil,
	}
}

func (s *loadableSectionEmitter) receiver() ioProto {
	return s.oh
}

func (s *loadableSectionEmitter) name() string {
	return s.sect.Name
}
func (s *loadableSectionEmitter) section() *elf.Section {
	return s.sect
}
func (s *loadableSectionEmitter) currentAddr() uint32 {
	return s.current
}

func (s *loadableSectionEmitter) next() bool {
	switch s.state {
	case swStart:
		if s.loadable.entrypoint != uint64signal {
			s.state = swBigEntryPoint
		} else {
			s.state = swBigAddr
		}
		return true
	case swEntryPoint:
		s.state = swAddr
		return true
	case swBigEntryPoint:
		s.state = swEntryPoint
		return true
	case swBigAddr:
		s.state = swAddr
		return true
	case swAddr:
		s.state = swData
		if !s.loadable.inflate {
			s.seeker = s.sect.Open()
			s.seeker.Seek(0, io.SeekStart)
		}
		s.current = 0
		return true
	case swData:
		s.current += uint32(s.pendingLineLength)
		if uint64(s.current) == s.sect.Size {
			return false
		}
		if s.current == s.resetPoint {
			log.Printf("we are resetting current location because nearing 16 bit limit")
			s.resetPoint += resetIncrement
			s.state = swBigAddr
		}
		return true
	}
	panic("unexpected emitter state")
}

//string return value here is of limited value, it's already been transmitted
func (s *loadableSectionEmitter) line() (string, error) {
	switch s.state {
	case swStart:
		panic("should never request a line in start state")
	case swBigEntryPoint:
		top := uint32(s.loadable.entrypoint >> 32)
		result := anticipation.EncodeBigEntry(top)
		s.pendingLineLength = uint16(len(result))
		err := s.oh.BigEntryPoint(result, top)
		if err != nil {
			return "", err
		}
		return result, nil
	case swEntryPoint:
		bottom32 := uint32(s.loadable.entrypoint & 0xffff_ffff)
		result := anticipation.EncodeSLA(bottom32)
		s.pendingLineLength = uint16(len(result))
		err := s.oh.EntryPoint(result, bottom32)
		if err != nil {
			return "", err
		}
		return result, nil
	case swBigAddr:
		top := uint32(s.loadable.vaddr >> 32)
		result := anticipation.EncodeBigAddr(top)
		s.pendingLineLength = uint16(len(result))
		err := s.oh.BigBaseAddr(result, top)
		if err != nil {
			return "", err
		}
		return result, nil
	case swAddr:
		bottom := uint32(s.loadable.vaddr & 0xffff_ffff)
		if s.loadable.addressType == anticipation.ExtendedSegmentAddress {
			result := anticipation.EncodeESA(uint16(bottom >> 4))
			s.pendingLineLength = uint16(len(result))
			err := s.oh.BaseAddrESA(result, bottom)
			if err != nil {
				return "", err
			}
			return result, nil
		} else {
			//this cloud be EITHER ExtendedLinearAddr or ExtensionBigLinearAddr
			result := anticipation.EncodeELA(uint16(bottom >> 16))
			s.pendingLineLength = uint16(len(result))
			err := s.oh.BaseAddrELA(result, bottom)
			if err != nil {
				return "", err
			}
			return result, nil
		}
	case swData:
		payloadSize := uint32(0x30)
		if uint32(s.sect.Size)-s.current < payloadSize {
			payloadSize = uint32(s.sect.Size) - s.current
		}
		currentLowest16ForProtocol := uint16(s.current&0xffff) + uint16(s.sect.Addr&0xffff)
		var result string
		if s.loadable.inflate {
			result = anticipation.EncodeDataBytes(s.buffer[:payloadSize], currentLowest16ForProtocol)
		} else {
			_, err := s.seeker.Seek(int64(s.current), io.SeekStart)
			if err != nil {
				return "", err
			}
			_, err = s.seeker.Read(s.buffer[:payloadSize])
			if err != nil {
				return "", err
			}
			result = anticipation.EncodeDataBytes(s.buffer[:payloadSize], currentLowest16ForProtocol)
		}
		s.pendingLineLength = uint16(payloadSize)
		err := s.oh.Data(result, s.buffer[:payloadSize])
		if err != nil {
			return "", err
		}
		return result, nil
	}
	panic("unexpected emitter state!")
}

// normally, you want to call next() immediately after this
func (s *loadableSectionEmitter) reset() {
	s.state = swStart
}

func (s *loadableSectionEmitter) read(buffer []uint8) (string, error) {
	return s.oh.Read(buffer)
}

//
// Constant section writer for sending the bootloader params
//
type constantParamsEmitter struct {
	addr              unsafe.Pointer
	params            *BootloaderParamsDef
	state             constantWriterState
	io                ioProto
	done              bool
	pendingLineLength uint16
}

func newContstantParamsEmitter(addr unsafe.Pointer, params *BootloaderParamsDef, io ioProto) emitter {
	return &constantParamsEmitter{addr: addr, params: params, io: io, state: cwStart}
}
func (c *constantParamsEmitter) line() (string, error) {
	switch c.state {
	case cwStart:
		panic("should never request a line in start state")
	case cwBigAddr:
		top := uint32(uintptr(c.addr) >> 32)
		result := anticipation.EncodeBigAddr(top)
		c.pendingLineLength = uint16(len(result))
		err := c.io.BigBaseAddr(result, top)
		if err != nil {
			return "", err
		}
		return result, nil
	case cwAddr:
		bottom := uint32(uintptr(c.addr) & 0xffffffff)
		result := anticipation.EncodeELA(uint16(bottom >> 16))
		c.pendingLineLength = uint16(len(result))
		err := c.io.BaseAddrELA(result, bottom)
		if err != nil {
			return "", err
		}
		return result, nil

	case cwData:
		//6 uint64s
		currentLowest16ForProtocol := uint16(uintptr(c.addr) & 0xffff)
		payloadSize := uint32(unsafe.Sizeof(BootloaderParamsDef{}))
		rawData := make([]byte, payloadSize)
		for i := 0; i < int(payloadSize); i++ {
			ptr := (*byte)(unsafe.Pointer(uintptr(unsafe.Pointer(&bootloaderParamsCopy)) + uintptr(i)))
			rawData[i] = *ptr
		}
		result := anticipation.EncodeDataBytes(rawData, currentLowest16ForProtocol)
		err := c.io.Data(result, rawData)
		if err != nil {
			return "", err
		}
		return result, nil
	}
	panic("unexpected emitter state!")

}
func (c *constantParamsEmitter) next() bool {
	switch c.state {
	case cwStart:
		c.state = cwBigAddr
		return true
	case cwBigAddr:
		c.state = cwAddr
		return true
	case cwAddr:
		c.state = cwData
		return true
	case cwData:
		//only one line
		if c.done {
			return true
		}
		c.done = true
		return false
	}
	panic("bad state of constantParamsEmitter")
}
func (c *constantParamsEmitter) reset() {
	c.done = false
}
func (c *constantParamsEmitter) name() string {
	return "bootloader parameters"
}
func (c *constantParamsEmitter) read(buffer []uint8) (string, error) {
	return c.io.Read(buffer)
}
func (c *constantParamsEmitter) receiver() ioProto {
	return c.io
}
func (c *constantParamsEmitter) currentAddr() uint32 {
	return uint32(uintptr(c.addr) & 0xffffffff)
}
