all: clean kernel8.img

# we use the bare metal linker script for the bootloader
antc: *.go
	tinygo build -target rpi3 -ldflags='-T ../../../../tinygo_delta/rpi3.ld' -o antc .

clean:
	rm antc kernel8.img *.o >/dev/null 2>/dev/null || true

kernel8.img: antc
	llvm-objcopy -O binary antc kernel8.img


run:
	qemu-system-aarch64 -semihosting -semihosting-config enable=on,target=native,chardev=char0 \
		--chardev pty,path=/dev/ttys009,mux=on,id=char0 \
		-M raspi3\
		-kernel kernel8.img \
		-serial null -serial chardev:char0

debug:
	qemu-system-aarch64 -semihosting -semihosting-config enable=on,target=native,chardev=char0 \
		--chardev pty,path=/dev/ttys009,mux=on,id=char0 \
		-M raspi3 -S -s \
		-kernel kernel8.img \
		-serial null -serial chardev:char0