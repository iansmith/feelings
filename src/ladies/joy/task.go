package joy

const MemoryMappedIO = 0x3F00_0000

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
