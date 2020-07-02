package anticipation

import (
	"bytes"
	"errors"
	"fmt"
	"log"
)

//needs to be all 1s on right, can't be larger than 255
const FileXFerDataLineSize = uint16(0xff)

type EncodeDecodeError struct {
	s string
}

func NewEncodeDecodeError(s string) error {
	return &EncodeDecodeError{s}
}
func (d *EncodeDecodeError) Error() string {
	return d.s
}

//
// We used to implement all but one of the Hex line types, but this required so
// many changes and workarounds we gave up and just made this protocol suitable
// for a 64bit world.  Trying to shoehorn 64 bit addresses into a protocol that
// had to be extended to support 32bits was a bridge too far...
//
type HexLineType int

const (
	DataLine               HexLineType = 0
	EndOfFile              HexLineType = 1
	StartLinearAddress     HexLineType = 5
	ExtensionSetParameters HexLineType = 0x80
	ExtensionSetDeviceTime HexLineType = 0x81
)

func (hlt HexLineType) String() string {
	switch hlt {
	case DataLine:
		return "DataLine"
	case EndOfFile:
		return "EndOfFile"
	case StartLinearAddress:
		return "StartLinearAddress"
	case ExtensionSetParameters:
		return "ExtensionSetParametersTime"
	case ExtensionSetDeviceTime:
		return "ExtensionSetDeviceTime"
	}
	return "unknown"
}

func hexLineTypeFromInt(i int) HexLineType {
	switch i {
	case 0:
		return DataLine
	case 1:
		return EndOfFile
	case 5:
		return StartLinearAddress
	case 0x80:
		return ExtensionSetParameters
	case 0x81:
		return ExtensionSetDeviceTime
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
		l := uint64(converted[0]) //number of DATA bytes, not total size
		addr := decode64bitAddress(converted)
		for i := uint64(0); i < l; i++ {
			val := converted[i+10]
			if !bb.Write(addr+i, val) {
				return true, false
			}
		}
		return false, false
	case EndOfFile:
		return false, true
	case ExtensionSetParameters: //4 64 bit integers
		length := converted[0]
		if length != 32 {
			print("!extension parameters must be exactly 32 bytes, but was :", length, "\n")
			return true, false
		}
		for i := 0; i < 4; i++ {
			value := uint64(0)
			for p := 7; p >= 0; p-- {
				placeValue := uint64(1 << (8 * p))
				//10 is because of constant valuesat left of converted[]
				//i*8 is which param
				//7-p is byte
				value += (placeValue * uint64(converted[(10)+(i*8)+(7-p)]))
			}
			print("set parameter ", i, " ", value, "\n")
			bb.SetParameter(i, value)
		}
		return false, false
	case StartLinearAddress: //64 bit addr
		length := converted[0]
		if length != 0 {
			print("!unexpected length byte on SLA:", length)
			return true, false
		}
		if len(converted) != 11 { //size 1, 8 addr, 1 type, 1 checksum
			print("!SLA value has wrong byte count:", len(converted), "\n")
			return true, false
		}
		addr := decode64bitAddress(converted)
		bb.SetEntryPoint(addr)
		return false, false
	}

	print("!unable to understand line type [processLine]\n")
	return false, true
}

// take in a string and return either an exception or a well formed value
func DecodeAndCheckStringToBytes(s string) ([]byte, HexLineType, uint64, error) {
	lenAs16 := uint16(len(s))
	converted := ConvertBuffer(lenAs16, []byte(s))
	if converted == nil {
		return nil, HexLineType(0), 0, errors.New("convert buffer failed")
	}
	var addr uint64
	lt, ok := ExtractLineType(converted)
	if !ok {
		return nil, DataLine, 0, NewEncodeDecodeError(fmt.Sprintf("unable to extract line type from: %s", s))
	}
	if lt == DataLine {
		addr = decode64bitAddress(converted)
	}
	if ok := ValidBufferLength(lenAs16, converted); ok == false {
		return nil, lt, addr, NewEncodeDecodeError(fmt.Sprintf("expected buffer length to be ok, but wasn't: %s", s))
	}
	if ok := CheckChecksum(lenAs16, converted); ok == false {
		return nil, lt, addr, NewEncodeDecodeError(fmt.Sprintf("expected checksum to be ok, but wasn't:%s", s))
	}
	return converted, lt, addr, nil
}

// received a line, check that it has a hope of being syntactically correct
func ValidBufferLength(l uint16, converted []byte) bool {
	total := uint16(23) //size of just framing in characters (colon, 2 len chars, 16 addr chars, 2 type chars, 2 checksum chars)
	if uint16(l) < total {
		print("!bad buffer length, can't be smaller than", total, ":", l, "\n")
		return false
	}
	total += uint16(converted[0]) * 2
	if l != total {
		print("!bad buffer length, expected ", total, " but got", l, " based on ", total*2, "\n")
		return false
	}
	return true
}

