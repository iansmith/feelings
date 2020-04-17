package main

import (
	rt "feelings/src/tinygo_runtime"
)

// tricky: you can't use testing.T here because of the all the linker and assembly
// shenanigans. You have to call this from the actual console code to do the test.
func sprintfGoodCases() bool {
	c := &ConsoleImpl{}
	good := true
	good = good && checkEquality(c.Sprintf("%d", 12), "12")
	good = good && checkEquality(c.Sprintf("%d", -7), "-7")
	good = good && checkEquality(c.Sprintf("%04d", -7), "-007")
	good = good && checkEquality(c.Sprintf("0x%04x", 63), "0x003f")
	good = good && checkEquality(c.Sprintf("0x%-4x", 63), "0x3f  ")
	good = good && checkEquality(c.Sprintf("%-10d", 63), "63        ")
	good = good && checkEquality(c.Sprintf("%4d", 63), "  63")
	good = good && checkEquality(c.Sprintf("%08d", 63), "00000063")
	good = good && checkEquality(c.Sprintf("%-08d", 63), "00000063")
	good = good && checkEquality(c.Sprintf("%s", "foo"), "foo")
	good = good && checkEquality(c.Sprintf("%6s", "foo"), "   foo")
	good = good && checkEquality(c.Sprintf("%-6s", "foo"), "foo   ")
	good = good && checkEquality(c.Sprintf("%-1s", "foo"), "foo")

	good = good && checkEquality(c.Sprintf("%10v", "foo"), "       foo")
	good = good && checkEquality(c.Sprintf("%10d", 22122), "     22122")
	good = good && checkEquality(c.Sprintf("%10d", -22122), "    -22122")
	good = good && checkEquality(c.Sprintf("%v", int64(12345678)), "12345678")
	good = good && checkEquality(c.Sprintf("%10v", int64(12345678)), "  12345678")
	good = good && checkEquality(c.Sprintf("%-10v", int64(-12345678)), "-12345678 ")
	good = good && checkEquality(c.Sprintf("%-10v", uint32(0x12345678)), "12345678  ")
	good = good && checkEquality(c.Sprintf("%-8v", uint32(65535)), "ffff    ")
	good = good && checkEquality(c.Sprintf("%8v", "baz"), "     baz")
	return good
}

func checkEquality(formatted string, expected string) bool {
	if formatted != expected {
		rt.MiniUART.WriteString("expected '" + expected + "' but got '" + formatted + "'")
		rt.MiniUART.WriteCR()
		return false
	}
	return true
}
