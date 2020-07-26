package emmc

import (
	"math/rand"
	"unsafe"

	"lib/trust"
	"lib/upbeat"
)

const tranquilDebug = false
const failLimit = 3
const pageUnit = 512

//go:extern loaded_bit_set
var loadedBitSet uint64

//go:extern sector_cache
var sectorCache uint64

type bufferManager interface {
	PossiblyLoad(sector sectorNumber) (unsafe.Pointer, EmmcError) //loads if page is not in an existing buffer
	DumpStats(clear bool)                                         //pass true if you wants stats cleared as well
}

type Tranquil struct {
	data         unsafe.Pointer //actually a contiguous buffer of 512 byte pages
	sizeInPages  uint32
	inUse        *upbeat.BitSet
	loader       loader
	saver        saver
	pageMap      map[sectorNumber]bufferEntry
	cacheHits    uint64
	cacheMisses  uint64
	cacheOusters uint64
}

type loader func(sectorNumber, unsafe.Pointer /*watch alignment!*/) EmmcError
type saver func(sectorNumber, unsafe.Pointer /*watch alignment!*/) EmmcError

// pass in a contiguous buffer, must be a multiple of sectorSize bytes and the number of pages
// 1page=sectorSize bytes is the 2nd param
func NewTraquilBufferManager(ptr unsafe.Pointer, sizeInSectors uint32, bitSetData unsafe.Pointer,
	ld loader, sv saver) *Tranquil {
	result := &Tranquil{
		sizeInPages: sizeInSectors,
		data:        ptr,
		inUse:       upbeat.NewBitSet(sizeInSectors, bitSetData),
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
func (t *Tranquil) PossiblyLoad(sector sectorNumber) (unsafe.Pointer, EmmcError) {
	entry, ok := t.pageMap[sector]
	if ok {
		if tranquilDebug {
			trust.Debugf("tranquil.PossblyLoad: cache hit, sector %d -> index %d",
				sector, entry.cachePage)
		}
		t.cacheHits++
		return entry.ptr, EmmcOk
	}
	t.cacheMisses++
	if tranquilDebug {
		trust.Debugf("tranquil.PossblyLoad: cache miss for %d", sector)
	}

	//do a few random samples seeing if we get lucky
	fails := 0
	haveWinner := false
	winner := cacheIndex(0)
	for fails < failLimit {
		fails++
		r := cacheIndex(rand.Intn(int(t.sizeInPages)))
		if t.inUse.On(upbeat.BitIndex(r)) {
			continue
		}
		winner = r
		haveWinner = true
		break
	}
	if !haveWinner {
		//any free spaces?
		for i := cacheIndex(0); i < cacheIndex(t.sizeInPages); i++ {
			if t.inUse.On(upbeat.BitIndex(i)) {
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
	//xxx needs to be rethoughqt when we get to things with DMA that are not synchronous
	//xxx we probably will need to lock the candidate page with some type of "expected" mark

	// compute where to load the data
	if tranquilDebug {
		trust.Debugf("tranquil.PossblyLoad: sector %d -> index %d", sector, winner)
	}

	ptr := unsafe.Pointer(uintptr(t.data) + uintptr(winner*pageUnit))
	//store the mapping
	t.pageMap[sector] = bufferEntry{ptr, winner}
	t.inUse.Set(upbeat.BitIndex(winner))
	if err := t.loader(sector, ptr); err != EmmcOk {
		trust.Errorf("buffer management failed to load page: %x", sector)
		return nil, EmmcFailedReadIntoCache
	}
	return ptr, EmmcOk
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
