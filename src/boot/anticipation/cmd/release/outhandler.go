package main

import (
	"debug/elf"
	"errors"
	"fmt"
	"log"

	"boot/anticipation"

	tty "github.com/mattn/go-tty"
)

///////////////////////////////////////////////////////////////////////
// protoReceiver deals with what to do with encoded lines
///////////////////////////////////////////////////////////////////////
type protoReceiver interface {
	Data(s string, data []uint8) error             //data is the original data (for cross check)
	DataInflate(s string, data uint16) error       // data is number of inflated bytes
	EntryPoint(s string, addr uint32) error        // addr is the lower 32bits of entry point
	BigEntryPoint(s string, addr uint32) error     // addr is the upper 32bits of entry point
	BaseAddrESA(s string, addr uint32) error       //  addr is 32bit base addr
	BigBaseAddr(s string, addr uint32) error       //addr is upper 32 bits of  base addr
	BaseAddrELA(s string, addr uint32) error       //addr is lower 32 bits of  base addr
	ExtensionUnixTime(s string, addr uint32) error //ticks in secs from epoch
	Read([]uint8) (string, error)                  //read the next thing from the other side
	NewSection(*elf.Section) error                 //just a notification
	EOF() (string, error)                          //just a notification
}

///////////////////////////////////////////////////////////////////////
// ttyReceiver is the model
///////////////////////////////////////////////////////////////////////

type ttyReceiver struct {
	io *tty.TTY
}

func newTTYReceiver(devTTYPath string) *ttyReceiver { //returns null when it can't open
	ttyObj, err := tty.OpenDevice(devTTYPath)
	if err != nil {
		log.Fatalf("%v ,,,%T", err, err)
	}
	_ = ttyObj.MustRaw()

	if err != nil {
		log.Printf("%v", err)
		return nil
	}
	return &ttyReceiver{io: ttyObj}
}
func (t *ttyReceiver) NewSection(_ *elf.Section) error {
	return nil //nothing to do for us
}

func (t *ttyReceiver) EntryPoint(s string, _ uint32) error {
	t.sendString(s)
	return nil
}
func (t *ttyReceiver) BigEntryPoint(s string, _ uint32) error {
	t.sendString(s)
	return nil
}
func (t *ttyReceiver) BaseAddrESA(s string, _ uint32) error {
	t.sendString(s)
	return nil
}
func (t *ttyReceiver) BaseAddrELA(s string, _ uint32) error {
	t.sendString(s)
	return nil
}
func (t *ttyReceiver) BigBaseAddr(s string, _ uint32) error {
	t.sendString(s)
	return nil
}
func (t *ttyReceiver) Data(s string, _ []uint8) error {
	t.sendString(s)
	return nil
}
func (t *ttyReceiver) DataInflate(s string, _ uint16) error {
	t.sendString(s)
	return nil
}

func (t *ttyReceiver) sendString(s string) {
	t.io.Output().WriteString(s)
	t.io.Output().WriteString("\n")
}
func (t *ttyReceiver) EOF() (string, error) {
	t.sendString(EOFLine)
	return EOFLine, nil
}

func (t *ttyReceiver) Read(data []uint8) (string, error) {
	count := uint16(0)
	dropped := 0
	for {
		r, err := t.io.Input().Read(data[count : count+1])
		if err != nil {
			return "", err
		}
		if r == 0 {
			log.Printf("retrying failed read (size zero)")
			continue
		}
		switch {
		case data[count] < 32 && data[count] != 10:
			continue
		case data[count] == 10:
			if dropped != 0 {
				log.Printf("dropped %d characters from line", dropped)
			}
			return string(data[:count]), nil
		default:
			if count == uint16(len(data)-1) {
				dropped++
				continue
			}
			count++
		}
	}
}

func (t *ttyReceiver) ExtensionUnixTime(l string, _ uint32) error {
	t.sendString(l)
	return nil
}

///////////////////////////////////////////////////////////////////////
// verifyReceiver checks that the loader is putting the code in the
// right place. It also verifies the bytes against the disk version.
// Used in tests (the -t option)
///////////////////////////////////////////////////////////////////////
type verifyReceiver struct {
	section *elf.Section
	data    []uint8
	current uint64
}

func newAddrCheckReceiver() protoReceiver {
	return &verifyReceiver{} //assumes they will call new sect in a sec
}

func (v *verifyReceiver) BigEntryPoint(s string, addr uint32) error {
	return nil
}

func (v *verifyReceiver) BigBaseAddr(s string, addr uint32) error {
	prev := v.current & 0xffff_ffff
	v.current = prev | (uint64(addr) << 32)
	return nil
}

func (v *verifyReceiver) NewSection(s *elf.Section) error {
	d, err := s.Data()
	if err != nil {
		return err
	}
	v.data = d
	v.section = s
	return nil
}

func (a *verifyReceiver) Data(s string, xcheck []uint8) error {
	decoded, _, addr, err := anticipation.DecodeAndCheckStringToBytes(s)
	if err != nil {
		return err
	}
	trueAddress := int(a.current+uint64(addr)) - int(a.section.Addr)

	dataBlob := decoded[4 : len(decoded)-1]
	if trueAddress+len(dataBlob) > len(a.data) {
		return errors.New(fmt.Sprintf("impossible address %08x for sect %s since data is only %08x long",
			trueAddress+len(dataBlob), a.section.Name, len(a.data)))
	}
	for i := 0; i < len(dataBlob); i++ {
		var reference byte
		if a.section.Type&elf.SHT_NOBITS > 0 {
			reference = 0 //bss segment, so it's just zero
		} else {
			reference = a.data[trueAddress+i] //from disk
		}
		if reference != dataBlob[i] { //from decode?{
			return errors.New(fmt.Sprintf("byte number 0x%08x differs between elf data (%02x) and decoded data from string(%02x)",
				trueAddress+i, reference, dataBlob[i]))
		}
		if reference != xcheck[i] {
			return errors.New(fmt.Sprintf("byte %08x differs between elf data (%02x) and cross check data provided(%02x)",
				trueAddress+i, reference, xcheck[i]))
		}
	}
	return nil
}

func (a *verifyReceiver) DataInflate(s string, size uint16) error {
	return nil
}
func (a *verifyReceiver) EntryPoint(s string, size uint32) error {
	return nil
}
func (v *verifyReceiver) BaseAddrESA(s string, addr uint32) error {
	v.current = uint64(addr)
	return nil
}
func (v *verifyReceiver) BaseAddrELA(s string, addr uint32) error {
	prev := v.current & 0xffff_ffff_0000_0000
	v.current = prev | uint64(addr)
	return nil
}
func (v *verifyReceiver) ExtensionUnixTime(s string, size uint32) error {
	return nil
}
func (v *verifyReceiver) Read(buffer []byte) (string, error) { //just update to next
	buffer[0] = '.'
	return string(buffer[0:1]), nil

}
func (v *verifyReceiver) EOF() (string, error) {
	return EOFLine, nil
}
