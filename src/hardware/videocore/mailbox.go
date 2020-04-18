package videocore

import (
	"feelings/src/hardware/rpi"
	"unsafe"

	"github.com/tinygo-org/tinygo/src/runtime/volatile"
)

var Mailbox *MailboxRegisterMap = (*MailboxRegisterMap)(unsafe.Pointer(rpi.MemoryMappedIO + 0x0000B880))

const Full = 0x80000000
const Empty = 0x40000000
const Response = 0x40000000

type MailboxRegisterMap struct {
	Read     volatile.Register32    //0x00
	reserved [3]volatile.Register32 //0x04-0x10
	Poll     volatile.Register32    //0x10
	Sender   volatile.Register32    // 0x14
	Status   volatile.Register32    // 0x18
	Config   volatile.Register32    //0x1c
	Write    volatile.Register32    //0x20
}

var mboxBuffer [36]volatile.Register32

/*
func Call(ch uint8) bool{
	unsigned int r = (((unsigned int)((unsigned long)&mbox)&~0xF) | (ch&0xF));

}
*/

/**
 * Make a mailbox call. Returns 0 on failure, non-zero on success
 */

//int mbox_call(unsigned char ch)
//{
//unsigned int r = (((unsigned int)((unsigned long)&mbox)&~0xF) | (ch&0xF));
///* wait until we can write to the mailbox */
//do{asm volatile("nop");}while(*MBOX_STATUS & MBOX_FULL);
///* write the address of our message to the mailbox with channel identifier */
//*MBOX_WRITE = r;
///* now wait for the response */
//while(1) {
///* is there a response? */
//do{asm volatile("nop");}while(*MBOX_STATUS & MBOX_EMPTY);
///* is it a response to our message? */
//if(r == *MBOX_READ)
///* is it a valid successful response? */
//return mbox[1]==MBOX_RESPONSE;
//}
//return 0;
//}
