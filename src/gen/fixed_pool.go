package gen

import (
	"math/rand"
	"unsafe"

	"lib/upbeat"
)

type GenericManagedPool struct {
	rawBits  [32]uint64
	elements *Generic
	bitset   *upbeat.BitSet
	num      int
	size     int
}

// NewGenericManagedPool returns a managed pool that allocates the
// elements pointed to by elements. Be careful that you are sure that each
// element is at least 4 byte aligned, 16 byte aligned if you want to use
// these object on the stack. Maximum allowed size is 2048 and size MUST
// be a multiple of 64.  Watch out for padding at the end of structs!
func NewGenericManagedPool(numElements uint32, bytesPerElement int,
	elements *Generic) GenericManagedPool {
	result := GenericManagedPool{elements: elements}
	if numElements >= 2048 || numElements%64 != 0 {
		panic("requested size is not valid for a pool")
	}
	result.num = int(numElements) // valid because it's positive and < 2048
	result.size = bytesPerElement
	result.bitset = upbeat.NewBitSet(numElements, unsafe.Pointer(&result.rawBits[0]))
	return result
}

// Alloc returns a pointer to an element in the pool.  It returns nil
// if the pool is exhausted.  Note that a pool may go from exhausted
// to working if dealloc() is called.
func (g *GenericManagedPool) Alloc() *Generic {
	tries := 0
	for tries < maxGuesses {
		guess := rand.Intn(g.num)
		if g.bitset.On(upbeat.BitIndex(guess)) {
			tries++
			continue
		}
		// use that one
		g.bitset.Set(upbeat.BitIndex(guess))
		return g.computePtrToElement(guess)
	}
	// ugly search
	for i := 0; i < g.num; i++ {
		if g.bitset.On(upbeat.BitIndex(i)) {
			continue
		}
		// use that one
		g.bitset.Set(upbeat.BitIndex(i))
		return g.computePtrToElement(i)
	}
	return nil
}

func (g *GenericManagedPool) Dealloc(ptr *Generic) {

	guess := g.num / 2
	guessPtr := g.computePtrToElement(guess)
	bottom := 0
	top := g.num
	// [bottom,top)
	for uintptr(unsafe.Pointer(guessPtr)) !=
		uintptr(unsafe.Pointer(ptr)) && top-1 != bottom {
		if uintptr(unsafe.Pointer(guessPtr)) < uintptr(unsafe.Pointer(ptr)) {
			bottom = guess + 1 // includes
		} else {
			top = guess // excludes
		}
		guess = ((top - bottom) / 2) + bottom
		guessPtr = g.computePtrToElement(guess)
	}
	if uintptr(unsafe.Pointer(guessPtr)) != uintptr(unsafe.Pointer(ptr)) {
		panic("pointer passed to dealloc() that is not from pool")
	}
	g.bitset.Clear(upbeat.BitIndex(guess))
}

func (g *GenericManagedPool) Full() bool {
	for i := 0; i < g.num; i++ {
		if g.bitset.On(upbeat.BitIndex(i)) {
			return false
		}
	}
	return true
}
func (g *GenericManagedPool) Empty() bool {
	for i := 0; i < g.num; i++ {
		if !g.bitset.On(upbeat.BitIndex(i)) {
			return false
		}
	}
	return true
}

func (g *GenericManagedPool) computePtrToElement(guess int) *Generic {
	offset := g.size * guess
	result := uintptr(unsafe.Pointer(g.elements)) + uintptr(offset)
	return (*Generic)(unsafe.Pointer(result))
}

//
// Convenience for working with a DL's Node type
//
type GenericNodeDLManagedPool GenericManagedPool

func NewGenericNodeDLManagedPool(numElements uint32, bytesPerElement int,
	elements *GenericNodeDL) GenericNodeDLManagedPool {
	converted := (*Generic)(unsafe.Pointer(elements))
	return GenericNodeDLManagedPool(NewGenericManagedPool(numElements, bytesPerElement, converted))
}
func (g *GenericNodeDLManagedPool) Alloc() *GenericNodeDL {
	a := (*GenericManagedPool)(g).Alloc()
	return (*GenericNodeDL)(unsafe.Pointer(a))
}
func (g *GenericNodeDLManagedPool) Dealloc(n *GenericNodeDL) {
	(*GenericManagedPool)(g).Dealloc((*Generic)(unsafe.Pointer(n)))
}
func (g *GenericNodeDLManagedPool) Full() bool {
	return (*GenericManagedPool)(g).Full()
}
func (g *GenericNodeDLManagedPool) Empty() bool {
	return (*GenericManagedPool)(g).Empty()
}
