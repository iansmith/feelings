package bootloader

const TTBR1Value = 0x3000_0000 // + (2_0000 * proc id)

// total addressable size of a kernel process is 64MB or 0x400_0000

// this is  the start of first page of stack, not root of stack
// which is 0xffff_fc00_00ff_fff0
const KernelProcessStackAddr = 0xffff_fc00_03ff_0000
const KernelProcessHeapAddr = 0xffff_fc00_0200_0000

//1MB of shared mapping (0x10_0000), but only using 1
const KernelProcessSharedMapping = 0xffff_fc00_01F0_0000 // => PHYS 0x3010_0000

// kernel process bss is just before heap (no exec)
// kernel process rw memory is next (no exec)
// kernel process ro memory is right above kernel process code (no write, no exec)
const KernelProcessLinkAddr = 0xffff_fc00_0000_0000 //(no write)

const MadeleineProcessor = 3

// this is the PHYS addr that the bootloader will place Madeleine at
// after it sets up the tables on appropriate processor.  this is
// the key
const MadeleinePlacement = 0x3020_0000 // just above Kernel Shared Mappings

//
// VM TABLES (PHYS ADDR)  Note: 0x0008_0000 gap to Kernel Shared Mappings
//
// 0x3000_0000 -> 0x3007_fff8  (2 memory pages per processor)
const MadelineTableLevel2 = 0x3006_0000      //page 7 in kernel area
const MadelineTableLevel3Start = 0x3007_0000 //page 8 in kernel area
// kernel vm tables in the first 8 pages of ram of kernel area
// this is one nearly wasted page (only one entry) and then a second
// one that has one entry in table per kernel page assigned to process
const KernelProcessVMTableStart = 0x3000_0000 //+ (2_0000 * proc id)

//
// KERNEL SHARED MAPPINGS AREA (PHYS ADDR)
//
//pages 0x3010_0000 to 0x301F_0000 are intended for sharing
const KernelSharedMappings = 0x3010_0000
