package main

import (
	"errors"
	"fmt"
	"unsafe"

	"device/arm"
	"machine"
	"runtime/volatile"

	"boot/anticipation"
	"lib/loader"
	"lib/trust"
	"lib/upbeat"
)

//these are indices into the MAIR register
const MemoryDeviceNoGatherNoReorderNoEarlyWriteAck = 0
const MemoryNoCache = 1
const MemoryNormal = 2

//these are values for the MAIR register
const MemoryDeviceNoGatherNoReorderNoEarlyWriteAckValue = 0x00 //it's hardware regs
const MemoryNoCacheValue = 0x44                                //not inner or outer cacheable
const MemoryNormalValue = 0xFF                                 //cache all you want, including using TLB

//drop bottom 64k
const noLast16 = 0xffffffffffff0000

//export _enable_mmu_tables
func enableMMUTables(mairVal uint64, tcrVal uint64, sctrlVal uint64, ttbr0 uint64, ttbr1 uint64)

//export _enable_mmu_tables_other_core
func enableMMUTablesOtherCore(mairVal uint64, tcrVal uint64, sctrlVal uint64, ttbr0 uint64, ttbr1 uint64, core uint64) int

const isKernelAddrMask = 0xfffffc0000000000

const TTBR0Val = uint64(0x10000) //this is where we START our page tables, must be 64K aligned
const TTBR1Val = uint64(0x10000) //this is where we START our page tables, must be 64K aligned

var logger *trust.Logger

var buffer oneLine
var lr *lineRing
var started = false

var metal *anticipation.MetalByteBuster

const interval = 0x4000000

func wait() {
	amount := 1500000000
	if started {
		amount = 1500000
	}
	for i := 0; i < amount; i++ {
		arm.Asm("nop") //wait
	}
}

//go:extern anticipation_addr
var anticipation_addr uint64

//go:extern
func wait10000()

func main() {
	buffer = make([]uint8, anticipation.FileXFerDataLineSize)
	lr = newLineRing() //probably overkill since never need more than 1 line
	metal = anticipation.NewMetalByteBuster()

	info := upbeat.SetFramebufferRes1024x768()
	if info == nil {
		panic("giving up, can't set framebuffer res")
	}
	logger = upbeat.NewConsoleLogger(info)

	//log.Printf("clock time %d", runtime.Semihostingv2UnixTime())
	// we ignore errors because we are running on baremetal and there is literally
	// nothing we can do ... the error was in the part that lets us do "print"
	logger.Debugf("#\n")
	logger.Debugf("# Stage 0 running: Anticipation bootloader\n")
	logger.Debugf("# Running from physical address 0x%x\n",
		(uint64)(uintptr(unsafe.Pointer(&anticipation_addr))))
	logger.Debugf("#\n")

	setupVM()

	//setup the mini uart so you can see output over serial
	machine.MiniUART = machine.NewUART()
	machine.MiniUART.Configure(&machine.UARTConfig{RXInterrupt: true})
	machine.MiniUART.WriteString("about to try to read the disk...\n")

	//this is to test to see if we need to a disk based boot
	ok := canBootFromDisk(logger)
	if ok {
		params := loader.NewKernelProcStartupInfo(maddie, 3)
		if err := params.KernelProcBootFromDisk(logger); err != loader.LoaderNoError {
			logger.Errorf("failed to load kernel process: %s", err)
		}
		logger.Debugf("Bootloader on Processor 0 deadlooping")
		for {
			arm.Asm("nop")
		}
	} else {
		trust.Infof("unable to locate disk or boot file, using serial...")
	}

	//tell the Interrupt controlller what's going on
	machine.IC.Enable1.SetAux()

	machine.QA7.LocalTimerControl.SetInterruptEnable()
	machine.QA7.LocalTimerControl.SetReloadValue(interval)
	machine.QA7.LocalTimerControl.SetTimerEnable()

	//route BOTH local timer and GPU to core 0 on IRQ
	machine.QA7.GPUInterruptRouting.IRQToCore0()
	machine.QA7.LocalInterrupt.SetCore0IRQ()

	//Tell Core0 which interrupts to consume
	machine.QA7.TimerInterruptControl[0].SetPhysicalNonSecureTimerIRQ()
	machine.QA7.IRQSource[0].SetGPU()
	upbeat.UnmaskDAIF()

	for {
		upbeat.MaskDAIF()
		if started {
			//we leave this loop with interrupts OFF
			break
		}
		upbeat.UnmaskDAIF()
		wait()
		upbeat.MaskDAIF()
	}
	machine.QA7.LocalTimerControl.ClearTimerEnable()

	//nothing to do but wait for interrupts, we use lr.next() to block
	//until we get a line, and lr.next implies interrupts are off
	for {
		s := lr.next(buffer)
		if len(s) == 0 {
			continue
		}
		done, err := processLine(s)
		if err != nil {
			machine.MiniUART.WriteString("! processing error:" + err.Error() + " " + s[0:16] + "\n")
		} else {
			machine.MiniUART.WriteString(". accept: " + s[0:16] + "\n")
		}
		if done {
			break
		}
	}
}

//export raw_exception_handler
func rawExceptionHandler(t uint64, esr uint64, addr uint64, el uint64, procId uint64) {
	if t != 5 {
		// this is in case we get some OTHER kind of exception
		logger.Errorf("raw exception handler:exception type %d and "+
			"esr %x with addr %x and EL=%d, ProcID=%x\n",
			t, esr, addr, el, procId)
		logger.Errorf("DEADLOOP\n")
		for {
			arm.Asm("nop")
		}
	}
	interruptReceive()
}

