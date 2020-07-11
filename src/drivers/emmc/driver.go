package emmc

import (
	"io"
	"machine"
	"time"

	"lib/trust"

	"unsafe"
)

type EmmcFileInfo interface {
	Name() string
	Size() int64
	ModTime() time.Time
	IsDir() bool
}

type EmmcError int32

type EmmcFile interface {
	Read(b []byte) (int, error)
	ReadAt(b []byte, off int64) (int, error)
	Close() error
}

type Emmc interface {
	Init() int
	Open(filename string) (EmmcFile, error)
	ReadDir(path string) ([]EmmcFileInfo, error)
	WindUp() EmmcError
}

var Impl = &emmcImpl{}

const (
	EmmcOk                     EmmcError = 0
	EmmcNoInterface            EmmcError = -1
	EmmcNoStart                EmmcError = -2
	EmmcNoMBR                  EmmcError = -3
	EmmcBadInitialRead         EmmcError = -4
	EmmcNotFile                EmmcError = -5
	EmmcNoEnt                  EmmcError = -6
	EmmcIO                     EmmcError = -7
	EmmcBadArg                 EmmcError = -8
	EmmcDataInhibitTimeout     EmmcError = -9
	EmmcBadReadBlock           EmmcError = -10
	EmmcBadReadMultiBlock      EmmcError = -11
	EmmcNoDataReady            EmmcError = -12
	EmmcBadEMMCCommand         EmmcError = -13
	EmmcNoResponseBuffer       EmmcError = -14
	EmmcNoDataDone             EmmcError = -15
	EmmcOpCondTimeout          EmmcError = -16
	EmmcFailedCID              EmmcError = -17
	EmmcFailedRelativeAddr     EmmcError = -18
	EmmcFailedCSD              EmmcError = -19
	EmmcFailedSelectCard       EmmcError = -20
	EmmcFailedSetBlockLen      EmmcError = -21
	EmmcBadBIOSParamBlock      EmmcError = -22
	EmmcBadMBR                 EmmcError = -23
	EmmcNoMBRSignature         EmmcError = -24
	EmmcBadPartitions          EmmcError = -25
	EmmcBadFAT32BootSignature  EmmcError = -26
	EmmcBadFAT32FilesystemType EmmcError = -27
	EmmcBadFAT16BootSignature  EmmcError = -28
	EmmcBadFAT16FilesystemType EmmcError = -29
	EmmcEOF                    EmmcError = -30
	EmmcNoBuffer               EmmcError = -31
	EmmcAlreadyClosed          EmmcError = -32
	EmmcFailedReadIntoCache    EmmcError = -33
	EmmcNotDirectory           EmmcError = -34
	EmmcUnexpectedEOF          EmmcError = -35
	EmmcSeekFailed             EmmcError = -37
	EmmcUnknown                EmmcError = -37
)

func (e EmmcError) Error() string {
	return e.String()
}

func (e EmmcError) String() string {
	switch e {
	case 0:
		return "EmmcOk"
	case -1:
		return "EmmcNoInterface"
	case -2:
		return "EmmcNoStart"
	case -3:
		return "EmmcNoMBR"
	case -4:
		return "EmmcBadInitialRead"
	case -5:
		return "EmmcNotFile"
	case -6:
		return "EmmcNoEnt"
	case -7:
		return "EmmcIO"
	case -8:
		return "EmmcBadArg"
	case -9:
		return "EmmcDataInhibitTimeout"
	case -10:
		return "EmmcBadReadBlock"
	case -11:
		return "EmmcBadReadMultiBlock"
	case -12:
		return "EmmcNoDataReady"
	case -13:
		return "EmmcBadEMMCCommand"
	case -14:
		return "EmmcNoResponseBuffer"
	case -15:
		return "EmmcNoDataDone"
	case -16:
		return "EmmcOpCondTimeout"
	case -17:
		return "EmmcFailedCID"
	case -18:
		return "EmmcFailedRelativeAddr"
	case -19:
		return "EmmcFailedCSD"
	case -20:
		return "EmmcFailedSelectCard"
	case -21:
		return "EmmcFailedSetBlockLen"
	case -22:
		return "EmmcBadBIOSParamBlock"
	case -23:
		return "EmmcBadMBR"
	case -24:
		return "EmmcNoMBRSignature"
	case -25:
		return "EmmcBadPartitions"
	case -26:
		return "EmmcBadFAT32BootSignature"
	case -27:
		return "EmmcBadFAT32Filesystem"
	case -28:
		return "EmmcBadFAT16BootSignature"
	case -29:
		return "EmmcBadFAT16Filesystem"
	case -30:
		return "EmmcEOF"
	case -31:
		return "EmmcNoBuffer"
	case -32:
		return "EmmcAlreadyClosed"
	case -33:
		return "EmmcFailedReadIntoCache"
	case -34:
		return "EmmcNotDirectory"
	case -35:
		return "EmmcUnexpectedEOF"
	case -36:
		return "EmmcSeekFailed"
	case -37:
		return "EmmcUknown"
	}
	return "BadEmmcErrorValue"
}

