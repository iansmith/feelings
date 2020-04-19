all: clean kernel8.img

# we use the bare metal linker script for the bootloader
uart: *.go
	clang --target=aarch64-elf -c -o ../../tinygo_conf/assembly/baremetal_boot.o ../../src/hardware/arm-cortex-a53/barmetal_boot.S
	tinygo build -target ../../tinygo_conf/rpi3.json -ldflags='../../tinygo_conf/assembly/baremetal_boot.o -T ../../tinygo_conf/rpi3.ld' -o uart .

clean:
	rm uart kernel8.img *.o >/dev/null 2>/dev/null || true

kernel8.img: uart
	llvm-objcopy -O binary uart kernel8.img

run:
	qemu-system-aarch64 -M raspi3 -kernel kernel8.img -serial null -serial stdio
