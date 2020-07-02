package main

import (
	"log"

	"boot/anticipation"
)

///////////////////////////////////////////////////////////////////////
// loadableSect is how we keep track of sections in the bootloader
// transmission protocol. loadable sect should not have any reference
// to the debug/elf package.
///////////////////////////////////////////////////////////////////////
type loadableSect struct {
	name        string
	vaddr       uint64
	inflate     bool
	addressType anticipation.HexLineType
	size        uint64
}

func newLoadableSect(name string, v uint64, inflate bool, size uint64) *loadableSect {
	if size > 0xffffffff {
		log.Fatalf("unable to process sect %s, it is larger than 0xffffffff (32 bits): %x", name, size)
	}
	return &loadableSect{
		name:    name,
		vaddr:   v,
		inflate: inflate,
		size:    size,
	}
}

///////////////////////////////////////////////////////////////////////
// loadableSectionListener is hooked to an ElfProcessor so we can get
// callbacks about what is in the file.  This indirect arrangement is
// because the same listener is used on feelings.
///////////////////////////////////////////////////////////////////////

type loadableSectionListener struct {
	p ElfProcessor
	t *trueListener
}

func newLoadableSectionListener(filename string) (*loadableSectionListener, error) {
	p, err := NewUnixElfProcessor(filename)
	if err != nil {
		return nil, err
	}
	return &loadableSectionListener{
		p: p,
		t: newTrueListener(),
	}, nil
}

func (l *loadableSectionListener) Process(symbols []string) error {
	return l.p.Process(l.t, symbols)
}

func (l *loadableSectionListener) AllSectionNames() []string {
	result := []string{}
	for name, _ := range l.t.sect {
		result = append(result, name)
	}
	return result
}

func (l *loadableSectionListener) NumSections() int {
	return len(l.t.sect)
}

func (l *loadableSectionListener) Close() {
	if l.p != nil {
		l.p.Close()
	}
}

func (l *loadableSectionListener) GetEntry() (uint64, string) {
	return l.t.entryPoint, l.t.entrySectionName
}
func (l *loadableSectionListener) GetLastAddr() uint64 {
	return l.t.last
}

func (l *loadableSectionListener) SymbolValue(s string) (uint64, bool) {
	u, b := l.t.sym[s]
	return u, b
}
func (l *loadableSectionListener) SectionSize(name string) (uint64, error) {
	s, ok := l.t.sect[name]
	if !ok {
		return 0, NoSuchSection
	}
	return s.size, nil
}
func (l *loadableSectionListener) MustSectionLast(name string) uint64 {
	s, ok := l.t.sect[name]
	if !ok {
		panic("unable to find the section " + name)
	}
	return s.size + s.vaddr
}

func (l *loadableSectionListener) ReadAt(name string, buffer []byte, offset int64) (int, error) {
	return l.p.ReadAt(name, buffer, offset)
}
func (l *loadableSectionListener) Data(name string) ([]byte, error) {
	return l.p.Data(name)
}
func (l *loadableSectionListener) SectionAddr(name string) (uint64, error) {
	s, ok := l.t.sect[name]
	if !ok {
		return 0, NoSuchSection
	}
	return s.vaddr, nil
}

func (l *loadableSectionListener) MustIsInflate(name string) bool {
	s, ok := l.t.sect[name]
	if !ok {
		panic("unable to find section " + name + " to check to see if it needs inflating")
	}
	return s.inflate
}

///////////////////////////////////////////////////////////////////////
// trueListener: nested inside the loadableSectionListener is the
// trueListener... this is to more clearly separate the callbacks
// from loadableSectionListener API.
///////////////////////////////////////////////////////////////////////

type trueListener struct {
	sect             map[string]*loadableSect
	entryPoint       uint64
	entrySectionName string
	sym              map[string]uint64
	last             uint64
}

func newTrueListener() *trueListener {
	result := &trueListener{}
	result.sect = make(map[string]*loadableSect)
	result.sym = make(map[string]uint64)
	return result
}

//Section callback
func (t *trueListener) Section(name string, addr uint64, inflate bool, _ int, size uint64) {
	s := newLoadableSect(name, addr, inflate, size)
	t.sect[name] = s
}

//EntryPoint callback
func (t *trueListener) EntryPoint(addr uint64, name string) {
	t.entryPoint = addr
	t.entrySectionName = name
}

//Symbol callback
func (t *trueListener) Symbol(name string, value uint64) {
	t.sym[name] = value
}

//LastAddress callback
func (t *trueListener) LastAddress(addr uint64) {
	t.last = addr
}
