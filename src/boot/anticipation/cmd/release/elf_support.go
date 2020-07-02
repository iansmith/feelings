package main

import (
	"debug/elf"
	"fmt"
	"io"
	"os"
)

// ElfListener is the way to get notifications about the different parts
// of the elf file.  Generally, these are called in an order that methods
// higher in the list of ElfListener are called before methods that are lower.
type ElfListener interface {
	//inflate here means that the section is bss or similar so there is no
	//data to be loaded from disk... this is one call per section
	Section(name string, addr uint64, inflate bool, flags int, size uint64)
	EntryPoint(addr uint64, inSectionName string)
	//highest virtual address seen in these sections
	LastAddress(addr uint64)
	//we call this when we find one of the symbols passed in Process
	// if the symbol is not found, you get no callback
	Symbol(name string, value uint64)
}

type notElfFormatErr struct {
}

func (n *notElfFormatErr) Error() string {
	return fmt.Sprintf("file is not elf format (failed to read header)")
}

type tooManyLoadable struct {
}

func (n *tooManyLoadable) Error() string {
	return fmt.Sprintf("found multiple loadable programs in elf file!")
}

type noLoadable struct {
}

func (n *noLoadable) Error() string {
	return fmt.Sprintf("no loadable program found in elf file!")
}

type noTextSegment struct {
}

func (n *noTextSegment) Error() string {
	return fmt.Sprintf("no text segment found in elf file!")
}

type cantReadSymbols struct {
}

func (n *cantReadSymbols) Error() string {
	return fmt.Sprintf("cant read symbols elf file!")
}

type noSuchSection struct {
}

func (n *noSuchSection) Error() string {
	return fmt.Sprintf("cant find given section in the sections of elf file!")
}

var NotElfFormat error = &notElfFormatErr{}
var TooManyLoadablePrograms error = &tooManyLoadable{}
var NoLoadableProgram error = &noLoadable{}
var NoTextSegment error = &noTextSegment{}
var CantReadSymbols error = &cantReadSymbols{}
var NoSuchSection error = &noSuchSection{}

type ElfProcessor interface {
	//returns semantic error about content of the file or nil
	//TooManyLoadablePrograms, NoLoadableProgram, NoTextSegment,CantReadSymbols
	Process(l ElfListener, symbols []string) error
	//Close is necessary to release elf data structures
	Close()
	//read part of a section given by offset into buff (see io.ReadAt)
	ReadAt(sectionName string, buffer []byte, offset int64) (int, error)
	//reads and returns the whole section, if it is compressed
	Data(sectionName string) ([]byte, error)
}

// create a unixElfProcessor, then pass your ElfListener to it's process method.
type unixElfProcessor struct {
	filename string
	elfFile  *elf.File
	sectMap  map[string]*elf.Section
}

//returns NotElfFormat if the file isn't elf (by checking the header)
//returns a not found error if it can't find the file at all.  If you
//get NotElfFormat you don't need to close (elfprocessor is nil anyway).
func NewUnixElfProcessor(filename string) (ElfProcessor, error) {
	fp, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	rd := io.ReaderAt(fp)

	//this call does the check of elf file format:
	//https://golang.org/src/debug/elf/file.go?s=5710:5752#L230
	f, err := elf.NewFile(rd)
	if err != nil {
		fp.Close()
		_, ok := err.(*elf.FormatError)
		if ok {
			return nil, NotElfFormat
		}
		return nil, err
	}

	return &unixElfProcessor{
		filename: filename,
		elfFile:  f,
		sectMap:  make(map[string]*elf.Section),
	}, nil
}

// Close should be called when you no longer need the underlying elf data.
func (e *unixElfProcessor) Close() {
	if e.elfFile != nil {
		e.elfFile.Close()
		e.elfFile = nil
	}
}

//
func (e *unixElfProcessor) Process(l ElfListener, syms []string) error {
	ep := e.elfFile.Entry
	var last uint64

	seenText := false
	for _, section := range e.elfFile.Sections {
		inflate := false
		switch section.Name {
		case ".text":
			seenText = true
		case ".rodata", ".data", ".exc":
		case ".bss":
			inflate = true
		default:
			continue //skip it
		}
		l.Section(section.Name, section.Addr, inflate, 0, section.Size)
		e.sectMap[section.Name] = section
	}
	if !seenText {
		return NoTextSegment
	}
	containedIn := ""
	for _, prog := range e.elfFile.Progs {
		if prog.ProgHeader.Type&elf.PT_LOAD == 0 {
			continue
		}
		for _, s := range e.sectMap {
			if ep >= s.Addr && ep < s.Addr+s.Size {
				containedIn = s.Name
			}
			if s.Addr+s.Size > last {
				last = s.Addr + s.Size
			}
		}
	}

	l.EntryPoint(ep, containedIn)
	l.LastAddress(last)

	symbols, err := e.elfFile.Symbols()
	if err != nil {
		return CantReadSymbols
	}
	for _, s := range symbols {
		n := s.Name
		for _, candidate := range syms /*ones the user is looking for*/ {
			if candidate == n {
				l.Symbol(candidate, s.Value)
			}
		}
	}

	return nil
}

func (u *unixElfProcessor) ReadAt(sectionName string, buffer []byte, offset int64) (int, error) {
	s, ok := u.sectMap[sectionName]
	if !ok {
		return 0, NoSuchSection
	}
	if uint64(uint64(len(buffer))+uint64(offset)) > s.Size {
		buffer = buffer[:s.Size-uint64(offset)]
	}
	r, err := s.ReadAt(buffer, offset)
	if err != nil {
		return 0, err
	}
	return r, nil
}

func (u *unixElfProcessor) Data(sectionName string) ([]byte, error) {
	s, ok := u.sectMap[sectionName]
	if !ok {
		return nil, NoSuchSection
	}
	return s.Data()
}
