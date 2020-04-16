package main

import "github.com/tinygo-org/tinygo/src/machine"

func main() {
	machine.MiniUART.WriteString("hello, world.\n")
	KExit(0)
}
