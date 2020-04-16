package anticipation

import (
	"bytes"
	"fmt"
)

//neds to be all 1s
const FileXFerDataLineSize = uint16(0x1ff)

type DecodeError struct {
	s string
}

func NewDecodeError(s string) error {
	return &DecodeError{s}
}
func (d *DecodeError) Error() string {
	return d.s
}

//
// We implement all the hexlinetypes except 3, which is some ancient
// X86 thing involving memory segments...
//
type HexLineType int

const (
	DataLine               HexLineType = 0
	EndOfFile              HexLineType = 1
	ExtendedSegmentAddress HexLineType = 2
	ExtendedLinearAddress  HexLineType = 4
	StartLinearAddress     HexLineType = 5
	ExtensionUnixTime      HexLineType = 0x80
)

func (hlt HexLineType) String() string {
	switch hlt {
	case DataLine:
		return "DataLine"
	case EndOfFile:
		return "EndOfFile"
	case ExtendedSegmentAddress:
		return "ExtendedSegmentAddress"
	case ExtendedLinearAddress:
		return "ExtendedLinearAddress"
	case StartLinearAddress:
		return "StartLinearAddress"
	case ExtensionUnixTime:
		return "ExtensionUnixTime"
	}
	return "unknown"
}

func hexLineTypeFromInt(i int) HexLineType {
	switch i {
	case 0:
		return DataLine
	case 1:
		return EndOfFile
	case 2:
		return ExtendedSegmentAddress
	case 4:
		return ExtendedLinearAddress
	case 5:
		return StartLinearAddress
	case 0x80:
		return ExtensionUnixTime
	}
	panic("!unable to understand line type\n")
}

///////////////////////////////////////////////////////////////////////////////////
// DECODE
///////////////////////////////////////////////////////////////////////////////////

// deal with a received hex line and return (error?,done?)
func ProcessLine(t HexLineType, converted []byte, bb byteBuster) (bool, bool) {
	switch t {
	case DataLine:
		l := converted[0]
		offset := (uint32(converted[1]) * 256) + (uint32(converted[2]))
		//baseAddr + value in the line => basePtr
		baseAddr := bb.BaseAddress() + offset
		var val uint8
		for i := uint32(0); i < uint32(l); i++ {
			addr := baseAddr + i
			val = converted[4+i]
			if !bb.Write(addr, val) {
				return true, false
			}
		}
		return false, false
	case EndOfFile:
		return false, true
	case ExtendedSegmentAddress: //16 bit addr
		length := converted[0]
		if length != 2 {
			print("!ESA value has too many bytes:", length, "\n")
			return true, false
		}
		esaAddr := uint32(converted[4])*256 + uint32(converted[5])
		esaAddr = esaAddr << 4 //it's assumed to be a multiple of 16
		bb.SetBaseAddr(esaAddr)
		print("#updating current base address to ", esaAddr, "\n")
		return false, false
	case ExtendedLinearAddress: //32 bit addr but only top 16 passed
		length := converted[0]
		if length != 2 {
			print("!ELA value has too many bytes:", length, "\n")
			return true, false
		}
		elaAddr := uint32(converted[4])*256 + uint32(converted[5])
		elaAddr = elaAddr << 16 //data supplied is high order 32
		bb.SetBaseAddr(elaAddr)
		print("#updating current base address to ", elaAddr, "\n")
		return false, false
	case ExtensionUnixTime: //32 bit int from epoch
		length := converted[0]
		if length != 4 {
			print("!extension time value has too many bytes:", length, "\n")
			return true, false
		}
		t := uint32(converted[4])*0x1000000 + uint32(converted[5])*0x10000 + uint32(converted[6])*0x100 + uint32(converted[7])
		bb.SetUnixTime(t)
		print("#setting current time to unix ", t, "\n")
		return false, false
	case StartLinearAddress: //32 bit addr
		length := converted[0]
		if length != 4 {
			print("!SLA value has too many bytes:", length, "\n")
			return true, false
		}
		slaAddr := uint32(converted[4])*0x1000000 + uint32(converted[5])*0x10000 + uint32(converted[6])*0x100 + uint32(converted[7])
		bb.SetEntryPoint(slaAddr)
		return false, false
	}

	print("!unable to understand line type [processLine]\n")
	return false, true
}

// received a line, check that it has a hope of being syntactically correct
func ValidBufferLength(l int, converted []byte) bool {
	total := uint8(11) //size of just framing in characters (colon, 2 len chars, 4 addr chars, 2 type chars, 2 checksum chars)
	if uint8(l) < total {
		print("!bad buffer length, can't be smaller than", total, ":", l, "\n")
		return false
	}
	total += converted[0] * 2
	if uint8(l) != total {
		print("!bad buffer length, expected", total, "but got", l, " based on ", converted[0], "\n")
		return false
	}
	return true
}

// verify line's checksum
func CheckChecksum(l int, converted []byte) bool {
	sum := uint64(0)
	limit := (l - 1) / 2
	for i := 0; i < limit; i++ {
		sum += uint64(converted[i])
	}
	complement := ^sum
	complement++
	checksum := uint8(complement & 0xff)
	if checksum != 0 {
		print("!bad checksum! expected 0 and got ", checksum, " from declared checksum of ", converted[limit-1], "\n")
		return false
	}
	return true
}

