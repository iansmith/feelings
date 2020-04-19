file timer
layout regs
target remote:1234
#hbreak *0x96a88 if $x30 == 0
#hbreak *0x96b24 if $x30 == 0
#hbreak *0x80818
b *0x80e1c