//go:noinline
func interruptReceive() {
	atLeastOne := true
	for atLeastOne {
		atLeastOne = false
		switch {
		//note that we don't jump into this case if there is a transmit intr
		case !machine.Aux.MUIIR.InterruptPendingIsSet():
			atLeastOne = true
			for {
				if !machine.Aux.MULSR.DataReadyIsSet() {
					break
				}
				//this is slightly dodgy, but since interrupts are off, it's ok
				if !started {
					started = true
				} else {
					//reset watchdog timer
					machine.QA7.LocalTimerClearReload.SetReload()
					machine.QA7.LocalTimerClearReload.SetClear() //sad nomenclature
				}
				//pull the character into the internal buffer
				ch := byte(machine.Aux.MUData.Receive())
				switch {
				case ch == 10:
					machine.MiniUART.LoadRx(10)
					moved := machine.MiniUART.CopyRxBuffer(buffer)
					lr.addLineToRing(string(buffer[:moved]))
				case ch < 32 || ch > 127:
					//nothing
				default:
					machine.MiniUART.LoadRx(ch) //put it in the receive buffer
				}
			}
		case machine.QA7.LocalTimerControl.InterruptPendingIsSet():
			//we really should do a lock here but becase we are running
			//on bare metal, we'll get away with this read of a shared
			//variable
			waitCount++
			atLeastOne = true
			if started {
				logger.Debugf("___________WATCHDOG! __________\n")
				//machine.MiniUART.WriteString(". watchdog timeout during transfer\n")
			} else {
				logger.Debugf("anticipation: local timer interrupt: #%03d", waitCount)
				machine.MiniUART.WriteString(fmt.Sprintf(". local timer interrupt: #%03d\n", waitCount))
			}
			machine.QA7.LocalTimerClearReload.SetClear() //ugh, nomenclature
			machine.QA7.LocalTimerClearReload.SetReload()
		}
	}
}

var waitCount = 0

func processLine(line string) (bool, error) {
	//really should do a lock here, but on baremetal will be ok
	waitCount = 0

	//clip off the LF that came from server
	end := len(line)
	if end > 0 && line[end-1] == 10 {
		end--
	}
	// just do what the line says
	converted, lt, _, err := anticipation.DecodeAndCheckStringToBytes(line[:end])
	if err != nil {
		return false, err
	}
	wasError, done := anticipation.ProcessLine(lt, converted, metal)
	if wasError {
		return false, errors.New("unable to execute line " + line)
	}
	if done {
		if !metal.EntryPointIsSet() {
			return false, errors.New("no entry point has been set")
		}
		// normally our CALLER does the confirm, but we are never going to
		// reach there
		machine.MiniUART.WriteString(".\n") //signal the sender everything is ok
		logger.Infof(" === jumping to kernel at address %x ===\n", metal.EntryPoint())
		upbeat.MaskDAIF() //turn off interrupts while we boot up the kernel
		//turn off the interrupts so we don't get them in kernel until we are ready
		machine.IC.Disable1.SetAux() //sadly, you *set* things in the DISable reg to turn off
		machine.QA7.LocalTimerControl.ClearTimerEnable()
		print("boot param 0: ", metal.GetParameter(0), "\n")
		jumpToKernel(metal.EntryPoint(), metal.GetParameter(0), metal.GetParameter(1),
			metal.GetParameter(2), metal.GetParameter(3))
	}
	//keep going
	return false, nil
}

//export jump_to_kernel
func jumpToKernel(ep uint64, blockPtr uint64, paramPtr uint64, _ uint64, _ uint64)

//export jump_to_kernel_proc
func jumpToKernelProc(procId uint64, ttbr1 uint64, entryPoint uint64, paramPtrSource uint64, paramPtrDest uint64, _ uint64)

var sleepOneTickAmount uint32

func clockCalibration(logger *trust.Logger) uint32 {
	n := uint64(1000)
	numIters := 10
	var z volatile.Register32
	sets := uint32(455000)
	logger.Debugf("Calibrating clock...")
outer:
	for iter := 0; iter < numIters; iter++ {
		sum := uint64(0)
		prev := machine.SystemTime()
		for i := uint64(0); i < n; i++ {
			for j := uint32(0); j < sets; j++ {
				z.Set(z.Get() + j)
			}
			x := machine.SystemTime()
			diff := x - prev
			sum += diff
			// log.Printf("xxx %d: %d, %d", i, x, x-prev)
			prev = x
		}
		avg := float64(sum) / float64(n)
		if avg < 1.005 && avg > .995 {
			break outer
		}
		if iter != numIters-1 {
			logger.Debugf("iteration: %d, %d,%4.2f", iter+1, z.Get(), avg)
		}
		if avg < 1.0 {
			adjustment := 1.0 / avg
			logger.Debugf("adjust up by %0.2f", (adjustment - 1.0))
			setsAsFloat := float64(sets) * adjustment
			sets = uint32(setsAsFloat)
		} else {
			adjustment := avg - 1.0
			logger.Debugf("adjust down by %0.2f", adjustment)
			change := float64(sets) * adjustment
			sets -= uint32(change)
		}
	}
	logger.Debugf("clock calibration: %d", sets)
	return sets
}

func sleepOneTick() {
	var z volatile.Register32
	for j := uint32(0); j < sleepOneTickAmount; j++ {
		z.Set(z.Get() + j)
	}
}
