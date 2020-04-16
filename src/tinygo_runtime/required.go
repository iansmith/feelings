package tinygo_runtime

import "github.com/tinygo-org/tinygo/src/device/arm"

type BaremetalRT struct {
}

func (b *BaremetalRT) Putchar(c byte) int {
	MiniUART.WriteByte(c)
	return 0
}
func (b *BaremetalRT) Abort() {
	MiniUART.WriteString("Aborting...")
	for {
		arm.Asm("nop")
	}
}
func (b *BaremetalRT) PostInit() {
}

func (b *BaremetalRT) Ticks() int64 {
	return 0
}

func (b *BaremetalRT) SleepTicks(_ int64) {
	return
}
