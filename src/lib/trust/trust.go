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
	StatsMask MaskLevel = 0x10
	fatalMask MaskLevel = 0x80
)

var level = fatalMask | StatsMask | ErrorMask | WarnMask | InfoMask | DebugMask

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
		fallthrough
	case mask&StatsMask > 0:
		result |= StatsMask
	}
	r := level & 0x1f
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
		result += "debug "
		fallthrough
	case level&StatsMask > 0:
		result += "stats"
	}
	return result
}

func logf(l MaskLevel, format string, params ...interface{}) {
	if level&l == 0 {
		return
	}
	start := 0
	switch {
	case l&ErrorMask > 0:
		fmt.Printf("ERROR:")
	case l&WarnMask > 0:
		fmt.Printf(" WARN:")
	case l&InfoMask > 0:
		fmt.Printf(" INFO:")
	case l&DebugMask > 0:
		fmt.Printf("DEBUG:")
	case l&StatsMask > 0:
		s, ok := params[0].(string)
		if !ok {
			s = "unknown"
		}
		fmt.Printf("len of params %d", len(params))
		fmt.Printf("STATS[%s]:", s)
		start = 1
	}
	if len(format) == 0 {
		format = "\n"
	} else if format[len(format)-1] != '\n' {
		format += "\n"
	}
	fmt.Printf(format, params[start:]...)
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

//Stats prints the given log message (format + params) using the StatsMask level and
//takes an extra parameter that will be visible in the log message as the category
//of stats that is reported.
func Statsf(category string, format string, params ...interface{}) {
	logf(StatsMask, format, append([]interface{}{category}, params...)...)
}
