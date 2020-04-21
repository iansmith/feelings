all: clean kernel8.img

ASM_SRCS=../../src/hardware/arm-cortex-a53/barmetal_boot.S \
../../src/hardware/arm-cortex-a53/semihosting.S \
exception.S

ASM_OBJS=../../tinygo_conf/assembly/baremetal_boot.o \
../../tinygo_conf/assembly/semihosting.o \
../../tinygo_conf/assembly/exception.o


# we use the bare metal linker script for the bootloader
semi: *.go ../../src/hardware/arm-cortex-a53/*.S
	clang --target=aarch64-elf -c -o ../../tinygo_conf/assembly/baremetal_boot.o ../../src/hardware/arm-cortex-a53/barmetal_boot.S
	clang --target=aarch64-elf -c -o ../../tinygo_conf/assembly/semihosting.o ../../src/hardware/arm-cortex-a53/semihosting.S
	clang --target=aarch64-elf -c -o ../../tinygo_conf/assembly/exception.o exception.S
	tinygo build -target ../../tinygo_conf/rpi3.json -ldflags="$(ASM_OBJS) -T ../../tinygo_conf/rpi3.ld" -o semi .

clean:
	rm semi kernel8.img *.o $(ASM_OBJS) >/dev/null 2>/dev/null || true

kernel8.img: semi
	llvm-objcopy -O binary semi kernel8.img

run:
	qemu-system-aarch64 -M raspi3 -kernel kernel8.img -serial null -serial stdio

debug:
	qemu-system-aarch64  \
		-M raspi3 -S -s \
		-kernel kernel8.img \
		-semihosting -semihosting-config enable=on,target=native \
		-serial null -serial stdio
