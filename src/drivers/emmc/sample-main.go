package emmc

/*
//export raw_exception_handler
func rawExceptionHandler(t uint64, esr uint64, addr uint64, el uint64, procId uint64) {
	trust.Errorf("interrupt: type=%d, esr=%x, addr=%x, el=%d,  procId=%d",
		t, esr, addr, el, procId)
	for {
		arm.Asm("nop")
	}
}
*/

/*
func sampleMain() {
	machine.MiniUART = machine.NewUART()
	_ = machine.MiniUART.Configure(&machine.UARTConfig{  })

	buffer := make([]byte, 512)
	//for now, hold the buffers on stack
	sectorCache := make([]byte, 0x200<<6) //0x40 pages
	sectorBitSet := make([]uint64, 1)
	trust.DefaultLogger.SetLevel(trust.EverythingButDebug)

	//raw init of interface
	if emmcinit() != 0 {
		trust.Errorf("Unable init emmc interface")
		machine.Abort()
	}
	// set the clock to the init speed (slow) and set some flags so
	// we will be ready for proper init
	emmcenable()

	if err := sdfullinit(); err != EmmcOk {
		trust.Errorf("Unable to do a full initialization of the EMMC interafce, aborting")
		machine.Abort()
	}

	sdcard, err := fatGetPartition(buffer) //data read into this buffer
	if err != EmmcOk {
		trust.Errorf("Unable to read MBR or unable to parse BIOS parameter block")
		machine.Abort()
	}

	tranq := NewTraquilBufferManager(unsafe.Pointer(&sectorCache[0]), 0x40,
		unsafe.Pointer(&sectorBitSet[0]), nil, nil)
	fs := NewFAT32Filesystem(tranq, sdcard)
	path := "/motd-news"
	rd, err := fs.Open(path)
	if err != EmmcOk {
		trust.Errorf("unable to open path: %s: %v", path, err)
		machine.Abort()
	}
	readerBuf := make([]uint8, 256)
	builder := strings.Builder{}
	for {
		n, err := rd.Read(readerBuf)
		if err == io.EOF {
			break
		}
		if err != EmmcOk {
			trust.Errorf("failed reading file: %s", err.Error())
			machine.Abort()
		}
		if n == 0 {
			continue
		}
		if _, err := builder.Write(readerBuf[:n]); err != nil {
			trust.Errorf("failed to write to builder: %v", err)
			machine.Abort()
		}
		fmt.Printf(builder.String())
	}
	machine.Abort()
}
*/