type emmcImpl struct {
	fs *FAT32Filesystem
}

func (e *emmcImpl) Init() EmmcError {
	//for now, hold the buffers on heap
	sectorCache := make([]byte, sectorSize<<6) //0x40 pages
	sectorBitSet := make([]uint64, 1)

	//raw init of interface
	if emmcinit() != 0 {
		trust.Errorf("Unable init emmc interface")
		return EmmcNoInterface
	}
	// set the clock to the init speed (slow) and set some flags so
	// we will be ready for proper init
	emmcenable()

	if err := sdfullinit(); err != EmmcOk {
		trust.Errorf("Unable to do a full initialization of the EMMC interafce, aborting")
		return EmmcNoStart
	}
	mbrBuffer := make([]byte, sectorSize)
	sdcard, err := fatGetPartition(mbrBuffer) //data read into this buffer
	if err != EmmcOk {
		trust.Errorf("Unable to read MBR or unable to parse BIOS parameter block")
		return err
	}

	tranq := NewTraquilBufferManager(unsafe.Pointer(&sectorCache[0]), 0x40,
		unsafe.Pointer(&sectorBitSet[0]), nil, nil)
	e.fs = NewFAT32Filesystem(tranq, sdcard)
	return EmmcOk
}

func (e *emmcImpl) WindUp() {
	var resp [4]uint32
	err := emmccmd(0, 0, &resp) //tells it to "go idle"
	if err != EmmcOk {
		trust.Errorf("unable to shutdown: %d", err)
	}
}
func (e *emmcImpl) ReadDir(path string) ([]*dirEnt, error) {
	trust.Errorf("read dir called with %s... not implemented\n", path)
	machine.Abort()
	return nil, nil
}

func (e *emmcImpl) Open(path string) (EmmcFile, error) {
	fr, err := e.fs.Open(path)
	if err != EmmcOk {
		return nil, err
	}
	return &emmcFileImpl{fr}, nil
}

type emmcFileImpl struct {
	fr *fatDataReader
}

func (e *emmcFileImpl) Read(buf []byte) (int, error) {
	if e.fr == nil {
		return 0, EmmcAlreadyClosed
	}
	n, tmp := e.fr.Read(buf)
	if tmp == EmmcEOF {
		return n, io.EOF
	}
	if tmp == EmmcOk {
		return n, nil
	}
	return n, tmp
}

func (e *emmcFileImpl) ReadAt(buf []byte, off int64) (int, error) {
	if e.fr == nil {
		return 0, EmmcAlreadyClosed
	}
	n, tmp := e.fr.ReadAt(buf, off)
	//rule: if n<len(buf) then return an error about why
	if n < len(buf) && tmp == EmmcOk {
		trust.Errorf("ReadA: Violation of ReadAt contract! Short read, %d of %d "+
			"with no error", n, len(buf))
		machine.Abort()
	}
	if tmp == EmmcEOF {
		return n, io.EOF
	}
	if tmp == EmmcOk {
		return n, nil
	}
	return n, tmp
}

func (e *emmcFileImpl) Close() error {
	e.fr = nil
	//xxx should blow away cached pages of this file
	return nil
}
