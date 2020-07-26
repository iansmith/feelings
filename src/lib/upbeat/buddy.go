package upbeat

// order 0 : 1 page
// order 1 : 2 pages (128K)
// order 2: 4 pages (256k)
// order 3: 8 pages (512K)
// order 4: 16 pages (1024K) (1M)
// order 5: 32 pages (2048K) (2M)
// order 6: 64 pages (4096K) (4M)
// order 7: 128 pages (8192K) (8M)
// order 8: 256 pages (16384K) (16M)
// order 9: 512 pages (32768K) (32M)
// order 10: 1024 pages (65536K) (64M)
// order 11: 2048 pages (131072K) (128M)
// order 12: 4096 pages (262144K) (256M)
// order 13: 8192 pages (524288K) (512M)

// distance from VM tables to kernel space is
// 0x2FF0 pages => 12272 or

type PageBuddy struct {
	start int
	end   int
}

const maxGuesses = 5

func NewPageBuddy(startPage int, endPage int) *PageBuddy {
	return nil
}

//
// This is the data storage for the various linked lists that make up
// the buddy lists for each level.

type Buddy struct {
	startPage uint64 // inclusive
	endPage   uint64 // exclusive
}

// FirstAddress is the address of the first byte of the buddy segment.
func (b *Buddy) FirstAddress() uint64 {
	return b.startPage * 0x10000
}

// LastAddressExclusive is the address of the first byte PAST the end of
// the buddy segment.
func (b *Buddy) LastAddressExclusive() uint64 {
	return b.endPage * 0x10000
}
