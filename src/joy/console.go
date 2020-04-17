package main

import (
	"feelings/src/golang/strconv"
	rt "feelings/src/tinygo_runtime"
	"reflect"
)

var Console ConsoleInterface = &ConsoleImpl{}

type ConsoleInterface interface {
	Logf(string, ...interface{})
	Sprintf(string, ...interface{}) string
}

type ConsoleImpl struct {
}

func (c *ConsoleImpl) Logf(format string, values ...interface{}) {
	if format == "" {
		return
	}
	rt.MiniUART.WriteString(c.Sprintf(format, values...))
	if format[len(format)-1] != '\n' {
		rt.MiniUART.WriteCR()
	}
}

func (c *ConsoleImpl) Sprintf(format string, values ...interface{}) string {
	current := 0 //which param
	i := 0
	result := ""
	for {
		if i >= len(format) { //reached end of string
			break
		}
		s := format[i]
		if s != '%' {
			if format[i] == '\n' {
				result = result + string('\r') + string('\n')
			} else {
				result = result + string(format[i])
			}
			i++
			continue
		}
		if i == len(format)-1 {
			result += "!format string ended with %"
			return result
		}
		// %% case
		if format[i+1] == '%' {
			result += "%"
			i++
			continue
		}
		if len(values) <= current {
			result += "!missing value"
			return result
		} else {
			value := values[current]
			ch, size, ok := snarfSpecifier(format, i+1)
			i += 1 + len(size) //single char+spec
			current++
			if !ok {
				result += "!unterminated % specifier"
				return result
			}
			result += printType(ch, value, size)
			i++
		}
	}
	return result
}

func printType(ch uint8, value interface{}, sz string) string {
	switch ch {
	case 'd', 'x':
		base := 16
		if ch == 'd' {
			base = 10
		}
		switch reflect.ValueOf(value).Kind() {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return formatString(ch, strconv.FormatInt(reflect.ValueOf(value).Int(), base), sz)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return formatString(ch, strconv.FormatUint(reflect.ValueOf(value).Uint(), base), sz)
		case reflect.Uintptr:
			return formatString(ch, "->"+strconv.FormatUint(reflect.ValueOf(value).Uint(), base), sz)
		default:
			return "!mismatched type %" + string(ch) + "!"
		}
	case 's':
		switch reflect.ValueOf(value).Kind() {
		case reflect.String:
			return formatString(ch, reflect.ValueOf(value).String(), sz)
		default:
			return "!mismatched type %" + string(ch) + "!"
		}
	case 'v':
		switch reflect.ValueOf(value).Kind() {
		case reflect.String:
			return formatString('v', reflect.ValueOf(value).String(), sz)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return formatString('v', strconv.FormatInt(reflect.ValueOf(value).Int(), 10), sz)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return formatString('v', strconv.FormatUint(reflect.ValueOf(value).Uint(), 16), sz)
		default:
			return "!mismatched type %" + string(ch) + "!"
		}
	}
	return "!unknown control char " + string(ch) + "!"
}

func stringWithSize(s string, size int) string {
	pos := size
	if pos < 0 {
		pos = -pos
	}
	if len(s) >= pos {
		return s
	}
	var result string
	if size < 0 {
		result = s
		for i := len(s); i < pos; i++ {
			result = result + " "
		}
	} else {
		result = ""
		for i := len(s); i < pos; i++ {
			result = result + " "
		}
		result = result + s
	}
	return result
}

func formatString(ch uint8, s string, raw string) string {
	//now the gnarly case
	leadingZeros := false
	var sz int64
	if raw != "" {
		if raw[0] == '0' {
			leadingZeros = true
		}
		if raw[0] == '-' && len(raw) > 1 && raw[1] == '0' {
			leadingZeros = true
		}
		var err error
		sz, err = strconv.ParseInt(raw, 10, 32)
		if err != nil {
			return "!bad size spec " + raw + "!"
		}
	} else {
		sz = 0
	}

	switch ch {
	case 's', 'v':
		return stringWithSize(s, int(sz))
	case 'd', 'x':
		pos := sz
		if pos < 0 {
			pos = -pos
		}
		if leadingZeros {
			negativeLZ := false
			if s[0] == '-' {
				negativeLZ = true
			}
			if !negativeLZ {
				for i := len(s); i < int(pos); i++ {
					s = "0" + s
				}
			} else {
				s = s[1:]
				for i := len(s); i < int(pos)-1; i++ {
					s = "0" + s
				}
				s = "-" + s
			}
		}
		return stringWithSize(s, int(sz))
	}
	return "!unknown control char " + string(ch) + "!"
}

func snarfSpecifier(s string, start int) (uint8, string, bool) {
	completed := false
	qualifier := ""
	var keychar uint8
	for i := start; !completed && i < len(s); i++ {
		switch s[i] {
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '-':
			qualifier += string(s[i])
		case 'd', 's', 'x', 'v':
			completed = true
			keychar = s[i]
		default:
			completed = true
			keychar = '?'
		}
	}
	return keychar, qualifier, completed
}
