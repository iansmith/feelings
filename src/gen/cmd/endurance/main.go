package main

import (
	"fmt"
	"math/rand"
	"unsafe"

	"device/arm"

	"lib/upbeat"
)

//go:export raw_exception_handler
func rawExceptionHandler(t uint64, esr uint64, addr uint64, el uint64, procId uint64) {
	fmt.Printf("raw exception handler: %d@%x  esr=%x,el=%d,processor=%d", t, addr,
		esr, el, procId)
	for {
		arm.Asm("nop")
	}
}

type Stringish struct {
	S string
	L int
}

const maxGuesses = 3
const enduranceListSize = 400
const numOps = 1000

var poolDataNode [512]StringishNodeDL
var poolData [512]Stringish

var rawBitsForPool [32]uint64
var rawBitsForPoolNode [32]uint64

var pool = StringishManagedPool{
	elements: unsafe.Pointer(&poolData[0]),
	bitset:   upbeat.NewBitSet(512, uintptr(unsafe.Pointer(&rawBitsForPool[0]))),
	num:      512,
	size:     int(unsafe.Sizeof(Stringish{})),
}

var nodePool = StringishNodeDLManagedPool{
	elements: unsafe.Pointer(&poolDataNode[0]),
	bitset:   upbeat.NewBitSet(512, uintptr(unsafe.Pointer(&rawBitsForPoolNode[0]))),
	num:      512,
	size:     int(unsafe.Sizeof(StringishNodeDL{})),
}

var dl = NewStringishFixedDL(&nodePool, &pool)

func main() {

	rand.Seed(2)

	for i := 0; i < enduranceListSize; i++ {
		name := fmt.Sprintf("endurance%03d", i) // allocates
		node := dl.Append()
		node.L = len(name)
		node.S = name
	}

	opNum := 0
	for opNum < numOps && !dl.Empty() {
		n := rand.Intn(10)
		switch {
		case n < 2:
			fmt.Printf("insert@%d\n", opNum)
			elem := dl.Append()
			name := fmt.Sprintf("endurance:op%03d", opNum)
			elem.L = len(name)
			elem.S = name
		case n < 3:
			fmt.Printf("dequeue@%d\n", opNum)
			dl.DequeueAndRelease()
		case n < 4:
			fmt.Printf("pop@%d\n", opNum)
			dl.PopAndRelease()
		case n < 5:
			fmt.Printf("pop 2nd@%d\n", opNum)
			target := dl.First().Next()
			if target == nil {
				fmt.Printf("...skipping pop 2, first element\n")
				break
			}
			dl.RemoveAndRelease(target)
		case n < 6:
			fmt.Printf("dequeue 2nd@%d\n", opNum)
			target := dl.Last().Prev()
			if target == nil {
				fmt.Printf("...skipping dequeue 2, last element\n")
				break
			}
			dl.RemoveAndRelease(target)
		case n < 7:
			if !dl.Empty() {
				t := rand.Intn(dl.Length())
				fmt.Printf("remove (%d) 2nd@%d\n", t, opNum)
				target := dl.Nth(t)
				dl.RemoveAndRelease(target)
			}
		case n < 8:
			if !dl.Empty() {
				t := rand.Intn(dl.Length())
				target := dl.Nth(t)
				fmt.Printf("insert after (%d) %d\n", t, opNum)
				node, elem := dl.Alloc()
				name := fmt.Sprintf("endurance:op%03d", opNum)
				elem.L = len(name)
				elem.S = name
				dl.InsertAfter(target, node)
			}
		default:
			s := rand.Intn(dl.Length())
			t := rand.Intn(dl.Length())
			fmt.Printf("remove (%d) then insert after (%d) @%d\n", s, t, opNum)

			if s != t {
				targetT := dl.Nth(t)
				targetS := dl.Nth(s)
				dl.Remove(targetS)
				dl.InsertBefore(targetT, targetS)
			}
		}
		opNum++
	}
	if dl.Empty() {
		fmt.Printf("OK: list is empty so stopped\n")
	} else {
		fmt.Printf("after %d ops, DL size is %d\n", numOps, dl.Length())
		fmt.Printf("pools are full? %v\n", dl.PoolsFull())
		fmt.Printf("clearing remaining items from pool\n")
	}

	for !dl.Empty() {
		dl.PopAndRelease()
	}

	if !dl.PoolsFull() {
		fmt.Printf("error in pool, expected all nodes to be returned\n")
	} else {
		fmt.Printf("OK: all pools are full after clearing list\n")
	}

}
