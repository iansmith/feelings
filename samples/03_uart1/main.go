package main

import (
	"machine"
	"runtime/volatile"
)

func main() {
	machine.MiniUART = machine.NewUART()
	machine.MiniUART.Configure(machine.UARTConfig{RXInterrupt: true})
	//interrupts start as off
	machine.InitInterrupts()
	//all interrupts are "unexpected" until we set this
	machine.SetExceptionHandlerEl1hInterrupts(miniUARTReceive)

	machine.MiniUART.WriteString("hello, uart")
	machine.MiniUART.WriteCR()

	//tell the interrupt controller what we want and then unmask interrupts
	machine.InterruptController.EnableIRQs1.SetBits(machine.AuxInterrupt)
	machine.UnmaskDAIF()

	//nothing to do but wait for interrupts, so we count forever
	//gotta use something volatile or the optimizer will get rid of it
	r := volatile.Register32{}
	for {
		for i := 0x0; i < 0xffffffff; i++ {
			r.Set(r.Get() + 1)
		}
		machine.MiniUART.WriteByte(byte('.'))
	}
}

//go:export
var value *uint64

func miniUARTReceive(t uint64, esr uint64, addr uint64) {
	//this ignores the possibility that HasBits(6) because docs (!)
	//say that bits 2 and 1 cannot both be set, so we just check bit 2
	if machine.Aux.MiniUARTInterruptIdentify.HasBits(4) {
		//pull the character into the internal buffer
		ch := machine.MiniUART.ReadByte()
		if ch != 13 {
			machine.MiniUART.WriteByte(ch) //echo it back, so typist can see it
			machine.MiniUART.LoadRx(ch)    //put it in the receive buffer
		} else {
			machine.MiniUART.WriteCR()
			machine.MiniUART.WriteString("Line: ")
			machine.MiniUART.DumpRxBuffer()
			machine.MiniUART.WriteCR()
		}
	} else {
		machine.MiniUART.WriteString("Expected RX interrupt, but none found!")
	}
}
