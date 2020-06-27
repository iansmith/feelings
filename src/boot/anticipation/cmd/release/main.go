package main

import (
	"time"

	"boot/anticipation"

	"debug/elf"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"unsafe"
)

const uint64signal = uint64(0x1234567887654321)
const KernelLoadPoint = 0xfffffc0000000000
const PageSize = 0x10000

type transmitState int

var helpFlag = flag.Bool("h", false, "get usage info")
var testFlag = flag.Bool("t", false, "encode a file and decode each data line to see if they match")
var ptyFlag = flag.String("p", "", "supply a pseudo TTY to output to")
var verbose = flag.Int("v", 0, "verbosity level: 0 terse (default), 1 debug info, 2 show everything ")

// sadly, we had to COPY this here from upbeat.BootLoaderParamsDef because the
// hostgo will refuse to link due to other things in lib upbeat
type BootloaderParamsDef struct {
	EntryPoint   uint64
	KernelLast   uint64
	UnixTime     uint64
	StackPointer uint64
	HeapStart    uint64
	HeapEnd      uint64
}

var bootloaderParamsLocation unsafe.Pointer
var bootloaderParamsCopy BootloaderParamsDef

///////////////////////////////////////////////////////////////////////
// loadableSect is how we match up program headers to sections
///////////////////////////////////////////////////////////////////////
type loadableSect struct {
	name        string
	vaddr       uint64
	entrypoint  uint64
	inflate     bool
	addressType anticipation.HexLineType
}

func newLoadableSect(name string, v uint64, inflate bool, flg elf.SectionFlag, size uint64) *loadableSect {
	if flg&elf.SHF_ALLOC == 0 {
		log.Fatalf("unable to process sect %s: it does not have the SHF_ALLOC flag", name)
	}
	if size > 0xffffffff {
		log.Fatalf("unable to process sect %s, it is larger than 0xffffffff (32 bits): %x", name, size)
	}
	return &loadableSect{
		name:       name,
		vaddr:      v,
		inflate:    inflate,
		entrypoint: uint64signal,
	}
}
func (l *loadableSect) setEntryPoint(a uint64) {
	//no restriction on the size anymore, this was previously used to trap > 32 bit size values
}

///////////////////////////////////////////////////////////////////////
// main
///////////////////////////////////////////////////////////////////////
func main() {
	flag.Parse()
	if flag.NArg() == 0 {
		usage()
	}
	if *helpFlag {
		usage()
	}
	fp, err := elf.Open(flag.Arg(0))
	if err != nil {
		log.Fatalf(" %v", err)
	}
	if *verbose > 0 {
		log.Printf("@@@ opening file %s, entry point is %x", flag.Arg(0), fp.Entry)
	}
	defer fp.Close()

	//get a list of loadable sections
	lsect := []*loadableSect{}
	for _, section := range fp.Sections {
		switch section.Name {
		case ".text", ".rodata", ".data", ".exc":
			lsect = append(lsect, newLoadableSect(section.Name, section.Addr, false, section.Flags, section.Size))
		case ".bss":
			lsect = append(lsect, newLoadableSect(section.Name, section.Addr, true, section.Flags, section.Size))
		}
	}

	//no need to check entry point for 32 bits anymore
	entryPoint := fp.Entry

	//walk program headers, marking the sections as needed in terms of where and when to load
	for _, prog := range fp.Progs {
		if prog.ProgHeader.Type&elf.PT_LOAD == 0 {
			continue
		}
		for _, ls := range lsect {
			s := fp.Section(ls.name)
			if entryPoint >= ls.vaddr && entryPoint < ls.vaddr+s.Size {
				ls.entrypoint = entryPoint
				bootloaderParamsCopy.EntryPoint = ls.entrypoint
			}
			if ls.vaddr+s.Size > bootloaderParamsCopy.KernelLast {
				bootloaderParamsCopy.KernelLast = ls.vaddr + s.Size
			}
		}
	}

	symbols, err := fp.Symbols()
	if err != nil {
		log.Fatalf("unable to load symbols: %v", err)
	}
	for _, sym := range symbols {
		if sym.Name == "bootloader_params" {
			bootloaderParamsLocation = unsafe.Pointer(uintptr(sym.Value))
			break
		}
	}

	//check that we have an entry point
	ok := false
	for _, l := range lsect {
		if l.entrypoint != uint64signal {
			ok = true
			break
		}
	}
	if !ok {
		log.Fatalf("unable to match entry point %x with any sect!", entryPoint)
	}

	//
	// Where is the output going?
	//

	if *testFlag {
		selfTest(flag.Arg(0), fp, lsect)
	}
	if *ptyFlag != "" {
		oh := newTTYIOProto(*ptyFlag)
		if oh == nil {
			log.Fatalf("unable to connect to %s", *ptyFlag)
		}
		protocol(flag.Arg(0), fp, lsect, oh)
	}
	if !*testFlag && *ptyFlag == "" {
		log.Printf("neither testflag nor pty flag/parameter supplied, not doing anything")
	}

}