// extract the line type, 00 (data), 01 (eof), or 02 (esa) and (ok?)
func ExtractLineType(converted []byte) (HexLineType, bool) {
	switch converted[3] {
	case 0:
		return DataLine, true
	case 1:
		return EndOfFile, true
	case 2:
		return ExtendedSegmentAddress, true
	case 4:
		return ExtendedLinearAddress, true
	case 5:
		return StartLinearAddress, true
	case 0x80:
		return ExtensionUnixTime, true
	case 3:
		print("!unimplemented line type in hex transmission [StartSegmentAddress] ")
		return DataLine, false
	default:
		print("!bad buffer type:", converted[3], "\n")
		return DataLine, false
	}
}

// change buffer of ascii->converted bytes by taking the ascii values (2 per byte) and making them proper bytes
func ConvertBuffer(l int, raw []byte) []byte {
	//l-1 because the : is skipped so the remaining number of characters must be even
	if (l-1)%2 == 1 {
		print("!bad payload, expected even number of hex bytes (%d):", l)
		for i := 0; i < l; i++ {
			print(i, "=", raw[i], " ")
		}
		print("\n")
		return nil
	}
	converted := make([]byte, l)
	//skip first colon
	for i := 1; i < l; i += 2 {
		v, ok := bufferValue(i, raw)
		if !ok {
			return nil // they already sent the error to the other side
		}
		converted[(i-1)/2] = v
	}
	return converted
}

// this hits buffer[i] and buffer[i+1] to convert an ascii byte
// returns false to mean you had a bad character in the input
func bufferValue(i int, buffer []byte) (uint8, bool) {
	total := uint8(0)
	switch buffer[i] {
	case '0':
	case '1':
		total += 16 * 1
	case '2':
		total += 16 * 2
	case '3':
		total += 16 * 3
	case '4':
		total += 16 * 4
	case '5':
		total += 16 * 5
	case '6':
		total += 16 * 6
	case '7':
		total += 16 * 7
	case '8':
		total += 16 * 8
	case '9':
		total += 16 * 9
	case 'a', 'A':
		total += 16 * 10
	case 'b', 'B':
		total += 16 * 11
	case 'c', 'C':
		total += 16 * 12
	case 'd', 'D':
		total += 16 * 13
	case 'e', 'E':
		total += 16 * 14
	case 'f', 'F':
		total += 16 * 15
	default:
		print("!bad character in payload hi byte(number #", i, "):", buffer[i], "\n")
		return 0xff, false
	}
	switch buffer[i+1] {
	case '0':
	case '1':
		total++
	case '2':
		total += 2
	case '3':
		total += 3
	case '4':
		total += 4
	case '5':
		total += 5
	case '6':
		total += 6
	case '7':
		total += 7
	case '8':
		total += 8
	case '9':
		total += 9
	case 'a', 'A':
		total += 10
	case 'b', 'B':
		total += 11
	case 'c', 'C':
		total += 12
	case 'd', 'D':
		total += 13
	case 'e', 'E':
		total += 14
	case 'f', 'F':
		total += 15
	default:
		print("!bad character in payload low byte (number #", i+1, "):", buffer[i+1], "\n")
		return 0xff, false
	}
	return total, true
}

///////////////////////////////////////////////////////////////////////////////////
// ENCODING
///////////////////////////////////////////////////////////////////////////////////

func EncodeDataBytes(raw []byte, offset uint16) string {
	buf := bytes.Buffer{}
	buf.WriteString(fmt.Sprintf(":%02X%04X%02X", len(raw), offset, int(DataLine)))
	for _, b := range raw {
		buf.WriteString(fmt.Sprintf("%02x", b))
	}
	cs := createChecksum(raw, offset, DataLine)
	buf.WriteString(fmt.Sprintf("%02X", cs))
	return buf.String()
}

func EncodeSLA(entryPoint uint32) string {
	buf := bytes.Buffer{}
	raw := []byte{byte(entryPoint & 0xff000000 >> 24), byte(entryPoint & 0x00ff0000 >> 16),
		byte(entryPoint & 0x0000ff00 >> 8), byte(entryPoint & 0x000000ff)}
	buf.WriteString(fmt.Sprintf(":040000%02X%08X", int(StartLinearAddress), entryPoint))
	cs := createChecksum(raw, 0, StartLinearAddress)
	buf.WriteString(fmt.Sprintf("%02X", cs))
	return buf.String()
}

// only pass the most significant 16 bits of the 32 bit base
func EncodeELA(base uint16) string {
	buf := bytes.Buffer{}
	buf.WriteString(fmt.Sprintf(":020000%02X%04X", int(ExtendedLinearAddress), base))
	raw := []byte{byte(base & 0xff00 >> 8), byte(base & 0x00ff)}
	cs := createChecksum(raw, 0, ExtendedLinearAddress)
	buf.WriteString(fmt.Sprintf("%02X", cs))
	return buf.String()
}

// only pass the top 16 bits of 24 bit base
func EncodeESA(base uint16) string {
	buf := bytes.Buffer{}
	buf.WriteString(fmt.Sprintf(":020000%02X%02X", int(ExtendedSegmentAddress), base))
	raw := []byte{byte(base & 0xff00 >> 8), byte(base & 0x00ff)}
	cs := createChecksum(raw, 0, ExtendedSegmentAddress)
	buf.WriteString(fmt.Sprintf("%02X", cs))
	return buf.String()
}

// tricky: offset only used by the data packet since everything else has 0 offset (not used)
func createChecksum(raw []byte, offset uint16, hlt HexLineType) uint8 {
	sum := len(raw)
	sum += int(offset & 0xff)
	sum += int(offset>>8) & 0xff
	sum += int(hlt)
	for _, v := range raw {
		sum += int(v)
	}
	sum = ^sum
	sum += 1
	sum = sum & 0xff
	return uint8(sum)
}
