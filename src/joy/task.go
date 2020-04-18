package joy

import (
	"feelings/src/hardware/rpi"
	"unsafe"
)

//
// CPUContext
//

type CPUContext interface {
}

type CPUContextImpl struct {
	x19 uint64
	x20 uint64
	x21 uint64
	x22 uint64
	x23 uint64
	x24 uint64
	x25 uint64
	x26 uint64
	x27 uint64
	x28 uint64
	fp  uint64
	sp  uint64
	pc  uint64
}

//
// Task (process or thread)
//
type Task interface {
}

type TaskImpl struct {
	CPUContextImpl
	state        uint64
	counter      uint64
	priority     uint64
	preemptCount uint64
}

func makeInitTask() Task {
	t := &TaskImpl{
		CPUContextImpl: CPUContextImpl{
			0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		},
		state:        0,
		counter:      0,
		priority:     1,
		preemptCount: 0,
	}
	return &t
}

//
// TaskList
//

type TaskList interface {
	CopyProcess() bool
}

type TaskListImpl struct {
	task []Task
}

const TaskListSize = 64

func makeTaskList() TaskList {
	impls := make([]*TaskImpl, TaskListSize)
	tl := &TaskListImpl{
		task: make([]Task, TaskListSize),
	}
	for i, impl := range impls {
		tl.task[i] = impl
	}
	return tl
}

func (t *TaskListImpl) CopyProcess() bool {
	return false
}

//
// Memory
//
const PageShift = 12
const TableShift = 9
const SectionShift = PageShift + TableShift
const PageSize = 1 << PageShift
const SectionSize = 1 << SectionShift
const LowMemory = 2 * SectionSize
const HighMemory = uint64(rpi.MemoryMappedIO)
const PagingMemory = HighMemory - LowMemory
const PagingPages = PagingMemory / PageSize

type Pager interface {
	GetFreePage() unsafe.Pointer
}

var memMap [PagingPages]uint16 //xxx shouldn't this be a bitmask? or at least uint8?

func getFreePage() unsafe.Pointer {
	for i := uint64(0); i < uint64(PagingPages); i++ {
		if memMap[i] == 0 {
			memMap[i] = 1
			return (unsafe.Pointer)(uintptr((LowMemory + i*PageSize)))
		}
	}
	return nil
}

func freePage(ptr unsafe.Pointer) {
	memMap[(uintptr(ptr)-LowMemory)/PageSize] = 0
}
