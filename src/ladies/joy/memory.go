package joy

import (
	"boot/bootloader"
	"lib/trust"

	"unsafe"
)

type KPageId uint16

const NoKPageId KPageId = 0xffff

// just for cleanliness inside joy
type KMemDef interface {
	Init() JoyError //called at startup to cleanup stuff done by bootloader
	PageSize() uint64
	RAMStart() uint64 //first addr of kernel space
	InUse(KPageId) (bool, JoyError)
	Release(KPageId) (KPageId, JoyError)
	Get() (KPageId, JoyError)
	GetContiguousPages(n int) (KPageId, KPageId, JoyError)
	ToPtr(KPageId) unsafe.Pointer
}
type kmemAPIImpl struct{}

var KMemAPI KMemDef = &kmemAPIImpl{}

func (k *kmemAPIImpl) Init() JoyError {
	return kmemInit()
}
func (k *kmemAPIImpl) PageSize() uint64 {
	return kpageSize
}
func (k *kmemAPIImpl) RAMStart() uint64 {
	return kramStart
}
func (k *kmemAPIImpl) InUse(pg KPageId) (bool, JoyError) {
	b, err := kmemIsFree(pg)
	if err != JoyNoError {
		return false, err
	}
	return !b, JoyNoError
}
func (k *kmemAPIImpl) Release(pg KPageId) (KPageId, JoyError) {
	return kmemReleasePage(pg)
}
func (k *kmemAPIImpl) Get() (KPageId, JoyError) {
	return kmemGetFreePage()
}
func (k *kmemAPIImpl) GetContiguousPages(n int) (KPageId, KPageId, JoyError) {
	return kmemGetFreeContiguousPages(n)
}
func (k *kmemAPIImpl) ToPtr(id KPageId) unsafe.Pointer {
	return unsafe.Pointer(uintptr(kramStart) + uintptr((uint64(id) * kpageSize)))
}

const kpageSize = uint64(0x10000)            //64KB
const kramStart = uint64(0xfffffc0030000000) //kmemInit works around the kernel code,stack,heap

const knumPages = 766 //(because we loaded kernel code this is not 768)
const kinUseSize = 96 //bit vector composed of kinUseSize uint64s

const kProcStackPages = 2
const kProcHeapPages = 8

var kMemInUse [kinUseSize]uint64

// kmemInit sets up for familyImpl 0 and returns if everything that needed
// to be patched up (mostly heap) has been patched up.  This is a bit tricky
// because of the fact that we are allocating space for something that is
// already running (this thread) and whose SP is already set.
//go:noinline
func kmemInit() JoyError {

	trust.Infof("kmem init1")
	//kernel code page(s)
	pg := KPageId(0)
	_, err := kmemSetInUse(pg)
	if err != JoyNoError {
		return err
	}
	ptr := uintptr(KMemAPI.ToPtr(pg))
	kernelLast := (uint64(bootloader.InjectedParams.KernelCodePages()) * 0x10000) +
		bootloader.InjectedParams.KernelCodeStart - 8 //-8 so its a real addr
	for ptr+uintptr(kpageSize) < uintptr(kernelLast) {
		pg++
		_, err = kmemSetInUse(pg)
		if err != JoyNoError {
			return err
		}
		ptr = uintptr(KMemAPI.ToPtr(pg))
	}
	trust.Infof("kmem init2")
	//stack and heap was set up by the bootloader, but we want our data structs
	//to reflect this properly
	pg++
	_, err = kmemSetInUse(pg)
	if err != JoyNoError {
		return err
	}
	bottom := (*family)(KMemAPI.ToPtr(pg))
	ptr = uintptr(KMemAPI.ToPtr(pg))
	stackLast := (uint64(bootloader.InjectedParams.StackPages()) * 0x10000) +
		bootloader.InjectedParams.StackStart - 16 //16 byte align required

	for ptr+uintptr(kpageSize) < uintptr(stackLast) {
		pg++
		_, err = kmemSetInUse(pg)
		if err != JoyNoError {
			return err
		}
		ptr = uintptr(KMemAPI.ToPtr(pg))
	}
	trust.Infof("kmem init3")
	top := ptr + uintptr(kpageSize-16)
	*bottom = familyZero
	bottom.Stack = uint64(top)

	//we need setup heap
	start := bootloader.InjectedParams.HeapStart
	end := (uint64(bootloader.InjectedParams.HeapPages()) * 0x10000) +
		bootloader.InjectedParams.HeapStart - 8 //last real addr
	trust.Infof("kmem init4")
	pg++
	_, err = kmemSetInUse(pg)
	if err != JoyNoError {
		return err
	}
	ptr = uintptr(KMemAPI.ToPtr(pg))
	for ptr+uintptr(kpageSize) < uintptr(end) {
		pg++
		_, err = kmemSetInUse(pg)
		if err != JoyNoError {
			return err
		}
		ptr = uintptr(KMemAPI.ToPtr(pg))
	}

	trust.Infof("kmem init5")

	//kernel process init
	bottom.HeapStart = unsafe.Pointer(uintptr(start))
	bottom.HeapEnd = unsafe.Pointer(uintptr(end))
	currentFamily = (*family)(bottom)

	return JoyNoError
}

