package bcm2835

import "github.com/tinygo-org/tinygo/src/runtime/volatile"

type AuxPeripheralsRegisterMap struct {
	InterruptStatus           volatile.Register32 //0x00
	Enables                   volatile.Register32 //0x04
	reserved00                [14]uint32
	MiniUARTData              volatile.Register32 //0x40, 8 bits wide
	MiniUARTInterruptEnable   volatile.Register32 //0x44
	MiniUARTInterruptIdentify volatile.Register32 //0x48
	MiniUARTLineControl       volatile.Register32 //0x4C
	MiniUARTModemControl      volatile.Register32 //0x50
	MiniUARTLineStatus        volatile.Register32 //0x54, readonly
	MiniUARTModemStatus       volatile.Register32 //0x58, readonly
	MiniUARTScratch           volatile.Register32 //0x5C
	MiniUARTExtraControl      volatile.Register32 //0x60
	MiniUARTExtraStatus       volatile.Register32 //0x64
	MiniUARTBAUD              volatile.Register32 //0x68
	reserved01                [5]uint32
	SPI1ControlRegister0      volatile.Register32 //0x80
	SPI1ControlRegister1      volatile.Register32 //0x84
	SPI1Status                volatile.Register32 //0x88
	reserved02                volatile.Register32 //0x8C
	SPI1Data                  volatile.Register32 //0x90
	SPI1Peek                  volatile.Register32 //0x94
	reserved03                [10]uint32
	SPI2ControlRegister0      volatile.Register32 //0xC0
	SPI2ControlRegister1      volatile.Register32 //0xC4
	SPI2Status                volatile.Register32 //0xC8
	reserved04                volatile.Register32
	SPI2Data                  volatile.Register32 //0xD0
	SPI2Peek                  volatile.Register32 //0xD4
}

// mini uart: peripheral enable
const PeripheralMiniUART = 1 << 0

// mini uart: extra control bitfields
const ReceiveEnable = 1 << 0
const TransmitEnable = 1 << 1
const EnableRTS = 1 << 2
const EnableCTS = 1 << 3
const RTSFlowLevelMask = 0x1F //use with register32.ReplaceBits
const RTSFlowLevelFIFO3Spaces = 0 << 4
const RTSFlowLevelFIFO2Spaces = 1 << 4
const RTSFlowLevelFIFO1Space = 2 << 4
const RTSFlowLevelFIFO4Spaces = 3 << 4
const RTSAssertLevel = 1 << 6
const CTSAssertLevel = 1 << 7

// mini uart: line control register bitfields
//https://elinux.org/BCM2835_datasheet_errata
const DataLength8Bits = 3 << 0
const Break = 1 << 6
const DLab = 1 << 7

// mini uart: line control register bitfields
const ReadyToSend = 1 << 1

// mini uart: interrupt identify register bitfields
const Pending = 1 << 0
const TransmitInterruptsPending = 1 << 1 //Read
const ReceiveInterruptsPending = 2 << 1  //Read

const ClearFIFOsMask = 0x6       //use with register32.ReplaceBits
const ClearReceiveFIFO = 1 << 1  //Write
const ClearTransmitFIFO = 1 << 2 //Write

// mini uart: line status register bitfields
const ReceivedDataAvailable = 1 << 0
const ReceivedDataOverrun = 1 << 1
const TransmitFIFOSpaceAvailable = 1 << 5
const TransmitterIdle = 1 << 6

// mini uart: interrupt enable register bitfields
//https://elinux.org/BCM2835_datasheet_errata#p12 (does not explain two magic bits 3:2)
//https://github.com/LdB-ECM/Raspberry-Pi/blob/bc38ce183f731891d52a31df87df24904e466d0c/PlayGround/rpi-SmartStart.c#L255
const ReceiveFIFOReady = 1 << 0
const TransmitFIFOEmpty = 1 << 1
const LineStatusError = 1 << 2   //overrun error, parity error, framing error
const ModemStatusChange = 1 << 3 //changes to DSR/CTS
