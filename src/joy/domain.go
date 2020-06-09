package joy

import (
	"unsafe"

	"lib/trust"
)

// Maximum number of domains (roughly processes) in the system.
const MaxDomains = 64

// DomainState is info about a given doman contained in the DCB.
type DomainState int

const (
	DomainStateRunning DomainState = 0
	DomainStateZombie  DomainState = 1
)

// DomainFlags are just markers on the Domain for internal use.
type DomainFlags uint64

const (
	DomainFlagKernelThread DomainFlags = 0 << 1
)

//
// Each domain is recorded here.  Each ptr points to their page for stack.
//
var Domain [MaxDomains]*DomainControlBlock

//
// DomainControlBlock is where we store all of the data strutures that are
// per domain.
//
type DomainControlBlock struct {
	RSS          RegisterSavedState
	State        DomainState
	Counter      int64
	Priority     int64
	PreemptCount int64
	Stack        uint64
	HeapStart    unsafe.Pointer
	HeapEnd      unsafe.Pointer
	Flags        uint64 //bitfield
	Id           uint64 //really this is a uint16 but we do this for alignment
}

//
// RegisterSavedState is the saved registers from the last time the Domain
// was executing.
//
type RegisterSavedState struct {
	X19 uint64
	X20 uint64
	X21 uint64
	X22 uint64
	X23 uint64
	X24 uint64
	X25 uint64
	X26 uint64
	X27 uint64
	X28 uint64
	FP  uint64
	SP  uint64
	PC  uint64
}

// DomainOne is the information about the kernel process that starts everything.
var DomainZero = DomainControlBlock{
	RSS: RegisterSavedState{0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0},
	State:        DomainStateRunning,
	Counter:      0,
	Priority:     1,
	PreemptCount: 0,
	Stack:        0,
	Flags:        uint64(DomainFlagKernelThread),
}

// DomainsRunning is the number of schedulable units that could use
// processor time.
var DomainsRunning uint16

// CurrentDomain is the domain that is currently on the CPU.  This is
// an index into the array Domain.
var CurrentDomain *DomainControlBlock

// InitDomains is called once at startup time.   This initializes
// data structures for running, with DomainOne as the domain that
// is executing.
func InitDomains(stackPage unsafe.Pointer, heapStart unsafe.Pointer, heapEnd unsafe.Pointer) {
	//xxx how do we do stack alignment here? I just subtracted 16 so I can write
	//xxx to the stack as normal, but this seems pretty bogus
	top := unsafe.Pointer(uintptr(stackPage) + uintptr(KPageSize-16))
	bottom := (*DomainControlBlock)(unsafe.Pointer(uintptr(stackPage) + uintptr(KPageSize)))
	*bottom = DomainZero
	bottom.Stack = uint64(uintptr(top))
	bottom.HeapStart = heapStart
	bottom.HeapEnd = heapEnd
	DomainsRunning = 1
	CurrentDomain = (*DomainControlBlock)(top)
}

func DisallowPreemption() {
	CurrentDomain.PreemptCount++
}

//go:export PermitPreemption
func PermitPreemption() {
	CurrentDomain.PreemptCount--
}

func DomainCopy(fn func(uintptr), arg uint64) JoyError {
	DisallowPreemption()
	trust.Debugf("domain being copied")

	heapStart, err := KMemoryGetFreePage()
	if err != JoyNoError {
		return err
	}
	heapEnd := unsafe.Pointer(uintptr(heapStart) + uintptr(KPageSize))
	codeAndStack, err := KMemoryGetFreePage()
	if err != JoyNoError {
		return err
	}
	top := uintptr(codeAndStack) + uintptr(KPageSize-16)
	newDomain := (*DomainControlBlock)(unsafe.Pointer(uintptr(codeAndStack) + uintptr(KPageSize)))

	newDomain.Priority = CurrentDomain.Priority
	newDomain.State = DomainStateRunning
	newDomain.Counter = newDomain.Priority
	newDomain.PreemptCount = 1
	newDomain.HeapStart = heapStart
	newDomain.HeapEnd = heapEnd
	newDomain.RSS.X19 = LaunderFunctionPtr1(fn)
	newDomain.RSS.X20 = arg
	newDomain.RSS.PC = LaunderFunctionPtr0(retFromFork)
	newDomain.RSS.SP = uint64(top)
	index, err := findNewDomainSlot()
	if err != JoyNoError {
		return err
	}
	Domain[index] = newDomain
	newDomain.Id = uint64(DomainsRunning)
	DomainsRunning++
	trust.Debugf("domain copied successfully (%d)", newDomain.Id)
	PermitPreemption()
	return JoyNoError
}

func findNewDomainSlot() (int, JoyError) {
	for i := 0; i < MaxDomains; i++ {
		if Domain[i] == nil {
			return i, JoyNoError
		}
	}
	return 0, MakeError(ErrorDomainNoMoreDomains)
}

func setHeapPointers(start uint64, end uint64)
func retFromFork()

func LaunderFunctionPtr0(func()) uint64
func LaunderFunctionPtr1(func(uintptr)) uint64
