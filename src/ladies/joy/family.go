package joy

import (
	"unsafe"

	"lib/trust"
)

// Family public API
// InitFamilies()     called once at startup
// PermitPremption    called to make the current family preemtable
// ProhibitPremption  called to make the current family not preemtable
// Copy               called to copy another family (fork); creates runnable family
// Reclaim            called to reclaim the resources of a family that has exited
//
// This is just for cleanliness inside joy.
type FamilyAPIDef interface {
	Init()
	PermitPreemption()
	ProhibitPreemption()
	Copy(id FamilyId, fn FuncPtr, arg uint64) (FamilyId, JoyError)
	Reclaim(id uint16) JoyError
}

type FamilyId uint16

const NoFamilyId FamilyId = 0xffff

type familyAPIImpl struct {
}

var FamilyAPI FamilyAPIDef = &familyAPIImpl{}

func (f *familyAPIImpl) Init() {
	initFamilies()
}
func (f *familyAPIImpl) PermitPreemption() {
	permitPreemption()
}
func (f *familyAPIImpl) ProhibitPreemption() {
	prohibitPreemption()
}
func (f *familyAPIImpl) Copy(id FamilyId, fn FuncPtr, arg uint64) (FamilyId, JoyError) {
	return familyCopy(id, fn, arg)
}

func (f *familyAPIImpl) Reclaim(id uint16) JoyError {
	return familyReclaim(id)
}

//type of a function pointer for our purposes... has to point to a simple machine
//address, not something like a closure!  There is no checking.  This type is
//usually connected to the elements in the funcnames.list file
type FuncPtr uint64

// Maximum number of families in the system.
const maxFamilies = 64

// familyState is info about a given family contained in the FCB.
type familyState int

const (
	fsRunning familyState = 0
	fsZombie  familyState = 1
)

// familyFlags are just markers on the familyImpl for internal use.
type familyFlags uint64

const (
	ffKernelThread familyFlags = 0 << 1
)

//
// Each family is recorded here.  The FCB points to a region of memory at the
// bottom (only reached on stack overflow) of their stack.
//
var familyImpl [maxFamilies]*family

//
// family is where we store all of the data structures that are
// per family.
//
type family struct {
	rss          RegisterSavedState
	state        familyState
	counter      int64
	priority     int64
	preemptCount int64
	Stack        uint64
	HeapStart    unsafe.Pointer
	HeapEnd      unsafe.Pointer
	flags        uint64 //bitfield
	Id           uint64 //really this is a uint16 but we do this for alignment
}

//
// RegisterSavedState is the saved registers from the last time the familyImpl
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

// familyZero is the information about the kernel process that starts everything.
// Or maybe it's where the epidemic started.
var familyZero = family{
	rss: RegisterSavedState{0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0},
	state:        fsRunning,
	counter:      0,
	priority:     2,
	preemptCount: 0,
	flags:        uint64(ffKernelThread),
}

// familiesRunning is the number of schedulable units that could use
// processor time.
var familiesRunning uint16

// currentFamily is the domain that is currently on the CPU.  This is
// an index into the array familyImpl.
var currentFamily *family

// initFamilies is called once at startup time.   This sets up some
// data structures for Family 0 that are not memory related.
func initFamilies() {
	familiesRunning = 1
	familyImpl[0] = currentFamily
}

func prohibitPreemption() {
	currentFamily.preemptCount++
}

//go:export permit_preemption
func permitPreemption() {
	currentFamily.preemptCount--
}

func familyCopy(id FamilyId, fn FuncPtr, arg uint64) (FamilyId, JoyError) {
	prohibitPreemption()

	newStack1st, newStackLast, err := KMemAPI.GetContiguousPages(kProcStackPages)
	if err != JoyNoError {
		return NoFamilyId, err
	}
	top := uintptr(KMemAPI.ToPtr(newStackLast)) + uintptr(kpageSize-16)
	newFamily := (*family)(KMemAPI.ToPtr(newStack1st))

	newHeap1st, newHeapLast, err := KMemAPI.GetContiguousPages(kProcHeapPages)
	if err != JoyNoError {
		return NoFamilyId, err
	}
	newHeapEnd := unsafe.Pointer(uintptr(KMemAPI.ToPtr(newHeapLast)) + uintptr(kpageSize))

	newFamily.priority = currentFamily.priority
	newFamily.state = fsRunning
	newFamily.counter = newFamily.priority
	newFamily.preemptCount = 1
	newFamily.HeapStart = KMemAPI.ToPtr(newHeap1st)
	newFamily.HeapEnd = newHeapEnd
	newFamily.rss.X19 = uint64(fn)
	newFamily.rss.X20 = arg
	newFamily.rss.PC = retFromForkPtr
	newFamily.rss.SP = uint64(top)
	index, err := findFamilySlot()
	if err != JoyNoError {
		return NoFamilyId, err
	}
	familyImpl[index] = newFamily
	newFamily.Id = uint64(familiesRunning)
	familiesRunning++
	trust.Debugf("family copied successfully (FCB is %p, X19=%x, PC=%x) with prio %d",
		newFamily, newFamily.rss.X19, newFamily.rss.PC, newFamily.priority)
	permitPreemption()
	return index, JoyNoError
}

//go:extern
var retFromForkPtr uint64

func retFromFork()

// find next empty slot
func findFamilySlot() (FamilyId, JoyError) {
	for i := 0; i < maxFamilies; i++ {
		if familyImpl[i] == nil {
			return FamilyId(i), JoyNoError
		}
	}
	return 0, MakeError(ErrorFamilyNoMoreFamilies)
}

func familyReclaim(id uint16) JoyError {
	return JoyNoError
}
