package main

import (
	"debug/elf"
	"io"
	"unsafe"

	"device/arm"

	"boot/bootloader"
	"drivers/emmc"
	"lib/trust"
)

const maddie = "/feelings/madeleine"
const KernelPageSize = 0x10000

func canBootFromDisk(logger *trust.Logger) (emmc.EmmcFile, bool) {
	var err error
	var fp emmc.EmmcFile
	if emmc.Impl.Init() != emmc.EmmcOk {
		logger.Errorf("Unable to initialize EMMC driver, " +
			"booting from serial port...")
		return nil, false
	}
	fp, err = emmc.Impl.Open(maddie)
	if err != nil {
		logger.Errorf("Unable to find %s binary, "+
			"booting from serial port...", maddie)
		return nil, false

	}
	logger.Infof("found bootable lady: %s", maddie)
	return fp, true
}

// This does the work of loading the binary in pieces and then attaching
// it to pages.
func bootDisk(fp emmc.EmmcFile, logger *trust.Logger) {
	elfFile, err := elf.NewFile(fp)
	if err != nil {
		trust.Debugf("Error attaching elf reader: %v", err)
		return
	}
	logger.Debugf("entry point: %016x", elfFile.FileHeader.Entry)

	totalExcText := uint64(0)

	nameToSizeInPages := make(map[string]uint64)
	//figure out the page sizes for these
	for _, sect := range elfFile.Sections {
		switch sect.Name {
		case ".text", ".bss", ".data", ".rodata", ".exc":
			overhang := uint64(1)
			if sect.Size%KernelPageSize == 0 {
				overhang = 0
			}
			sizeInPages := (sect.Size / KernelPageSize) + overhang
			if sect.Name == ".text" || sect.Name == ".exc" {
				totalExcText += sizeInPages
			} else {
				nameToSizeInPages[sect.Name] = sizeInPages
			}
		}
	}
	overhang := uint64(1)
	if totalExcText%KernelPageSize == 0 {
		overhang = 0
	}
	nameToSizeInPages[".text"] = totalExcText/KernelPageSize + overhang
	for k, v := range nameToSizeInPages {
		logger.Debugf("%-12s:%d pages", k, v)
	}

	startPhys := bootloader.MadeleinePlacement
	currentPhys := uintptr(startPhys)
	for _, sectName := range []string{".exc", ".text", ".rodata", ".data", ".bss"} {
		section := elfFile.Section(sectName)
		logger.Debugf("loading %-10s @ 0x%x", sectName, currentPhys)
		read := uint64(0)
		for read < section.Size {
			reader := elfFile.Section(sectName).Open()
			l := sectionBufferSize
			if section.Size-read < sectionBufferSize {
				l = int(section.Size - read)
			}
			n, err := reader.Read(sectionBuffer[:l])
			if err != nil {
				if err == io.EOF {
					break
				}
				logger.Errorf("Unable to read section %s: %v", sectName, err)
				return
			}
			for i := 0; i < n; i++ {
				ptr := (*byte)(unsafe.Pointer(currentPhys + uintptr(i)))
				*ptr = sectionBuffer[i]
			}
			currentPhys += uintptr(n)
			read += uint64(n)
		}
		if sectName != ".exc" { //glom the .exc and the .text together
			if currentPhys%KernelPageSize != 0 {
				diff := KernelPageSize - (currentPhys % KernelPageSize)
				currentPhys += uintptr(diff)
			}
		}
	}
	for {
		arm.Asm("nop")
	}
}

const sectionBufferSize = 512

var sectionBuffer [sectionBufferSize]byte
