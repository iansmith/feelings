package main

import (
	"time"

	"boot/anticipation"

	"flag"
	"fmt"
	"io"
	"log"
	"os"
)

const KernelLoadPoint = 0xfffffc0000000000
const PageSize = 0x10000

// this is the place in the KERNEL where we are going to place a copy of
// the structure BootloaderParamsDef... the bootloaderParamsCopy is just to
// make it easier to set the fields
var bootloaderParamsLocation uint64 //ptr
var bootloaderParamsCopy BootloaderParamsDef

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

var KeySymbols = []string{"bootloader_params"}

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
	lsl, err := newLoadableSectionListener(flag.Arg(0))
	if err != nil {
		log.Fatalf(" %v", err)
	}
	if err := lsl.Process(KeySymbols); err != nil {
		log.Fatalf("%s: %v", flag.Arg(0), err)
	}
	if *verbose > 0 {
		ep, name := lsl.GetEntry()
		log.Printf("@@@ opening file %s, entry point is %x in %s",
			flag.Arg(0), ep, name)
	}
	defer lsl.Close()

	v, hasBootloaderSym := lsl.SymbolValue(KeySymbols[0])
	if !hasBootloaderSym {
		log.Fatalf("unable to find %s in the elf file", KeySymbols[0])
	}
	bootloaderParamsLocation = v

	//
	// Where is the output going?
	//

	if *testFlag {
		selfTest(flag.Arg(0), lsl)
	}
	if *ptyFlag != "" {
		oh := newTTYIOProto(*ptyFlag)
		if oh == nil {
			log.Fatalf("unable to connect to %s", *ptyFlag)
		}
		protocol(flag.Arg(0), lsl, oh)
	}
	if !*testFlag && *ptyFlag == "" {
		log.Printf("neither testflag nor pty flag/parameter supplied, not doing anything")
	}

}

func selfTest(filename string, lsl *loadableSectionListener) {

	encodeAndDecode(filename, lsl)
	protocol(filename, lsl, newAddrCheckReceiver())
}

func encodeAndDecode(filename string, lsl *loadableSectionListener) {
	for _, l := range lsl.AllSectionNames() {
		log.Printf("encoding test: test encoding of file %s, sect %s", filename, l)
		if lsl.MustIsInflate(l) {
			continue //bss
		}
		s, err := lsl.SectionSize(l)
		if err != nil {
			log.Fatalf("unable to find section %s to get size", l)
		}
		if s == 0 {
			continue // empty
		}
		buffer := make([]byte, anticipation.FileXFerDataLineSize)
		bb := anticipation.NewNullByteBuster()
		offset := uint64(0)
		for {
			if offset == s {
				break
			}
			r, err := lsl.ReadAt(l, buffer, int64(offset))
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Fatalf("failed trying to read sect of binary: %v", err)
			}
			line := anticipation.EncodeDataBytes(buffer[:r], offset)
			converted, lt, addr, err := anticipation.DecodeAndCheckStringToBytes(line)
			if lt != anticipation.DataLine {
				log.Fatalf("unexpected line type: %s", lt)
			}
			if uint64(addr) != (offset) {
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
			offset += uint64(len(buffer[:r]))
		}
	}
	log.Printf("encoding test: everything seems to be ok")
}

func protocol(filename string, lsl *loadableSectionListener, oh ioProto) {
	//
	//build a list of emitters for each blob of stuff we send
	//

	//two extra sections: one for kernel params, one for entry point
	emitterList := make([]emitter, lsl.NumSections()+2)
	for i, name := range lsl.AllSectionNames() {
		se := newSectionEmitter(lsl, name, oh)
		emitterList[i] = se
	}

	//entry point emitter
	ep, n := lsl.GetEntry()
	if ep == 0 || n == "" {
		panic("no entry point set")
	}
	emitterList[lsl.NumSections()] = newConstantEntryPointEmitter(ep, oh)

	//next emmitter does the boot parameter copying magic
	emitterList[lsl.NumSections()+1] = newContstantParamsEmitter(bootloaderParamsLocation,
		&bootloaderParamsCopy, oh)
	computeBootloaderParameters()

	//
	// Protocol Loop
	//

	tx := newTransmitLooper(emitterList, oh)
	if *verbose > 0 {
		log.Printf("@@@ file %s, sect %s", filename, tx.current.sectionName())
	}

	//right now, we only use the first of 4 params
	tx.param[kernelParamAddressBlockAddr] = bootloaderParamsLocation
	tx.param[1] = 0
	tx.param[2] = 0
	tx.param[3] = 0

	name := tx.current.sectionName()
	tx.current.receiver().NewSection(lsl, name)
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
			log.Printf("ignoring empty response, maybe should RETRY??\n")
			//sendLineToDevice(tx)
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
			log.Printf("RETRY offset addr 0x%08x in %s\n",
				tx.current.nextAddr(), tx.current.sectionName())
			tx.errorCount++
			switch {
			case tx.errorCount > 5:
				log.Fatalf("aborting, too many errors in a row")
			case tx.errorCount > 2:
				tx.current.reset()
				b := tx.current.moreLines()
				if !b {
					log.Fatalf("bad state, should never reset an empty emitter!")
				}
			}
		case '.':
			tx.errorCount = 0 //no more consecutive errors
			switch tx.state {
			case tsParams:
				tx.next() //called for effect
				sendLineToDevice(tx)
			case tsData:
				if !tx.current.moreLines() { //done with sect?
					if tx.next() {
						if *verbose > 0 {
							log.Printf("@@@ file %s, section %s", filename,
								tx.current.sectionName())
						}
					}
					tx.current.receiver().NewSection(lsl, name)
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
		panic("here")
		log.Fatalf("error reading moreLines line from encoder: %v", err)
	}
	if *verbose == 2 {
		log.Printf("--> %s", l)
	}

}

func computeBootloaderParameters() {
	stackPages := uint64(2)
	heapPages := uint64(8)

	//we need to set the bootloader params
	bootloaderParamsCopy.UnixTime = uint64(time.Now().Unix())
	page := uint64(KernelLoadPoint)
	//does this page cover the kernel's loaded size
	for page+(PageSize-1) < bootloaderParamsCopy.KernelLast {
		page += PageSize
	}
	// kernel code takes N pages
	// example:kernel stack takes 2 page (N+1, N+2
	// example: kernel heap takes 8 pages (N+3...N+10)
	page += (stackPages * PageSize)
	//this is the "wrong" end of the stack page (if stack reaches here, we are hosed)
	bootloaderParamsCopy.StackPointer = page + (PageSize - 0x10) //16 byte alignment required
	page += PageSize
	bootloaderParamsCopy.HeapStart = page
	page += (heapPages * PageSize)
	bootloaderParamsCopy.HeapEnd = page + (PageSize - 8) //example: END of N+10th page
	//log.Printf("kernel boot parameters: %#v and address %x", bootloaderParamsCopy, bootloaderParamsLocation)
}
