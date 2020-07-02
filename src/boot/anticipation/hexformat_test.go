package anticipation

import (
	"fmt"
	"strings"
	"testing"
)

func TestGoodLines(t *testing.T) {
	checkPerfectLine(t, ":0B0000000000000010006164647265737320676170A7", DataLine)
	checkPerfectLine(t, ":00000000000000000001FF", EndOfFile)
	checkPerfectLine(t, ":00010203040506070805D7", StartLinearAddress)
}

func TestEncodeStartLinearAddress(t *testing.T) {
	result := EncodeStartLinearAddress(0x01020304050607FF)
	if result != ":0001020304050607FF05E0" {
		t.Errorf("expected :0001020304050607FF05E0")
		t.Logf("but got  %s", result)
	}
}
func TestBadChecksum(t *testing.T) {
	bcs := ":10000000000000010000214601360121470136007EFE09D2190149"
	converted := ConvertBuffer(uint16(len(bcs)), []byte(bcs))
	t.Logf("expecting to see 'bad checksum'")
	if CheckChecksum(uint16(len(bcs)), converted) {
		t.Errorf("expected to have a bad checksum, but didn't")
	}

}

func TestMissingChar(t *testing.T) {
	mc := ":10000000000000010000214601360121470136007EFE09D190140"
	t.Logf("expecting to see 'bad payload'")
	converted := ConvertBuffer(uint16(len(mc)), []byte(mc))
	if converted != nil {
		t.Errorf("expected to have a bad conversion input length because removed a '2', but didn't")
	}
}

func checkPerfectLine(t *testing.T, t1 string, ltype HexLineType) {
	t.Helper()
	converted := ConvertBuffer(uint16(len(t1)), []byte(t1))

	if converted == nil {
		t.Errorf("expected line to convert correctly: %s", t1)
	}

	lt, ok := ExtractLineType(converted)
	if !checkLineType(t, lt, ok, ltype, true) {
		return
	}
	if ok := ValidBufferLength(uint16(len(t1)), converted); ok == false {
		t.Error("expected buffer length to be ok, but wasn't")
	}
	if ok := CheckChecksum(uint16(len(t1)), converted); ok == false {
		t.Error("expected checksum to be ok, but wasn't")
	}
}

func checkLineType(t *testing.T, lt HexLineType, ok bool, expectedLt HexLineType, expectedOk bool) bool {
	t.Helper()
	if ok != expectedOk {
		t.Error("expected lineType ok to be ", expectedOk, " but was ", ok)
		return false
	}
	if lt != expectedLt {
		t.Errorf("bad line type, expected "+expectedLt.String()+" but got %s", lt.String())
		return false
	}
	return true
}

func TestDataEncoding(t *testing.T) {
	data := []byte{0x01, 0x02, 00, 00, 00, 0x03}
	s := EncodeDataBytes(data, 0x1234)
	s = strings.ToLower(s)
	expected := ":06000000000000123400010200000003ae"
	if s != expected {
		t.Errorf("expected %s but got %s", expected, s)
	}
}

func TestParameterEncoding(t *testing.T) {
	var v [4]uint64
	for i := 0; i < 4; i++ {
		v[i] = 0xff << (i * 8)
	}
	s := EncodeExtensionSetParameters(v)
	s = strings.ToLower(s)
	p1 := "00000000000000ff"
	p2 := "000000000000ff00"
	p3 := "0000000000ff0000"
	p4 := "00000000ff000000"
	expected := fmt.Sprintf(":20000000000000000080%s%s%s%s64", p1, p2, p3, p4)
	if s != expected {
		t.Errorf("expected %s", expected)
		t.Errorf(" but got %s", s)
	}
}

func TestExampleParameterEncoding(t *testing.T) {
	ex := EncodeExtensionSetParameters([4]uint64{0xFFFFFC00300010D8, 0, 0, 0})
	s := ":20000000000000000080FFFFFC00300010D80000000000000000000000000000000000000000000000004E"
	if ex != s {
		t.Errorf("unable to even get encode correct in decode test: %s", ex)
		t.Errorf("                                        expected: %s", s)
	}
	_, lt, _, err := DecodeAndCheckStringToBytes(s)
	if err != nil {
		t.Errorf("failed to decode parameters properly: %s", err.Error())
	}
	if lt != ExtensionSetParameters {
		t.Errorf("wrong type extracted! expected set params but got %s", lt.String())
	}
}
