// +build rpi3 rpi3_qemu

package rpi

//This file is for things that are specific to the *model* Raspberry Pi 3 and
//are different on other rpi models.  Use the build tag rpi to get the generic
//properties of all Raspberry Pis.
const MemoryMappedIO = uintptr(0x3F000000)
