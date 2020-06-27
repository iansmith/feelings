package joy

import (
	"lib/trust"
	"lib/upbeat"

	"unsafe"
)

const KPageSize = uint64(0x10000)            //64KB
const KRamStart = uint64(0xfffffc0030000000) //KMemoryInit works around the kernel code,stack,heap

//const KRamEnd = 0xfffffc0033000000 //actually 1 byte into next page so exclusive
const KNumPages = 766 //(because we loaded kernel code this is not 768)
const KInUseSize = 96 //bit vector

const KernelProcStackPages = 2
const KernelProcHeapPages = 8

var KMemInUse [KInUseSize]uint64

// KMemoryInint sets up for Domain 0 and returns if everything that needed
// to be patched up (mostly heap) has been patched up.  This is a bit tricky
// because of the fact that we are allocating space for something that is
// already running (this thread) and whose SP is already set.
func KMemoryInit() JoyError {

	//trust.Infof("kmemoryinit: %x", KRamStart)
	//our code page
	pg := 0
	pagePtr, err := KMemorySetInUse(pg)
	if err != JoyNoError {
		return err
	}
	trust.Debugf("code alloc %d", pg)
	ptr := uintptr(pagePtr)
	for ptr+uintptr(KPageSize) < uintptr(upbeat.BootloaderParams.KernelLast) {
		trust.Debugf("code alloc -- %d", pg)
		pg++
		pagePtr, err = KMemorySetInUse(pg)
		if err != JoyNoError {
			return err
		}
		ptr = uintptr(pagePtr)
	}
	//stack and heap was set up by the bootloader, but we want our data structs
	//to reflect this properly
	pg++
	trust.Debugf("stack alloc %d", pg)
	pagePtr, err = KMemorySetInUse(pg)
	if err != JoyNoError {
		return err
	}
	bottom := (*DomainControlBlock)(pagePtr)
	ptr = uintptr(pagePtr)
	for ptr+uintptr(KPageSize) < uintptr(upbeat.BootloaderParams.StackPointer) {
		pg++
		trust.Debugf("stack -- alloc %d", pg)
		pagePtr, err := KMemorySetInUse(pg)
		if err != JoyNoError {
			return err
		}
		ptr = uintptr(pagePtr)
	}
	top := ptr + uintptr(KPageSize-16)
	*bottom = DomainZero
	bottom.Stack = uint64(top)

	//we need setup heap
	start := upbeat.BootloaderParams.HeapStart
	end := upbeat.BootloaderParams.HeapEnd

	pg++
	pagePtr, err = KMemorySetInUse(pg)
	if err != JoyNoError {
		return err
	}
	trust.Debugf("heap alloc %d", pg)
	ptr = uintptr(pagePtr)
	for ptr+uintptr(KPageSize) < uintptr(end) {
		pg++
		pagePtr, err = KMemorySetInUse(pg)
		if err != JoyNoError {
			return err
		}
		trust.Debugf("heap alloc -- %d", pg)
		ptr = uintptr(pagePtr)
	}

	//kernel process init
	bottom.HeapStart = unsafe.Pointer(uintptr(start))
	bottom.HeapEnd = unsafe.Pointer(uintptr(end))
	CurrentDomain = (*DomainControlBlock)(bottom)

	return JoyNoError
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

// returns the start of the 1st page and the *start* of the last page
// XXX HORRIBLE LOCK MISTAKE
func KMemoryGetFreeContiguousPages(n int) (unsafe.Pointer, unsafe.Pointer, JoyError) {
	if n < 1 {
		return nil, nil, MakeError(ErrorMemoryBadPageRequest)
	}
outer:
	for i := 0; i < KNumPages-n; i++ {
		// LOCK LOCK LOCK, check then change!
		for j := 0; j < n; j++ {
			ok, err := KMemoryIsFree(i + j)
			if err != JoyNoError {
				return nil, nil, err
			}
			if !ok {
				continue outer
			}
		}
		var resultStart, resultEnd unsafe.Pointer
		//if we reach here, all j checked out
		for j := 0; j < n; j++ {
			ptr, err := KMemorySetInUse(i + j)
			if err != JoyNoError {
				return nil, nil, err
			}
			if j == 0 {
				resultStart = ptr
			}
			if j == n-1 {
				resultEnd = ptr
			}
		}
		return resultStart, resultEnd, JoyNoError
	}
	return nil, nil, MakeError(MemoryContiguousNotAvailable)
}

func KMemoryGetFreePage() (unsafe.Pointer, JoyError) {
	for i := 0; i < KNumPages; i++ {
		ok, err := KMemoryIsFree(i)
		if err != JoyNoError {
			return nil, err
		}
		if ok {
			//trust.Infof("KMemoryGetFreePage: found free page %d", i)
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
	result := KMemInUse[index] & bit
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
		//trust.Infof("KMemorySetInUse: allocated page %d", kPage)
	} else {
		comp := ^bit
		KMemInUse[index] &= comp
		//trust.Infof("KMemorySetInUse: freed page %d", kPage)
	}
	return unsafe.Pointer(uintptr(KRamStart) + (uintptr(kPage) * uintptr(KPageSize))), JoyNoError
}
