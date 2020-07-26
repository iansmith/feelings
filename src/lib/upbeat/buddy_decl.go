package upbeat

import "unsafe"

// xxx this big/small thing is still pretty inefficient and risky.  It really
// xxx should be generated from some memory model so the correct size of each
// xxx was correctly calculated.
// xxx this whole file should be done mechanically, not by hand.
const buddyListSizeBig = 256
const bitsetSizeBig = 4 // buddyListSize / 64
const buddyListSizeSmall = 64
const bitsetSizeSmall = 1 // buddyListSize / 64

var BuddyLists = [14]BuddyFixedDL{
	NewBuddyFixedDL(&nodePool0, &pool0),
	NewBuddyFixedDL(&nodePool1, &pool1),
	NewBuddyFixedDL(&nodePool2, &pool2),
	NewBuddyFixedDL(&nodePool3, &pool3),
	NewBuddyFixedDL(&nodePool4, &pool4),
	NewBuddyFixedDL(&nodePool5, &pool5),
	NewBuddyFixedDL(&nodePool6, &pool6),
	NewBuddyFixedDL(&nodePool7, &pool7),
	NewBuddyFixedDL(&nodePool8, &pool8),
	NewBuddyFixedDL(&nodePool9, &pool9),
	NewBuddyFixedDL(&nodePool10, &pool10),
	NewBuddyFixedDL(&nodePool11, &pool11),
	NewBuddyFixedDL(&nodePool12, &pool12),
	NewBuddyFixedDL(&nodePool13, &pool13),
}

var poolDataNode0 [buddyListSizeBig]BuddyNodeDL
var poolData0 [buddyListSizeBig]Buddy
var poolDataNode1 [buddyListSizeBig]BuddyNodeDL
var poolData1 [buddyListSizeBig]Buddy
var poolDataNode2 [buddyListSizeBig]BuddyNodeDL
var poolData2 [buddyListSizeBig]Buddy
var poolDataNode3 [buddyListSizeBig]BuddyNodeDL
var poolData3 [buddyListSizeBig]Buddy
var poolDataNode4 [buddyListSizeBig]BuddyNodeDL
var poolData4 [buddyListSizeBig]Buddy
var poolDataNode5 [buddyListSizeBig]BuddyNodeDL
var poolData5 [buddyListSizeBig]Buddy
var poolDataNode6 [buddyListSizeBig]BuddyNodeDL
var poolData6 [buddyListSizeBig]Buddy
var poolDataNode7 [buddyListSizeBig]BuddyNodeDL
var poolData7 [buddyListSizeBig]Buddy
var poolDataNode8 [buddyListSizeSmall]BuddyNodeDL
var poolData8 [buddyListSizeSmall]Buddy
var poolDataNode9 [buddyListSizeSmall]BuddyNodeDL
var poolData9 [buddyListSizeSmall]Buddy
var poolDataNode10 [buddyListSizeSmall]BuddyNodeDL
var poolData10 [buddyListSizeSmall]Buddy
var poolDataNode11 [buddyListSizeSmall]BuddyNodeDL
var poolData11 [buddyListSizeSmall]Buddy
var poolDataNode12 [buddyListSizeSmall]BuddyNodeDL
var poolData12 [buddyListSizeSmall]Buddy
var poolDataNode13 [buddyListSizeSmall]BuddyNodeDL
var poolData13 [buddyListSizeSmall]Buddy

