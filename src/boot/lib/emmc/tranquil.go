package main

import (
	"errors"
	"lib/trust"
	"math/rand"

	"unsafe"
)

type BitSet struct {
	size uint32
	data unsafe.Pointer //actually an array of uint64s
}

const failLimit = 3
const pageUnit = 512

//bitsets have to be multiples of 64.  the ptr provided should be
//already allocated statically to be the place to store the data.
//caller must be sure it is aligned properly.
func NewBitSet(size uint32, ptr unsafe.Pointer) *BitSet {
	mask := ^(uint32(0x3f))
	if size&mask != size {
		trust.Errorf("your bitset size is not a multiple of 64: %d", size)
		return nil
	}
	result := &BitSet{
		data: ptr,
		size: size,
	}
	result.ClearAll()
	return result
}

func (b *BitSet) On(bit cacheIndex) bool {
	boff := uintptr(bit >> 3) //divide by 8 to get number of bytes to move pointer
	mask := uint64(1 << (bit % 64))
	tmp := (*uint64)(unsafe.Pointer(uintptr(b.data) + boff))
	return *tmp&mask != 0
}

func (b *BitSet) ClearAll() {
	numUint64s := b.size >> 6 //really dividing by 64 because  512/8
	for i := uint32(0); i < numUint64s; i++ {
		curr := (*uint64)(unsafe.Pointer(uintptr(i*8) + uintptr(unsafe.Pointer(b.data))))
		*curr = 0
	}
}

//go:extern loaded_bit_set
var loadedBitSet uint64

//go:extern sector_cache
var sectorCache uint64

type bufferManager interface {
	PossiblyLoad(sector sectorNumber) (unsafe.Pointer, error) //loads if page is not in an existing buffer
	DumpStats(clear bool)                                     //pass true if you wants stats cleared as well
}

type Tranquil struct {
	data         unsafe.Pointer //actually a contiguous buffer of 512 byte pages
	sizeInPages  uint32
	inUse        *BitSet
	loader       loader
	saver        saver
	pageMap      map[sectorNumber]bufferEntry
	cacheHits    uint64
	cacheMisses  uint64
	cacheOusters uint64
}

type loader func(sectorNumber, unsafe.Pointer /*watch alignment!*/) int
type saver func(sectorNumber, unsafe.Pointer /*watch alignment!*/) int

// pass in a contiguous buffer, must be a multiple of 512 bytes and the number of pages
// 1page=512bytes is the 2nd param
func NewTraquilBufferManager(ptr unsafe.Pointer, sizeInPages uint32, bitSetData unsafe.Pointer,
	ld loader, sv saver) *Tranquil {
	result := &Tranquil{
		sizeInPages: sizeInPages,
		data:        ptr,
		inUse:       NewBitSet(sizeInPages, bitSetData),
		pageMap:     make(map[sectorNumber]bufferEntry),
		loader:      ld,
		saver:       sv,
	}
	if result.loader == nil {
		result.loader = readInto
	}
	return result
}

type cacheIndex uint32

type bufferEntry struct {
	ptr       unsafe.Pointer
	cachePage cacheIndex
}

// PossiblyLoad returns a pointer to the data page requested, possibly loading the
// the page as it does so.  It
func (t *Tranquil) PossiblyLoad(sector sectorNumber) (unsafe.Pointer, error) {
	entry, ok := t.pageMap[sector]
	if ok {
		t.cacheHits++
		return entry.ptr, nil
	}
	t.cacheMisses++

	//do a few random samples seeing if we get lucky
	fails := 0
	haveWinner := false
	winner := cacheIndex(0)
	for fails < failLimit {
		fails++
		r := cacheIndex(rand.Intn(int(t.sizeInPages)))
		if t.inUse.On(r) {
			continue
		}
		winner = r
		haveWinner = true
		break
	}
	if !haveWinner {
		//any free spaces?
		for i := cacheIndex(0); i < cacheIndex(t.sizeInPages); i++ {
			if t.inUse.On(i) {
				continue
			}
			haveWinner = true
			winner = i
			break
		}
		if !haveWinner {
			t.cacheOusters++
			//randomly pick a loser, all spots full
			r := cacheIndex(rand.Intn(int(t.sizeInPages)))
			for sector, entry := range t.pageMap {
				if entry.cachePage == r {
					haveWinner = true
					winner = entry.cachePage
					delete(t.pageMap, sector)
					break
				}
			}
		}
	}
	if !haveWinner {
		panic("unable to find any cache slot to put new sector into!")
	}
	//xxx we are going to do the whole thing synchronously, but this strategy will
	//xxx needs to be rethought when we get to things with DMA that are not synchronous
	//xxx we probably will need to lock the candidate page with some type of "expected" mark

	// compute where to load the data
	ptr := unsafe.Pointer(uintptr(t.data) + uintptr(winner*pageUnit))
	//store the mapping
	t.pageMap[sector] = bufferEntry{ptr, winner}
	if err := t.loader(sector, ptr); err != sdOk {
		trust.Errorf("buffer management failed to load page: %x", sector)
		return nil, errors.New("read failed")
	}
	return ptr, nil
}

func (t *Tranquil) DumpStats(clear bool) {
	trust.Statsf("pageCache", "cache hits: %d, cache misses %d, cache hit %2.0f%%, ousters %d\n",
		t.cacheHits, t.cacheMisses,
		(float64(t.cacheHits)/(float64(t.cacheHits)+float64(t.cacheMisses)))*100.0,
		t.cacheOusters)
	if clear {
		t.cacheHits = 0
		t.cacheMisses = 0
		t.cacheOusters = 0
	}
}
