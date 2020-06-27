package anticipation

import (
	"strings"
	"testing"
)

func TestGoodLines(t *testing.T) {
	checkPerfectLine(t, ":0B0010006164647265737320676170A7", DataLine)
	checkPerfectLine(t, ":00000001FF", EndOfFile)
	checkPerfectLine(t, ":020000021200EA", ExtendedSegmentAddress)
	checkPerfectLine(t, ":10010000214601360121470136007EFE09D2190140", DataLine)
	checkPerfectLine(t, ":00000001FF", EndOfFile)
	checkPerfectLine(t, ":04000005000000CD2A", StartLinearAddress)
	checkPerfectLine(t, ":200000800001020304050607101112131415161720212223242526273031323334353637F0", ExtensionSetParameters)
}

func TestEndToEnd(t *testing.T) {
	values := []byte{97, 100, 100, 114, 101, 115, 115, 32, 103, 97, 112}
	bb := newFakeByteBuster(values, 0x10)

	if bb.EntryPointIsSet() {
		t.Errorf("entry point should not be changed at start")
	}

	esa := ":020000021200EA"
	checkAllAndProcess(t, esa, ExtendedSegmentAddress, bb, 0x12000, false) //base ptr moved from 2000 to 1200
	if bb.written != 0 {
		t.Errorf("should not write bytes on an ESA line")
	}
	if bb.EntryPointIsSet() {
		t.Errorf("entry point should not be set yet")
	}

	gw := ":0B0010006164647265737320676170A7"
	checkAllAndProcess(t, gw, DataLine, bb, 0x12000, true) //unchanged after a data line
	elastr := ":02000004FC0AF4"
	checkAllAndProcess(t, elastr, ExtendedLinearAddress, bb, 0xFC0A0000, false) //unchanged after a data line
	bigentryPoint := ":04000082DEADBEEF42"
	if bb.EntryPointIsSet() {
		t.Errorf("entry point should not be set yet!")
	}
	checkAllAndProcess(t, bigentryPoint, ExtensionBigEntryPoint, bb, 0xFC0A0000, false) //unchanged after a data line
	entryPoint := ":04000005000000CD2A"
	checkAllAndProcess(t, entryPoint, StartLinearAddress, bb, 0xFC0A0000, false) //unchanged after a data line
	if !bb.EntryPointIsSet() {
		t.Errorf("we set the entry point with an SLA but it is not visible in EntryPointIsSet")
	}
	if bb.EntryPoint() != 0xDEADBEEF000000CD {
		t.Errorf("expected entry point 0xDEADBEEF000000CD because of BigEntryPoint and SLA but got %08x", bb.EntryPoint())
	}
}

func checkAllAndProcess(t *testing.T, t1 string, hlt HexLineType, bb *fakeByteBuster, finalBase uint64, bytesChanged bool) {
	t.Helper()
	converted, lt, _, err := DecodeAndCheckStringToBytes(t1)
	if err != nil {
		t.Errorf("unable to decode string %s: %v", t1, err)
		return
	}
	hadError, isEnd := ProcessLine(lt, converted, bb)
	if hadError {
		t.Errorf("expected to not have any errors, but did in good line")
	}
	if isEnd {
		t.Errorf("expected to not be at end after a data line")
	}
	if bb.BaseAddress() != finalBase {
		t.Errorf("base pointer not as expected after line (expected %x but got %x)", finalBase, bb.BaseAddress())
	}
	if bytesChanged {
		if !bb.FinishedOk() {
			t.Errorf("wrong number of bytes written")
		}
	}
}

func TestBadChecksum(t *testing.T) {
	bcs := ":10010000214601360121470136007EFE09D2190149"
	converted := ConvertBuffer(uint16(len(bcs)), []byte(bcs))
	if CheckChecksum(uint16(len(bcs)), converted) {
		t.Errorf("expected to have a bad checksum, but didn't")
	}

}

func TestMissingChar(t *testing.T) {
	mc := ":10010000214601360121470136007EFE09D190140"
	converted := ConvertBuffer(uint16(len(mc)), []byte(mc))
	if converted != nil {
		t.Errorf("expected to have a bad conversion input length because removed a '2', but didn't")
	}
}

func TestAddressTooLow(t *testing.T) {
	atl := ":020000021200EA"
	converted := ConvertBuffer(uint16(len(atl)), []byte(atl))
	if !ValidBufferLength(uint16(len(atl)), converted) {
		t.Errorf("expected to have ok buffer length, courtesy of wikipedia")
	}

}

func checkPerfectLine(t *testing.T, t1 string, ltype HexLineType) {
	t.Helper()
	converted := ConvertBuffer(uint16(len(t1)), []byte(t1))

	if converted == nil {
		t.Error("expected t1 to convert correctly (from wikipedia)")
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
	expected := ":06123400010200000003ae"
	if s != expected {
		t.Errorf("expected %s but got %s", expected, s)
	}
}

func TestELAEncoding(t *testing.T) {
	newAddr := uint16(0xffff)

	s := EncodeELA(newAddr)
	s = strings.ToLower(s)
	expected := ":02000004fffffc"
	if s != expected {
		t.Errorf("expected %s but got %s", expected, s)
	}
}

func TestESAEncoding(t *testing.T) {
	newAddr := uint16(0x3456)

	s := EncodeESA(newAddr)
	s = strings.ToLower(s)
	expected := ":02000002345672"
	if s != expected {
		t.Errorf("expected %s but got %s", expected, s)
	}
}
func TestSLAEncoding(t *testing.T) {
	newAddr := uint32(0x000000CD)

	s := EncodeSLA(newAddr)
	s = strings.ToLower(s)
	expected := ":04000005000000cd2a"
	if s != expected {
		t.Errorf("expected %s but got %s", expected, s)
	}
}
func TestParameterEncoding(t *testing.T) {
	var v [4]uint64
	for i := 0; i < 4; i++ {
		v[i] = (0xff << (i * 8))
	}
	s := EncodeExtensionSetParameters(v)
	s = strings.ToLower(s)
	expected := ":4000008000000000000000ff000000000000ff000000000000ff000000000000ff00000061"
	if s != expected {
		t.Errorf("expected %s", expected)
		t.Errorf(" but got %s", s)
	}
}
