package trust

import (
	"fmt"

	tgr "runtime"
)

// Default Logger points to the UART that is the primary output for the kernel log messages.
var DefaultLogger = &Logger{sink: newDefaultSink()}

func (d *defaultSink) Printf(format string, params ...interface{}) {
	fmt.Printf("now in printf of  sink \n")
	_, _ = fmt.Printf(format, params...)
}

func newDefaultSink() *defaultSink {
	return &defaultSink{}
}

type defaultSink struct{}
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

type LogSink interface {
	Printf(format string, params ...interface{})
}

type Logger struct {
	sink  LogSink
	level MaskLevel
}

func NewLogger(sink LogSink) *Logger {
	return &Logger{
		sink:  sink,
		level: fatalMask | StatsMask | ErrorMask | WarnMask | InfoMask | DebugMask,
	}
}

// SetLevel lets you set an error mask directly. You can pass in something like
// ErrorMask | DebugMask to control exactly what gets printed.  It returns the
// previous mask.
func (l *Logger) SetLevel(mask MaskLevel) MaskLevel {
	if mask&0xf == 0 {
		l.sink.Printf(" WARN: trust.SetLevel is turning of log messages\n")
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
	r := l.level & 0x1f
	l.level = result | fatalMask
	return r
}
func (l *Logger) Level() MaskLevel {
	return l.level
}

func (l *Logger) LevelToString() string {
	result := ""
	switch {
	case l.level&ErrorMask > 0:
		result += "error "
		fallthrough
	case l.level&WarnMask > 0:
		result += "warn "
		fallthrough
	case l.level&InfoMask > 0:
		result += "info "
		fallthrough
	case l.level&DebugMask > 0:
		result += "debug "
		fallthrough
	case l.level&StatsMask > 0:
		result += "stats"
	}
	return result
}

func (l *Logger) logf(m MaskLevel, format string, params ...interface{}) {
	if l.level&m == 0 {
		return
	}
	start := 0
	switch {
	case m&ErrorMask > 0:
		l.sink.Printf("ERROR:")
	case m&WarnMask > 0:
		l.sink.Printf(" WARN:")
	case m&InfoMask > 0:
		l.sink.Printf(" INFO:")
	case m&DebugMask > 0:
		l.sink.Printf("DEBUG:")
	case m&StatsMask > 0:
		s, ok := params[0].(string)
		if !ok {
			s = "unknown"
		}
		l.sink.Printf("STATS[%s]:", s)
		start = 1
	}

	if len(format) == 0 {
		format = "\n"
	} else if format[len(format)-1] != '\n' {
		format += "\n"
	}

	if len(params) == start {
		l.sink.Printf(format)
	} else {
		l.sink.Printf(format, params...)
	}
}

//Fatalf prints the given log message (format + params) on stdout and then
//exits with the exitCode provided.  Fatalf is not maskable.
func Fatalf(exitCode int, format string, params ...interface{}) {
	DefaultLogger.Fatalf(exitCode, format, params)
}
func (l *Logger) Fatalf(exitCode int, format string, params ...interface{}) {
	l.logf(fatalMask, format, params...)
	tgr.Exit()
}

//Errorf prints the given log message (format + params) using the ErrorMask level.
func Errorf(format string, params ...interface{}) {
	DefaultLogger.Errorf(format, params...)
}

func (l *Logger) Errorf(format string, params ...interface{}) {
	l.logf(ErrorMask, format, params...)
}

//Warnf prints the given log message (format + params) using the WarnMask level.
func Warnf(format string, params ...interface{}) {
	DefaultLogger.Warnf(format, params...)
}

func (l *Logger) Warnf(format string, params ...interface{}) {
	l.logf(WarnMask, format, params...)
}

//Infof prints the given log message (format + params) using the InfoMask level.
func Infof(format string, params ...interface{}) {
	DefaultLogger.Infof(format, params...)
}
func (l *Logger) Infof(format string, params ...interface{}) {
	l.logf(InfoMask, format, params...)
}

//Debugf prints the given log message (format + params) using the DebugMask level.
func Debugf(format string, params ...interface{}) {
	DefaultLogger.Debugf(format, params...)
}

func (l *Logger) Debugf(format string, params ...interface{}) {
	l.logf(DebugMask, format, params...)
}

//Stats prints the given log message (format + params) using the StatsMask level and
//takes an extra parameter that will be visible in the log message as the category
//of stats that is reported.
func Statsf(category string, format string, params ...interface{}) {
	DefaultLogger.Statsf(category, format, params...)
}

func (l *Logger) Statsf(category string, format string, params ...interface{}) {
	l.logf(StatsMask, format, append([]interface{}{category}, params...)...)
}