func selfTest(filename string, fp *elf.File, lsect []*loadableSect) {

	encodeAndDecode(filename, fp, lsect)
	protocol(filename, fp, lsect, newAddrCheckReceiver())
}

func encodeAndDecode(filename string, fp *elf.File, lsect []*loadableSect) {
	for _, l := range lsect {
		log.Printf("encoding test: test encoding of file %s, sect %s", filename, l.name)
		if l.inflate {
			continue //bss
		}
		s := fp.Section(l.name)
		if s.Size == 0 {
			continue // empty
		}
		buffer := make([]byte, anticipation.FileXFerDataLineSize)
		bb := anticipation.NewNullByteBuster()
		offset := uint64(0)
		for {
			if offset == s.Size {
				break
			}
			r, err := s.ReadAt(buffer, int64(offset))
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatalf("failed trying to read sect of binary: %v", err)
			}
			line := anticipation.EncodeDataBytes(buffer[:r], uint16(offset))
			converted, lt, addr, err := anticipation.DecodeAndCheckStringToBytes(line)
			if lt != anticipation.DataLine {
				log.Fatalf("unexpected line type: %s", lt)
			}
			if addr != uint32(offset) {
				log.Fatalf("unexpected offset (expected 0x%04x but got 0x%04x)", offset, addr)
			}
			if err != nil {
				log.Fatalf("unable to decode line: %v", err)
			}
			anticipation.ProcessLine(lt, converted, bb)
			for i := 0; i < len(buffer[:r]); i++ {
				c := converted[4+i]
				if buffer[i] != c {
					log.Fatalf("bad encoding on byte %d from line %s", line)
				}
			}
			offset += uint64(len(buffer))
		}
	}
	log.Printf("encoding test: everything seems to be ok")
}

