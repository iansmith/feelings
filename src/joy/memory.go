package joy

import (
	"unsafe"

	"lib/trust"
)

const KPageSize = uint64(0x1000) //64KB
const KRamStart = uint64(0xfffffc0030000000)

//const KRamEnd = 0xffff_fc00_32FF_FFFF
const KNumPages = 768
const KInUseSize = 96 //bit vector

//extern _heap_start
var heap_start [0]uintptr //pointer from assembly

//extern _heap_end
var heap_end [0]uintptr //pointer from assembly

var KMemInUse [KInUseSize]uint64

// KMemoryInint sets up for Domain 0 and returns the stack to use for that domain
// if all goes well.  The other two pointers are the start and end of the heap
// for domain 0.
func KMemoryInit() (unsafe.Pointer, unsafe.Pointer, unsafe.Pointer, JoyError) {
	//our code page
	if _, err := KMemorySetInUse(0); err != JoyNoError {
		return nil, nil, nil, err
	}
	//stack was set up by the bootloader
	stack, err := KMemorySetInUse(1)
	if err != JoyNoError {
		return nil, nil, nil, err
	}
	//we need setup heap
	ptr, err := KMemorySetInUse(2)
	if err != JoyNoError {
		return nil, nil, nil, err
	}
	//setup our heap, we do this now so we can proceed with normal execution
	setHeapPointers(uint64(uintptr(ptr)), uint64(uintptr(ptr)+uintptr(KPageSize)))
	return stack, ptr, (unsafe.Pointer)(uintptr(ptr) + uintptr(KPageSize)), JoyNoError
}

func KMemoryReleasePage(kPage int) (unsafe.Pointer, JoyError) {
	if kPage < 0 || kPage >= KNumPages {
		return nil, MakeError(ErrorMemoryBadPageRequest)
	}
	isAlreadyFree, err := KMemoryIsFree(kPage)
	if err != JoyNoError {
		return nil, err
	}
	if isAlreadyFree {
		return nil, MakeError(ErrorMemoryAlreadyFree)
	}
	return KMemorySetNotInUse(kPage)
}

func KMemoryGetFreePage() (unsafe.Pointer, JoyError) {
	for i := 0; i < KNumPages; i++ {
		ok, err := KMemoryIsFree(i)
		if err != JoyNoError {
			return nil, err
		}
		if ok {
			trust.Infof("KMemoryGetFreePage: found free page %d", i)
			return KMemorySetInUse(i)
		}
	}
	return nil, MakeError(ErrorMemoryPageNotAvailable)
}

func KMemoryIsFree(kPage int) (bool, JoyError) {
	if kPage < 0 || kPage >= KNumPages {
		return false, MakeError(ErrorMemoryBadPageRequest)
	}
	bits := pageNumberToBits(kPage)
	return bits == 0, JoyNoError
}

func pageNumberToBits(kPage int) uint64 {
	index := kPage >> 3
	shift := kPage % 64
	bit := uint64(1) << shift
	comp := ^bit
	result := KMemInUse[index] & comp
	return result >> shift
}

func KMemorySetNotInUse(kPage int) (unsafe.Pointer, JoyError) {
	return KMemoryChangeState(kPage, false)
}

func KMemorySetInUse(kPage int) (unsafe.Pointer, JoyError) {
	return KMemoryChangeState(kPage, true)
}

func KMemoryChangeState(kPage int, isSet bool) (unsafe.Pointer, JoyError) {
	if kPage < 0 || kPage >= KNumPages {
		return nil, MakeError(ErrorMemoryBadPageRequest)
	}
	bits := pageNumberToBits(kPage)
	if bits != 0 {
		return nil, MakeError(ErrorMemoryPageAlreadyInUse)
	}
	index := kPage >> 3
	shift := kPage % 64
	bit := uint64(1) << shift
	if isSet {
		KMemInUse[index] |= bit
		trust.Infof("KMemorySetInUse: allocated page %d", kPage)
	} else {
		comp := ^bit
		KMemInUse[index] &= comp
		trust.Infof("KMemorySetInUse: freed page %d", kPage)
	}
	return unsafe.Pointer(uintptr(KRamStart) + uintptr(kPage*PageSize)), JoyNoError
}