var rawBitsForPool0 [bitsetSizeBig]uint64
var rawBitsForPoolNode0 [bitsetSizeBig]uint64
var rawBitsForPool1 [bitsetSizeBig]uint64
var rawBitsForPoolNode1 [bitsetSizeBig]uint64
var rawBitsForPool2 [bitsetSizeBig]uint64
var rawBitsForPoolNode2 [bitsetSizeBig]uint64
var rawBitsForPool3 [bitsetSizeBig]uint64
var rawBitsForPoolNode3 [bitsetSizeBig]uint64
var rawBitsForPool4 [bitsetSizeBig]uint64
var rawBitsForPoolNode4 [bitsetSizeBig]uint64
var rawBitsForPool5 [bitsetSizeBig]uint64
var rawBitsForPoolNode5 [bitsetSizeBig]uint64
var rawBitsForPool6 [bitsetSizeBig]uint64
var rawBitsForPoolNode6 [bitsetSizeBig]uint64
var rawBitsForPool7 [bitsetSizeBig]uint64
var rawBitsForPoolNode7 [bitsetSizeBig]uint64
var rawBitsForPool8 [bitsetSizeSmall]uint64
var rawBitsForPoolNode8 [bitsetSizeSmall]uint64
var rawBitsForPool9 [bitsetSizeSmall]uint64
var rawBitsForPoolNode9 [bitsetSizeSmall]uint64
var rawBitsForPool10 [bitsetSizeSmall]uint64
var rawBitsForPoolNode10 [bitsetSizeSmall]uint64
var rawBitsForPool11 [bitsetSizeSmall]uint64
var rawBitsForPoolNode11 [bitsetSizeSmall]uint64
var rawBitsForPool12 [bitsetSizeSmall]uint64
var rawBitsForPoolNode12 [bitsetSizeSmall]uint64
var rawBitsForPool13 [bitsetSizeSmall]uint64
var rawBitsForPoolNode13 [bitsetSizeSmall]uint64

var pool0 = BuddyManagedPool{
	elements: unsafe.Pointer(&poolData0[0]),
	bitset:   NewBitSet(buddyListSizeBig, uintptr(unsafe.Pointer(&rawBitsForPool0[0]))),
	num:      buddyListSizeBig,
	size:     int(unsafe.Sizeof(Buddy{})),
}

var nodePool0 = BuddyNodeDLManagedPool{
	elements: unsafe.Pointer(&poolDataNode0[0]),
	bitset:   NewBitSet(buddyListSizeBig, uintptr(unsafe.Pointer(&rawBitsForPoolNode0[0]))),
	num:      buddyListSizeBig,
	size:     int(unsafe.Sizeof(BuddyNodeDL{})),
}

var pool1 = BuddyManagedPool{
	elements: unsafe.Pointer(&poolData1[0]),
	bitset:   NewBitSet(buddyListSizeBig, uintptr(unsafe.Pointer(&rawBitsForPool1[0]))),
	num:      buddyListSizeBig,
	size:     int(unsafe.Sizeof(Buddy{})),
}

var nodePool1 = BuddyNodeDLManagedPool{
	elements: unsafe.Pointer(&poolDataNode1[0]),
	bitset:   NewBitSet(buddyListSizeBig, uintptr(unsafe.Pointer(&rawBitsForPoolNode1[0]))),
	num:      buddyListSizeBig,
	size:     int(unsafe.Sizeof(BuddyNodeDL{})),
}
var pool2 = BuddyManagedPool{
	elements: unsafe.Pointer(&poolData2[0]),
	bitset:   NewBitSet(buddyListSizeBig, uintptr(unsafe.Pointer(&rawBitsForPool2[0]))),
	num:      buddyListSizeBig,
	size:     int(unsafe.Sizeof(Buddy{})),
}

var nodePool2 = BuddyNodeDLManagedPool{
	elements: unsafe.Pointer(&poolDataNode2[0]),
	bitset:   NewBitSet(buddyListSizeBig, uintptr(unsafe.Pointer(&rawBitsForPoolNode2[0]))),
	num:      buddyListSizeBig,
	size:     int(unsafe.Sizeof(BuddyNodeDL{})),
}

var pool3 = BuddyManagedPool{
	elements: unsafe.Pointer(&poolData3[0]),
	bitset:   NewBitSet(buddyListSizeBig, uintptr(unsafe.Pointer(&rawBitsForPool3[0]))),
	num:      buddyListSizeBig,
	size:     int(unsafe.Sizeof(Buddy{})),
}