func protocol(filename string, fp *elf.File, lsect []*loadableSect, oh ioProto) {
	//
	//build a list of what we need
	//
	emitterList := make([]emitter, len(lsect)+1) //+1 for kernel params emitter
	for i, l := range lsect {
		s := fp.Section(l.name)
		se := newSectionEmitter(s, l, oh)
		emitterList[i] = se
	}
	//last emmitter does the boot parameter magic
	emitterList[len(lsect)] = newContstantParamsEmitter(bootloaderParamsLocation, &bootloaderParamsCopy, oh)
	//we need to set the bootloader params
	bootloaderParamsCopy.UnixTime = uint64(time.Now().Unix())
	page := uint64(KernelLoadPoint)
	//does this page cover the kernel's loaded size
	for page+(PageSize-1) < bootloaderParamsCopy.KernelLast {
		page += PageSize
	}
	// kernel code takes N pages
	// kernel stack takes 1 page (N+1)
	// kernel heap takes 2 pages (N+2,N+3)
	page += PageSize
	//this is the "wrong" end of the stack page (if stack reaches here, we are hosed)
	bootloaderParamsCopy.StackPointer = page
	page += PageSize
	bootloaderParamsCopy.HeapStart = page
	page += PageSize
	bootloaderParamsCopy.HeapEnd = page + (PageSize - 8) //END of second page
	log.Printf("kernel boot parameters: %#v and address %p", bootloaderParamsCopy, bootloaderParamsLocation)
	if len(emitterList) < 2 {
		log.Fatalf("unable to find any data to release! No sections for transmission!")
	}

	//
	// Protocol Loop
	//

	tx := newTransmitLooper(emitterList, oh)
	if *verbose > 0 {
		log.Printf("@@@ file %s, sect %s", filename, tx.current.name())
	}
	name := tx.current.name()
	copyOfSect := fp.Section(name)
	tx.current.receiver().NewSection(copyOfSect)
outer:
	for {
		l, err := tx.read()
		if err != nil {
			log.Fatalf("!!! error reading from tty: %v", err)
		}
		if *verbose == 2 {
			log.Printf("<-- %s", l)
		}
		if len(l) == 0 {
			log.Printf("ignoring empty line")
			continue
		}

		switch l[0] {
		case '#': //comment
			log.Print("### ", l[1:])
		case '@': //debug info
			if *verbose > 0 {
				log.Print("@@@ ", l[1:])
			}
		case '!': //error
			if *verbose < 2 { //verbose user has already seen this, no sense repeating
				log.Printf("!!! %s", l[1:])
			}
			log.Printf("RETRY offset addr 0x%08x in %s\n", tx.current.currentAddr(), tx.current.name())
			tx.errorCount++
			switch {
			case tx.errorCount > 5:
				log.Fatalf("aborting, too many errors in a row")
			case tx.errorCount > 2:
				tx.current.reset()
				b := tx.current.next()
				if !b {
					log.Fatalf("bad state, should never reset an empty emitter!")
				}
			}
		case '.':
			tx.errorCount = 0 //no more consecutive errors
			switch tx.state {
			case tsTime:
				tx.next() //called for effect
				sendLineToDevice(tx)
			case tsData:
				if !tx.current.next() { //done with sect?
					if tx.next() {
						if *verbose > 0 {
							log.Printf("@@@ file %s, sect %s", filename, tx.current.name())
						}
					}
					tx.current.receiver().NewSection(fp.Section(tx.current.name()))
				}
				sendLineToDevice(tx)
			case tsEnd:
				break outer
			}
		default:
			log.Printf("ignoring unexpected response: %s", l)
		}
	}
	if _, ok := oh.(*verifyIOProto); ok {
		log.Printf("verified all the data bytes and the address of loading them.")
		os.Exit(0)
	}

	log.Printf("transmission successful: %s", flag.Arg(0))
	log.Printf("--- kernel log ---")
	for {
		l, err := tx.read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("failed to read from client: %v", err)
		}
		if len(l) == 0 {
			if *verbose > 0 {
				log.Printf("@@@ ignoring empty line")
				continue
			}
		}
		switch l[0] {
		case '*':
			if *verbose == 2 {
				fmt.Printf("%s\n", l[1:])
			}
		case '@':
			if *verbose > 0 {
				fmt.Printf("@@@ %s\n", l[1:])
			}
		case '!':
			fmt.Printf("!!! %s\n", l[1:])
		case '#':
			fmt.Printf("### %s\n", l[1:])
		default:
			fmt.Printf("%s\n", l)
		}
	}
}

func usage() {
	fmt.Printf("usage: release [feelings kernel elf-format]\n")
	flag.PrintDefaults()
	os.Exit(1)
}

func sendLineToDevice(tx *transmitLooper) {
	// we get the line as a courtesy, but it's already been sent
	l, err := tx.line()
	if err != nil {
		log.Fatalf("error reading next line from encoder: %v", err)
	}
	if *verbose == 2 {
		log.Printf("--> %s", l)
	}

}