func kmemReleasePage(pg KPageId) (KPageId, JoyError) {
	if pg < 0 || pg >= knumPages {
		return NoKPageId, MakeError(ErrorMemoryBadPageRequest)
	}
	isAlreadyFree, err := kmemIsFree(pg)
	if err != JoyNoError {
		return NoKPageId, err
	}
	if isAlreadyFree {
		return NoKPageId, MakeError(ErrorMemoryAlreadyFree)
	}
	return kmemSetNotInUse(pg)
}

// returns the start of the 1st page and the *start* of the last page
// XXX HORRIBLE LOCK MISTAKE
func kmemGetFreeContiguousPages(n int) (KPageId, KPageId, JoyError) {
	if n < 1 {
		return 0, 0, MakeError(ErrorMemoryBadPageRequest)
	}
outer:
	for i := 0; i < knumPages-n; i++ {
		// LOCK LOCK LOCK, check then change!
		for j := 0; j < n; j++ {
			ok, err := kmemIsFree(KPageId(i + j))
			if err != JoyNoError {
				return NoKPageId, NoKPageId, err
			}
			if !ok {
				continue outer
			}
		}
		var resultStart, resultEnd KPageId
		//if we reach here, all j checked out
		for j := 0; j < n; j++ {
			_, err := kmemSetInUse(KPageId(i + j))
			if err != JoyNoError {
				return NoKPageId, NoKPageId, err
			}
			if j == 0 {
				resultStart = KPageId(i + j)
			}
			if j == n-1 {
				resultEnd = KPageId(i + j)
			}
		}
		return resultStart, resultEnd, JoyNoError
	}
	return NoKPageId, NoKPageId, MakeError(MemoryContiguousNotAvailable)
}

func kmemGetFreePage() (KPageId, JoyError) {
	for i := KPageId(0); i < knumPages; i++ {
		ok, err := kmemIsFree(i)
		if err != JoyNoError {
			return KPageId(i), err
		}
		if ok {
			return kmemSetInUse(i)
		}
	}
	return NoKPageId, MakeError(ErrorMemoryPageNotAvailable)
}

func kmemIsFree(pg KPageId) (bool, JoyError) {
	if pg < 0 || pg >= knumPages {
		return false, MakeError(ErrorMemoryBadPageRequest)
	}
	bits := pageNumberToBits(pg)
	return bits == 0, JoyNoError
}

func pageNumberToBits(pg KPageId) uint64 {
	kPage := uint64(pg)
	index := kPage >> 3
	shift := kPage % 64
	bit := uint64(1) << shift
	result := kMemInUse[index] & bit
	return result >> shift
}

//go:noinline
func kmemSetNotInUse(pg KPageId) (KPageId, JoyError) {
	return kmemChangeState(pg, false)
}

//go:noinline
func kmemSetInUse(pg KPageId) (KPageId, JoyError) {
	return kmemChangeState(pg, true)
}

//go:noinline
func kmemChangeState(pg KPageId, isSet bool) (KPageId, JoyError) {
	if pg < 0 || pg >= knumPages {
		return NoKPageId, MakeError(ErrorMemoryBadPageRequest)
	}
	bits := pageNumberToBits(pg)
	if bits != 0 && isSet {
		return NoKPageId, MakeError(ErrorMemoryPageAlreadyInUse)
	}
	if bits == 0 && !isSet {
		return NoKPageId, MakeError(ErrorMemoryPageAlreadyFree)
	}
	kPage := uint64(pg)
	index := kPage >> 3
	shift := kPage % 64
	bit := uint64(1) << shift
	if isSet {
		kMemInUse[index] |= bit
	} else {
		comp := ^bit
		kMemInUse[index] &= comp
	}
	return pg, JoyNoError
}
