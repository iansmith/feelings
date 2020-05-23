package sysdec

import (
	"fmt"
	"strings"
)

type AccessDef struct {
	read  bool
	write bool
	isSet bool //did they explictly set the field
}

func (a AccessDef) CanRead() bool {
	return a.read
}
func (a AccessDef) CanWrite() bool {
	return a.write
}
func (a AccessDef) IsSet() bool {
	return a.isSet
}
func Access(s string) AccessDef {
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	var a AccessDef
	switch s {
	case "": //do nothing
	case "r":
		a.read = true
		a.isSet = true
	case "w":
		a.write = true
		a.isSet = true
	case "rw":
		a.write = true
		a.read = true
		a.isSet = true
	default:
		panic("unable to understand Access value:" + s)
	}
	return a
}

func (a AccessDef) String() string {
	result := ""
	if a.read && a.write {
		result = "{read=true,write=true}"
	} else if a.write {
		result = "{read=false,write=true}"
	} else if a.read {
	}
	return result
}

type BitRangeDef struct {
	Lsb int
	Msb int
}

func (b *BitRangeDef) String() string {
	return fmt.Sprintf("[%d:%d]", b.Msb, b.Lsb)
}
func (b *BitRangeDef) Width() int {
	return (b.Msb - b.Lsb) + 1
}
func BitRange(Msb int, Lsb int) BitRangeDef {
	if Msb > 63 || Lsb > 63 || Msb < 0 || Lsb < 0 {
		panic("BitRange value for Msb/Lsb out of range")
	}
	if Msb < Lsb {
		panic("BitRange Msb < Lsb")
	}
	return BitRangeDef{Msb: Msb, Lsb: Lsb}
}
