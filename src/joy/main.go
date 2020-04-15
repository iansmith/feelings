package main

import "machine"

func main() {
	machine.MiniUART.Configure(machine.UARTConfig{})
	machine.MiniUART.WriteString("hello, world.\n")
}