var nodePool3 = BuddyNodeDLManagedPool{
	elements: unsafe.Pointer(&poolDataNode3[0]),
	bitset:   NewBitSet(buddyListSizeBig, uintptr(unsafe.Pointer(&rawBitsForPoolNode3[0]))),
	num:      buddyListSizeBig,
	size:     int(unsafe.Sizeof(BuddyNodeDL{})),
}

var pool4 = BuddyManagedPool{
	elements: unsafe.Pointer(&poolData4[0]),
	bitset:   NewBitSet(buddyListSizeBig, uintptr(unsafe.Pointer(&rawBitsForPool4[0]))),
	num:      buddyListSizeBig,
	size:     int(unsafe.Sizeof(Buddy{})),
}

var nodePool4 = BuddyNodeDLManagedPool{
	elements: unsafe.Pointer(&poolDataNode4[0]),
	bitset:   NewBitSet(buddyListSizeBig, uintptr(unsafe.Pointer(&rawBitsForPoolNode4[0]))),
	num:      buddyListSizeBig,
	size:     int(unsafe.Sizeof(BuddyNodeDL{})),
}

var pool5 = BuddyManagedPool{
	elements: unsafe.Pointer(&poolData5[0]),
	bitset:   NewBitSet(buddyListSizeBig, uintptr(unsafe.Pointer(&rawBitsForPool5[0]))),
	num:      buddyListSizeBig,
	size:     int(unsafe.Sizeof(Buddy{})),
}

var nodePool5 = BuddyNodeDLManagedPool{
	elements: unsafe.Pointer(&poolDataNode5[0]),
	bitset:   NewBitSet(buddyListSizeBig, uintptr(unsafe.Pointer(&rawBitsForPoolNode5[0]))),
	num:      buddyListSizeBig,
	size:     int(unsafe.Sizeof(BuddyNodeDL{})),
}
var pool6 = BuddyManagedPool{
	elements: unsafe.Pointer(&poolData6[0]),
	bitset:   NewBitSet(buddyListSizeBig, uintptr(unsafe.Pointer(&rawBitsForPool6[0]))),
	num:      buddyListSizeBig,
	size:     int(unsafe.Sizeof(Buddy{})),
}

