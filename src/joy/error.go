package joy

import "fmt"

const subsystemMask = 0x00ff_0000_0000_0000
const domainIDMask = 0x0000_ffff_0000_0000
const errorNumberMask = 0x0000_0000_0000_ffff

const JoyNoError = JoyError(0)

// Memory Errors
const MemorySubsystem = 1
const MemoryPageAlreadyInUse = 1
const MemoryPageNotAvailable = 2
const MemoryBadPageRequest = 3
const MemoryAlreadyFree = 4

var ErrorMemoryPageAlreadyInUse = errorValue(MemorySubsystem, MemoryPageAlreadyInUse)
var ErrorMemoryPageNotAvailable = errorValue(MemorySubsystem, MemoryPageNotAvailable)
var ErrorMemoryBadPageRequest = errorValue(MemorySubsystem, MemoryBadPageRequest)
var ErrorMemoryAlreadyFree = errorValue(MemorySubsystem, MemoryAlreadyFree)

// Domain Errors
const DomainSubsystem = 2
const DomainNoMoreDomains = 1

var ErrorDomainNoMoreDomains = errorValue(DomainSubsystem, DomainNoMoreDomains)

type JoyError uint64
type RawJoyError uint64 // error with just the constant part of the value filled in

var errorMap map[uint64]string

func JoyErrorMessage(j JoyError) string {
	return errorText(uint64(j))
}

func InitErrors() {
	errorMap = make(map[uint64]string)
	createError(MemorySubsystem, MemoryPageAlreadyInUse,
		"memory page is already in use by other process")
}

func createError(subsys byte, errorNumber uint16, format string) {
	n := errorValue(subsys, errorNumber)
	errorMap[uint64(n)] = format
}

func errorText(raw uint64) string {
	t, ok := errorMap[raw]
	if !ok {
		return "Unknown error code"
	}
	did := raw & domainIDMask
	return fmt.Sprintf("Domain %d: %s", did, t) //xxx allocation
}

func errorValue(subsys byte, errorNumber uint16) RawJoyError {
	ss := subsystemMask & (uint64(subsys) << 48)
	en := errorNumberMask & (uint64(errorNumber) << 0)
	return RawJoyError(ss | en)
}

// MakeError adds the dynamic fields (like current domain) to the error value.
func MakeError(rawError RawJoyError) JoyError {
	raw := uint64(rawError)
	did := (CurrentDomain.Id << 32) & domainIDMask
	return JoyError(raw | did)
}
