all: clean happiness

happiness: *.go ../../../../tinygo_conf/assembly/kernel_start.o
	clang --target=aarch64-elf -c -o ../../../../tinygo_conf/assembly/kernel_start.o ../../../hardware/arm-cortex-a53/kernel_start.S
	tinygo build  -target ../../../../tinygo_conf/rpi3.json -ldflags='-T ../../../../tinygo_conf/kernel.ld ../../../../tinygo_conf/assembly/kernel_start.o' -o happiness .

clean:
	rm happiness *.o >/dev/null 2>/dev/null || true

