package loader

// these are the ladies' mappings
const TTBR1Value = 0x3000_0000 // + (2_0000 * proc id)

// these are the users' mappings
const UserTTBR0Value = 6_0000 // + 2_0000 * proc id
const IndentityTTBR0Value = 1_0000

// total addressable size of a kernel process is 63MB or 0x3ef_0000

const KernelProcessStack = 0x3ef /*in pages*/
const KernelProcessStackAddr = 0xffff_fc00_03ef_0000
const KernelProcessHeap = 0x200 /* in pages */
const KernelProcessHeapAddr = 0xffff_fc00_0200_0000

// total addressable size of a user process is 1GB or 0x3fff_0000

const UserProcessStack = 0x3fff /*in pages*/
const UserProcessStackAddr = 0x3fff_0000
const UserlProcessHeap = 0x2000 /* in pages */
const UserProcessHeapAddr = 0x2000_0000

//1MB of shared mapping (0x10_0000) up to heap
const KernelProcessSharedMapping = 0xffff_fc00_01F0_0000 // => PHYS 0x3010_0000

// kernel process bss is just before shared mappings (no exec)
// kernel process rw memory is next (no exec)
// kernel process ro memory is right above kernel process code (no write, no exec)
const KernelProcessLinkAddr = 0xffff_fc00_0001_0000 //(no write)

// user process bss is just before any shared mappings (no exec)
// user process rw memory is next (no exec)
// user process ro memory is right above kernel process code (no write, no exec)
const UserProcessLinkAddr = 0x1_0000 //(no write)

const UserProcessLowestPhys = 0xe_0000

const MadeleineProcessor = 3

// this is the PHYS addr that the bootloader will place Madeleine at
// after it sets up the tables on appropriate processor.
const MadeleinePlacement = 0x3020_0000 // just above Kernel Shared Mappings

//
// VM TABLES (PHYS ADDR)
//
// 0x3000_0000 -> 0x3007_fff8  (2 memory pages per processor)
const MadelineTableLevel2 = 0x3006_0000      //page 7 in kernel area
const MadelineTableLevel3Start = 0x3007_0000 //page 8 in kernel area

//
// KERNEL SHARED MAPPINGS AREA (PHYS ADDR)
//
//pages 0x3010_0000 to 0x301F_0000 are intended for sharing
const KernelSharedMappings = 0x3010_0000
