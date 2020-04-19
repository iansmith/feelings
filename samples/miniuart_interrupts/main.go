package main

import (
	arm64 "feelings/src/hardware/arm-cortex-a53"
	"feelings/src/hardware/bcm2835"
	rt "feelings/src/tinygo_runtime"
	"runtime/volatile"
)

func main() {
	rt.MiniUART = rt.NewUART()
	rt.MiniUART.Configure(rt.UARTConfig{RXInterrupt: true})

	//interrupts start as off
	arm64.InitExceptionVector()
	//all interrupts are "unexpected" until we set this
	arm64.SetExceptionHandlerEl1hInterrupts(miniUARTReceive)

	rt.MiniUART.WriteString("hello, uart")
	rt.MiniUART.WriteCR()
	rt.MiniUART.WriteString("type some text, then hit <return>")
	rt.MiniUART.WriteCR()

	//tell the interrupt controller what we want and then unmask interrupts
	bcm2835.InterruptController.EnableIRQs1.SetBits(bcm2835.AuxInterrupt)
	arm64.UnmaskDAIF()

	//nothing to do but wait for interrupts, so we count forever
	//gotta use something volatile or the optimizer will get rid of it
	r := volatile.Register32{}
	for {
		for i := 0x0; i < 0xffffffff; i++ {
			r.Set(r.Get() + 1)
		}
		rt.MiniUART.WriteByte(byte('.'))
	}
}

//go:export
var value *uint64

func miniUARTReceive(t uint64, esr uint64, addr uint64) {
	//this ignores the possibility that HasBits(6) because docs (!)
	//say that bits 2 and 1 cannot both be set, so we just check bit 2
	if bcm2835.Aux.MiniUARTInterruptIdentify.HasBits(4) {
		//pull the character into the internal buffer
		ch := rt.MiniUART.ReadByte()
		if ch != 13 {
			rt.MiniUART.WriteByte(ch) //echo it back, so typist can see it
			rt.MiniUART.LoadRx(ch)    //put it in the receive buffer
		} else {
			rt.MiniUART.WriteCR()
			rt.MiniUART.WriteString("Line: ")
			rt.MiniUART.DumpRxBuffer()
			rt.MiniUART.WriteCR()
		}
	} else {
		rt.MiniUART.WriteString("Expected RX interrupt, but none found!")
	}
}