// verify line's checksum
func CheckChecksum(l uint16, converted []byte) bool {
	sum := uint64(0)
	limit := (l - 1) / 2
	for i := uint16(0); i < limit; i++ {
		sum += uint64(converted[i])
	}
	complement := ^sum
	complement++
	checksum := uint8(complement & 0xff)
	if checksum != 0 {
		print("!bad checksum! expected 0 and got ", checksum,
			" from declared checksum of ", converted[limit-1], "\n")
		return false
	}
	return true
}

// extract the line type, 00 (data), 01 (eof), etc
func ExtractLineType(converted []byte) (HexLineType, bool) {
	switch converted[9] {
	case 0:
		return DataLine, true
	case 1:
		return EndOfFile, true
	case 5:
		return StartLinearAddress, true
	case 0x80:
		return ExtensionSetParameters, true
	case 0x81:
		return ExtensionSetDeviceTime, true
	default:
		print("!bad hex format line type:", converted[3], "\n")
		return DataLine, false
	}
}

// change buffer of ascii->converted bytes by taking the ascii values (2 per byte) and making them proper bytes
func ConvertBuffer(l uint16, raw []byte) []byte {
	//l-1 because the : is skipped so the remaining number of characters must be even
	if (l-1)%2 == 1 {
		print("!bad payload, expected even number of hex bytes but got:", l-1, "\n")
		return nil
	}
	converted := make([]byte, (l-1)/2)
	//skip first colon
	for i := uint16(1); i < l; i += 2 {
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
func bufferValue(index uint16, buffer []byte) (uint8, bool) {
	i := int(index)
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

func decode64bitAddress(decoded []byte) uint64 {
	total := uint64(0)
	pos := 1                  //where in the sequence
	for p := 7; p >= 0; p-- { //place value shift required
		pv := uint64(1 << (p * 8))
		total += uint64(decoded[pos]) * pv
		pos++
	}
	return total
}

///////////////////////////////////////////////////////////////////////////////////
// ENCODING
///////////////////////////////////////////////////////////////////////////////////

func EncodeDataBytes(raw []byte, offset uint64) string {
	if len(raw) > 255 {
		log.Fatalf("intel hex format only allows 2 hex characters for the size\n"+
			"of a data buffer, it can't be more than 0xff bytes (you have %x)", len(raw))
	}
	buf := bytes.Buffer{}
	buf.WriteString(fmt.Sprintf(":%02X%016X%02X", len(raw), offset, int(DataLine)))
	for _, b := range raw {
		buf.WriteString(fmt.Sprintf("%02x", b))
	}
	cs := createChecksum(raw, offset, DataLine)
	buf.WriteString(fmt.Sprintf("%02X", cs))
	return buf.String()
}
func EncodeStartLinearAddress(offset uint64) string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf(":00%016X%02X", offset,
		int(StartLinearAddress)))
	cs := createChecksum(nil, offset, StartLinearAddress)
	buf.WriteString(fmt.Sprintf("%02X", cs))
	return buf.String()
}

// this takes 4 64 bit integers (32 bytes)
func EncodeExtensionSetParameters(v [4]uint64) string {
	buf := bytes.Buffer{}
	valueBuffer := bytes.Buffer{} //for checksum ease
	buf.WriteString(fmt.Sprintf(":200000000000000000%02X", int(ExtensionSetParameters)))
	for i := 0; i < 4; i++ {
		value := v[i]
		for p := 7; p >= 0; p-- {
			b := byte((value >> (p * 8)) & 0xff)
			valueBuffer.WriteByte(b)
		}
		buf.WriteString(fmt.Sprintf("%016X", value))
	}
	cs := createChecksum(valueBuffer.Bytes(), 0, ExtensionSetParameters)
	buf.WriteString(fmt.Sprintf("%02X", cs))
	return buf.String()
}

func createChecksum(dataBytes []byte, offset uint64, lineType HexLineType) uint8 {
	sum := uint64(len(dataBytes)) //represents first byte of packet
	ct := 1
	for i := 7; i >= 0; i-- { //the address, next 8 bytes
		v := offset & uint64((0xff)<<(i*8))
		sum += (v >> (i * 8))
		ct++
	}
	sum += uint64(lineType)
	for _, v := range dataBytes {
		sum += uint64(v)
	}
	sum = ^sum
	sum += 1
	sum = sum & 0xff
	return uint8(sum)
}
