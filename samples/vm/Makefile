all: clean kernel8.img

# we use the bare metal linker script for the bootloader
vm: *.go font.o
	clang --target=aarch64-elf -c -o ../../tinygo_conf/assembly/baremetal_boot.o ../../src/hardware/arm-cortex-a53/barmetal_boot.S
	tinygo build -target ../../tinygo_conf/rpi3.json -ldflags='../../tinygo_conf/assembly/baremetal_boot.o ./font.o -T ../../tinygo_conf/rpi3.ld' -o vm .

clean:
	rm vm kernel8.img *.o >/dev/null 2>/dev/null || true

kernel8.img: vm
	llvm-objcopy -O binary vm kernel8.img

font.o: font.psf
	ld.lld -m aarch64elf -r -b binary -o font.o font.psf

run:
	qemu-system-aarch64 -M raspi3 -kernel kernel8.img -serial null -serial stdio \

#-chardev pty,id=char1
#-drive file=outfile.bin,if=sd,format=raw

#-drive file=test.dd,if=sd,format=raw
#-drive file=/Users/iansmith/outfile.bin,if=sd,format=raw,drive=sd0


debug:
	qemu-system-aarch64  \
		-M raspi3 -S -s \
		-kernel kernel8.img \
		-semihosting -semihosting-config enable=on,target=native \
		-serial null -serial stdio
