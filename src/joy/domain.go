package joy

import (
	"unsafe"

	"lib/trust"
)

//type of a function pointer for our purposes... has to point to a simple machine
//address, not something like a closure!
type FuncPtr uint64

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
	Priority:     2,
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

// InitDomains is called once at startup time.   This sets up some
// data structures for process 0 that are not memory related.
func InitDomains() {
	DomainsRunning = 1
	Domain[0] = CurrentDomain
}

func DisallowPreemption() {
	CurrentDomain.PreemptCount++
}

//go:export PermitPreemption
func PermitPreemption() {
	CurrentDomain.PreemptCount--
}

func DomainCopy(fn FuncPtr, arg uint64) JoyError {
	DisallowPreemption()

	newStack1st, newStackLast, err := KMemoryGetFreeContiguousPages(KernelProcStackPages)
	if err != JoyNoError {
		return err
	}
	top := uintptr(newStackLast) + uintptr(KPageSize-16)
	trust.Debugf("domain being allocated (new one) -- stack root is at %x ",
		"(prio? %d) and current=%x", uintptr(newStack1st),
		CurrentDomain.Priority, uintptr(unsafe.Pointer(CurrentDomain)))
	newDomain := (*DomainControlBlock)(unsafe.Pointer(uintptr(newStack1st)))

	newHeap1st, newHeapLast, err := KMemoryGetFreeContiguousPages(KernelProcHeapPages)
	if err != JoyNoError {
		return err
	}
	newHeapEnd := unsafe.Pointer(uintptr(newHeapLast) + uintptr(KPageSize))

	newDomain.Priority = CurrentDomain.Priority
	newDomain.State = DomainStateRunning
	newDomain.Counter = newDomain.Priority
	newDomain.PreemptCount = 1
	newDomain.HeapStart = newHeap1st
	newDomain.HeapEnd = newHeapEnd
	newDomain.RSS.X19 = uint64(fn)
	newDomain.RSS.X20 = arg
	newDomain.RSS.PC = retFromForkPtr
	newDomain.RSS.SP = uint64(top)
	index, err := findNewDomainSlot()
	if err != JoyNoError {
		return err
	}
	Domain[index] = newDomain
	newDomain.Id = uint64(DomainsRunning)
	DomainsRunning++
	trust.Debugf("domain copied successfully (DCB is %p, X19=%x, PC=%x) with prio %d",
		newDomain, newDomain.RSS.X19, newDomain.RSS.PC, newDomain.Priority)
	PermitPreemption()
	return JoyNoError
}

//go:extern
var retFromForkPtr uint64

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

//func LaunderFunctionPtr0(func()) uint64
//func LaunderFunctionPtr1(func(uintptr)) uint64
