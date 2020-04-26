package trust

import (
	"feelings/src/golang/fmt"
	"feelings/src/lib/semihosting"
)

type MaskLevel int

const (
	Nothing   MaskLevel = 0x0
	ErrorMask MaskLevel = 0x1
	WarnMask  MaskLevel = 0x2
	InfoMask  MaskLevel = 0x4
	DebugMask MaskLevel = 0x8
	fatalMask MaskLevel = 0x80
)

var level = fatalMask | ErrorMask | WarnMask | InfoMask | DebugMask

// SetLevel lets you set an error mask directly. You can pass in something like
// ErrorMask | DebugMask to control exactly what gets printed.  It returns the
// previous mask.
func SetLevel(mask MaskLevel) MaskLevel {
	if mask&0xf == 0 {
		fmt.Printf(" WARN: trust.SetLevel is turning of log messages\n")
	}
	result := Nothing
	switch {
	case mask&ErrorMask > 0:
		result |= ErrorMask
		fallthrough
	case mask&WarnMask > 0:
		result |= WarnMask
		fallthrough
	case mask&InfoMask > 0:
		result |= InfoMask
		fallthrough
	case mask&DebugMask > 0:
		result |= DebugMask
	}
	r := level & 0xf
	level = result | fatalMask
	return r
}
func Level() MaskLevel {
	return level
}

func LevelToString() string {
	result := ""
	switch {
	case level&ErrorMask > 0:
		result += "error "
		fallthrough
	case level&WarnMask > 0:
		result += "warn "
		fallthrough
	case level&InfoMask > 0:
		result += "info "
		fallthrough
	case level&DebugMask > 0:
		result += "debug"
	}
	return result
}

func logf(l MaskLevel, format string, params ...interface{}) {
	if level&l == 0 {
		return
	}
	switch {
	case l&ErrorMask > 0:
		fmt.Printf("ERROR:")
	case level&WarnMask > 0:
		fmt.Printf(" WARN:")
	case level&InfoMask > 0:
		fmt.Printf(" INFO:")
	case level&DebugMask > 0:
		fmt.Printf("DEBUG:")
	}
	fmt.Printf(format, params...)
}

//Fatalf prints the given log message (format + params) on stdout and then
//exits with the exitCode provided.  Fatalf is not maskable.
func Fatalf(exitCode int, format string, params ...interface{}) {
	logf(fatalMask, format, params...)
	semihosting.Exit(uint64(exitCode))
}

//Errorf prints the given log message (format + params) using the ErrorMask level.
func Errorf(format string, params ...interface{}) {
	logf(ErrorMask, format, params...)
}

//Warnf prints the given log message (format + params) using the WarnMask level.
func Warnf(format string, params ...interface{}) {
	logf(WarnMask, format, params...)
}

//Infof prints the given log message (format + params) using the InfoMask level.
func Infof(format string, params ...interface{}) {
	logf(InfoMask, format, params...)
}

//Debugf prints the given log message (format + params) using the DebugMask level.
func Debugf(format string, params ...interface{}) {
	logf(DebugMask, format, params...)
}
