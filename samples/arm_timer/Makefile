all: clean kernel8.img

# we use the bare metal linker script for the bootloader
timer: *.go
	clang --target=aarch64-elf -c -o ../../tinygo_conf/assembly/baremetal_boot.o ../../src/hardware/arm-cortex-a53/barmetal_boot.S
	clang --target=aarch64-elf -c -o ../../tinygo_conf/assembly/semihosting.o ../../src/hardware/arm-cortex-a53/semihosting.S
	tinygo build -target ../../tinygo_conf/rpi3.json -ldflags='../../tinygo_conf/assembly/baremetal_boot.o ../../tinygo_conf/assembly/semihosting.o -T ../../tinygo_conf/rpi3.ld' -o timer .

clean:
	rm timer kernel8.img *.o >/dev/null 2>/dev/null || true

kernel8.img: timer
	llvm-objcopy -O binary timer kernel8.img

run:
	qemu-system-aarch64 -M raspi3 -kernel kernel8.img -serial null -serial stdio

debug:
	qemu-system-aarch64  \
		-M raspi3 -S -s \
		-kernel kernel8.img \
		-semihosting -semihosting-config enable=on,target=native \
		-serial null -serial stdio