var nodePool6 = BuddyNodeDLManagedPool{
	elements: unsafe.Pointer(&poolDataNode6[0]),
	bitset:   NewBitSet(buddyListSizeBig, uintptr(unsafe.Pointer(&rawBitsForPoolNode6[0]))),
	num:      buddyListSizeBig,
	size:     int(unsafe.Sizeof(BuddyNodeDL{})),
}
var pool7 = BuddyManagedPool{
	elements: unsafe.Pointer(&poolData7[0]),
	bitset:   NewBitSet(buddyListSizeBig, uintptr(unsafe.Pointer(&rawBitsForPool7[0]))),
	num:      buddyListSizeBig,
	size:     int(unsafe.Sizeof(Buddy{})),
}
var nodePool7 = BuddyNodeDLManagedPool{
	elements: unsafe.Pointer(&poolDataNode7[0]),
	bitset:   NewBitSet(buddyListSizeBig, uintptr(unsafe.Pointer(&rawBitsForPoolNode7[0]))),
	num:      buddyListSizeBig,
	size:     int(unsafe.Sizeof(BuddyNodeDL{})),
}
var pool8 = BuddyManagedPool{
	elements: unsafe.Pointer(&poolData8[0]),
	bitset:   NewBitSet(buddyListSizeSmall, uintptr(unsafe.Pointer(&rawBitsForPool8[0]))),
	num:      buddyListSizeSmall,
	size:     int(unsafe.Sizeof(Buddy{})),
}
var nodePool8 = BuddyNodeDLManagedPool{
	elements: unsafe.Pointer(&poolDataNode8[0]),
	bitset:   NewBitSet(buddyListSizeSmall, uintptr(unsafe.Pointer(&rawBitsForPoolNode8[0]))),
	num:      buddyListSizeSmall,
	size:     int(unsafe.Sizeof(BuddyNodeDL{})),
}
var pool9 = BuddyManagedPool{
	elements: unsafe.Pointer(&poolData9[0]),
	bitset:   NewBitSet(buddyListSizeSmall, uintptr(unsafe.Pointer(&rawBitsForPool9[0]))),
	num:      buddyListSizeSmall,
	size:     int(unsafe.Sizeof(Buddy{})),
}
var nodePool9 = BuddyNodeDLManagedPool{
	elements: unsafe.Pointer(&poolDataNode9[0]),
	bitset:   NewBitSet(buddyListSizeSmall, uintptr(unsafe.Pointer(&rawBitsForPoolNode9[0]))),
	num:      buddyListSizeSmall,
	size:     int(unsafe.Sizeof(BuddyNodeDL{})),
}
var pool10 = BuddyManagedPool{
	elements: unsafe.Pointer(&poolData10[0]),
	bitset:   NewBitSet(buddyListSizeSmall, uintptr(unsafe.Pointer(&rawBitsForPool10[0]))),
	num:      buddyListSizeSmall,
	size:     int(unsafe.Sizeof(Buddy{})),
}
var nodePool10 = BuddyNodeDLManagedPool{
	elements: unsafe.Pointer(&poolDataNode10[0]),
	bitset:   NewBitSet(buddyListSizeSmall, uintptr(unsafe.Pointer(&rawBitsForPoolNode10[0]))),
	num:      buddyListSizeSmall,
	size:     int(unsafe.Sizeof(BuddyNodeDL{})),
}
var pool11 = BuddyManagedPool{
	elements: unsafe.Pointer(&poolData11[0]),
	bitset:   NewBitSet(buddyListSizeSmall, uintptr(unsafe.Pointer(&rawBitsForPool11[0]))),
	num:      buddyListSizeSmall,
	size:     int(unsafe.Sizeof(Buddy{})),
}
var nodePool11 = BuddyNodeDLManagedPool{
	elements: unsafe.Pointer(&poolDataNode11[0]),
	bitset:   NewBitSet(buddyListSizeSmall, uintptr(unsafe.Pointer(&rawBitsForPoolNode11[0]))),
	num:      buddyListSizeSmall,
	size:     int(unsafe.Sizeof(BuddyNodeDL{})),
}
var pool12 = BuddyManagedPool{
	elements: unsafe.Pointer(&poolData12[0]),
	bitset:   NewBitSet(buddyListSizeSmall, uintptr(unsafe.Pointer(&rawBitsForPool12[0]))),
	num:      buddyListSizeSmall,
	size:     int(unsafe.Sizeof(Buddy{})),
}
var nodePool12 = BuddyNodeDLManagedPool{
	elements: unsafe.Pointer(&poolDataNode12[0]),
	bitset:   NewBitSet(buddyListSizeSmall, uintptr(unsafe.Pointer(&rawBitsForPoolNode12[0]))),
	num:      buddyListSizeSmall,
	size:     int(unsafe.Sizeof(BuddyNodeDL{})),
}
var pool13 = BuddyManagedPool{
	elements: unsafe.Pointer(&poolData13[0]),
	bitset:   NewBitSet(buddyListSizeSmall, uintptr(unsafe.Pointer(&rawBitsForPool13[0]))),
	num:      buddyListSizeSmall,
	size:     int(unsafe.Sizeof(Buddy{})),
}
var nodePool13 = BuddyNodeDLManagedPool{
	elements: unsafe.Pointer(&poolDataNode13[0]),
	bitset:   NewBitSet(buddyListSizeSmall, uintptr(unsafe.Pointer(&rawBitsForPoolNode13[0]))),
	num:      buddyListSizeSmall,
	size:     int(unsafe.Sizeof(BuddyNodeDL{})),
}
