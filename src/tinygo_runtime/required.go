package tinygo_runtime

import (
)

//go:extern _sbss
var _sbss [0]byte

//go:extern _ebss
var _ebss [0]byte

//export runtime.export_preinit
func preinit() {
}
