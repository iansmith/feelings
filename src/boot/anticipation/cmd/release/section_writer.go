package main

import (
	"boot/anticipation"
	"debug/elf"
	"io"
	"log"
)

const resetIncrement = 0xff00 //a little less than 64K

type sectionWriter interface {
	line() (string, error)
	next() bool //true if no more lines
	reset()     //return to beginning of sect
	name() string
	read([]uint8) (string, error)
	receiver() protoReceiver
}

type loadableSectionWriter struct {
	sect              *elf.Section
	loadable          *loadableSect
	state             sectionWriterState
	current           uint32
	resetPoint        uint32
	oh                protoReceiver
	buffer            []uint8
	seeker            io.ReadSeeker
	pendingLineLength uint16
}

type sectionWriterState int

const (
	swStart         sectionWriterState = 0
	swEntryPoint    sectionWriterState = 1
	swAddr          sectionWriterState = 2
	swData          sectionWriterState = 3
	swBigEntryPoint sectionWriterState = 4
	swBigAddr       sectionWriterState = 5
)

func newSectionEmitter(s *elf.Section, l *loadableSect, oh protoReceiver) *loadableSectionWriter {
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
	return &loadableSectionWriter{sect: s,
		loadable:   l,
		state:      swStart,
		oh:         oh,
		buffer:     make([]uint8, anticipation.FileXFerDataLineSize+1),
		current:    0,
		resetPoint: resetIncrement, //16 bits in a data line means we need a reset before rollover
		seeker:     nil,
	}
}

func (s *loadableSectionWriter) receiver() protoReceiver {
	return s.oh
}

func (s *loadableSectionWriter) name() string {
	return s.sect.Name
}
func (s *loadableSectionWriter) section() *elf.Section {
	return s.sect
}

func (s *loadableSectionWriter) next() bool {
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
	panic("unexpected sectionWriter state")
}

//string return value here is of limited value, it's already been transmitted
func (s *loadableSectionWriter) line() (string, error) {
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
	panic("unexpected sectionWriter state!")
}

// normally, you want to call next() immediately after this
func (s *loadableSectionWriter) reset() {
	s.state = swStart
}

func (s *loadableSectionWriter) read(buffer []uint8) (string, error) {
	return s.oh.Read(buffer)
}
