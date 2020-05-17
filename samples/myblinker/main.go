package main

import (
	"device/arm"
	"fmt"
	arm64 "hardware/arm-cortex-a53"
	"machine"
)

//export raw_exception_handler
func rawExceptionHandler(exType uint64, syndrome uint64, addr uint64) {
	print("rawExceptionHandler type=",exType, " syndrome=",uintptr(syndrome), " addr=",uintptr(addr), "\n")
	if first==0 {
		first=machine.SystemTime()
	}
	machine.ActivityLED(on)
	on=!on
	machine.ArmTimer.IRQClear.Set(0x1)//???
}

const periodInMuSecs = 1 * 1000 /*millis*/ * 1000 /*micros*/
var first uint64
var on=false
func main() {
	//need this if we are on real hardware
	machine.MiniUART=machine.NewUART()
	machine.MiniUART.Configure(&machine.UARTConfig{RXInterrupt: false})


	arm64.QuadA7.LocalInterruptRouting.Set(0)
	arm64.QuadA7.LocalTimerWriteFlags.SetBits(arm64.QuadA7TimerInterruptFlagClear |
		arm64.QuadA7TimerReload)
	arm64.QuadA7.GPUInterruptsRouting.Set(0b000) //make sure they go to core0
	arm64.QuadA7.Core0IRQSource.Set(arm64.QuadA7GPUFast)
	arm64.QuadA7.Core1IRQSource.Set(0)
	arm64.QuadA7.Core2IRQSource.Set(0)
	arm64.QuadA7.Core3IRQSource.Set(0)

	print("setting up timers....\n")
	//t:=machine.SystemTime()
	timerInterval:=uint32(100000)//3 million musecs
	rate, ok:=machine.GetClockRate()
	if !ok {
		panic("unable to get the GPU clock rate")
	}
	print("clock rate is ", rate," divider ", machine.ArmTimer.PreDivide.Get(),"\n")
	rate/=250 //there is a pre-divider of 250?  (bits 9:0 of reg 1C) -- docs suggest default is 126
	temp:=uint64(timerInterval)
	temp*=uint64(rate)
	temp/=1000000
	target:=uint32( temp & 0xffffffff) //32 bit value

	machine.ArmTimer.Control.ClearCounterEnableDisable()
	machine.ArmTimer.Control.SetNoPreScale()
	machine.ArmTimer.Control.ThirtyTwoBit()
	//machine.ArmTimer.Reload.SetBits(timerInterval)
	machine.InterruptController.EnableBasic.SetARMTimer()
	machine.ArmTimer.Control.SetInterruptEnableDisable()
	machine.ArmTimer.Control.SetCounterEnableDisable()
	machine.ArmTimer.IRQClear.Set(0x1)//???
	machine.ArmTimer.Load.Set(target)
	machine.ArmTimer.Reload.Set(target)

	arm.Asm("msr daifclr, #3") //IRQ and FIQ
	print("fcheck ",target,"\n")
	//for first==0 {
	//	arm.Asm("nop")
	//}
	//print("fok\n")
	//delta:=t-first
	//print("delta for ",timerInterval," is ", delta, "\n")
	for {
		for i:=0; i<100000000; i++ {
			arm.Asm("nop")
		}
		print("clock value?",machine.ArmTimer.Value.Get()," clock counter?", machine.ArmTimer.Counter.Get(),"\n")
	}
}

