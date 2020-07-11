package emmc

import (
	"os"
	"strings"
	"time"
)

type PosixErrorName uint32

const (
	None   PosixErrorName = 0
	Access PosixErrorName = 13
	BadF   PosixErrorName = 9
	MFile  PosixErrorName = 24
	NFile  PosixErrorName = 23
	NoEnt  PosixErrorName = 2
	NoMem  PosixErrorName = 12
	NotDir PosixErrorName = 20

	Unknown PosixErrorName = 0xffff
)

func (p PosixErrorName) String() string {
	var s string
	switch p {
	case None:
		s = "ENONE"
	case Access:
		s = "EACCES"
	case MFile:
		s = "EMFILE"
	case BadF:
		s = "EBADF"
	case NFile:
		s = "ENFILE"
	case NoEnt:
		s = "ENOENT"
	case NoMem:
		s = "ENOMEM"
	case NotDir:
		s = "ENOTDIR"
	case Unknown:
		s = "EUNKNOWN"
	default:
		s = "EUNKNOWN"
	}
	return s
}

var posixErrorTable = map[PosixErrorName]string{
	None:    "No error",
	Access:  "Permission denied",
	MFile:   "Too many open files",
	BadF:    "Bad file number",
	NFile:   "File table overflow",
	NoEnt:   "No such file or directory",
	NoMem:   "Out of memory",
	NotDir:  "Not a directory",
	Unknown: "Unknown error occurred.",
}

type PosixError struct {
	errno PosixErrorName
}

func (p *PosixError) Error() string {
	return posixErrorTable[p.errno]
}
func (p *PosixError) ErrorAbbreviation() string {
	return p.errno.String()
}

var (
	ENone    = &PosixError{None}
	EAccess  = &PosixError{Access}
	EMFile   = &PosixError{MFile}
	EBadF    = &PosixError{BadF}
	ENFile   = &PosixError{NFile}
	ENoEnt   = &PosixError{NoEnt}
	ENoMem   = &PosixError{NoMem}
	ENotDir  = &PosixError{NotDir}
	EUnknown = &PosixError{Unknown}
)

type DirEntType uint8

const (
	DirEntUnknown   DirEntType = 0
	DirEntBlock     DirEntType = 1
	DirEntCharacter DirEntType = 2
	DirEntDirectory DirEntType = 3
	DirEntFIFO      DirEntType = 4
	DirEntSymLink   DirEntType = 5
	DirEntRegular   DirEntType = 6
	DirEntSocket    DirEntType = 7
)

type Dir struct {
	fs       *FAT32Filesystem
	sector   sectorNumber
	inode    inodeNumber
	path     string
	contents []dirEnt
}

type dirEnt struct {
	name           string
	lastWrite      time.Time
	LastAccess     time.Time
	Create         time.Time
	isDir          bool
	size           uint32
	Path           string
	Inode          inodeNumber
	firstClusterLo uint16
	firstClusterHi uint16
}

func (d *dirEnt) Name() string {
	return d.name
}
func (d *dirEnt) Size() uint64 {
	return uint64(d.size)
}
func (d *dirEnt) ModTime() time.Time {
	return d.lastWrite
}
func (d *dirEnt) IsDir() bool {
	return d.isDir
}

func NewDir(fs *FAT32Filesystem, path string, sector sectorNumber, sizeHint int) *Dir {
	result := &Dir{
		fs:       fs,
		sector:   sector,
		contents: make([]dirEnt, 0, sizeHint),
		path:     path,
	}
	inode, ok := fs.inodeMap[path]
	if !ok {
		inode = fs.NewInode()
		fs.inodeMap[path] = inode
	}
	result.inode = inode
	return result
}

func (d *Dir) addEntry(longName string, raw *rawDirEnt) {
	if len(d.contents) == cap(d.contents) {
		//ugh, copy
		tmp := make([]dirEnt, 0, cap(d.contents)*2)
		tmp = append(tmp, d.contents...)
		d.contents = tmp
	}
	yr := int(((raw.CreateDate >> 9) & 0x7f) + 1980)
	mon := int((raw.CreateDate >> 5) & 0xf)
	day := int(raw.CreateDate & 0x1f)

	hr := (raw.CreateTime >> 11) & 0x1f
	min := (raw.CreateTime >> 5) & 0x3f
	sec := raw.CreateTime & 0x3f
	create := time.Date(yr, time.Month(mon), day, int(hr), int(min), int(sec), 0, time.UTC)

	yr = int(((raw.LastAccessDate >> 9) & 0x7f) + 1980)
	mon = int((raw.LastAccessDate >> 5) & 0xf)
	day = int(raw.LastAccessDate & 0x1f)
	access := time.Date(yr, time.Month(mon), day, int(hr), int(min), int(sec), 0, time.UTC)

	yr = int(((raw.WriteDate >> 9) & 0x7f) + 1980)
	mon = int((raw.WriteDate >> 5) & 0xf)
	day = int(raw.WriteDate & 0x1f)

	hr = (raw.WriteTime >> 11) & 0x1f
	min = (raw.WriteTime >> 5) & 0x3f
	sec = raw.WriteTime & 0x3f
	write := time.Date(yr, time.Month(mon), day, 0, 0, 0, 0, time.UTC)

	isDir := false
	if raw.Attrib&attributeSubdirectory != 0 {
		isDir = true
	}

	entry := dirEnt{
		name:           longName,
		isDir:          isDir,
		size:           raw.Size,
		Create:         create,
		lastWrite:      write,
		LastAccess:     access,
		firstClusterLo: raw.FirstClusterLo,
		firstClusterHi: raw.FirstClusterHi,
	}

	entryPath := strings.Join([]string{d.path, longName}, string(os.PathSeparator))
	entry.Path = entryPath
	inode, ok := d.fs.inodeMap[entryPath]
	if !ok {
		inode = d.fs.NewInode()
		d.fs.inodeMap[entryPath] = inode
	}
	entry.Inode = inode
	d.contents = append(d.contents, entry)
}
