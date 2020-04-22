all: clean kernel8.img

# we use the bare metal linker script for the bootloader
sdcard: *.go
	clang --target=aarch64-elf -c -o ../../tinygo_conf/assembly/baremetal_boot.o ../../src/hardware/arm-cortex-a53/barmetal_boot.S
	tinygo build -target ../../tinygo_conf/rpi3.json -ldflags='../../tinygo_conf/assembly/baremetal_boot.o -T ../../tinygo_conf/rpi3.ld' -o sdcard .

clean:
	rm sdcard kernel8.img *.o >/dev/null 2>/dev/null || true

kernel8.img: sdcard
	llvm-objcopy -O binary sdcard kernel8.img

run:
	qemu-system-aarch64 -M raspi3 -kernel kernel8.img -serial null -serial mon:stdio \
	-chardev pty,id=char1 \
	-drive file=/Users/iansmith/outfile.bin,if=sd,format=raw

#-drive file=/Users/iansmith/outfile.bin,if=sd,format=raw,drive=sd0


debug:
	qemu-system-aarch64  \
		-M raspi3 -S -s \
		-kernel kernel8.img \
		-semihosting -semihosting-config enable=on,target=native \
		-drive file=/Users/iansmith/outfile.bin,if=sd,format=raw \
		-serial null -serial stdio
