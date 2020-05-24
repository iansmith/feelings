all: clean kernel8.img kernel8.img.hardware
FEELINGS=/Users/iansmith/feelings
NAME=timer

$(NAME): *.go
	GOPATH=$(FEELINGS) tinygo build -target rpi3_qemu -o $(NAME) .

$(NAME).hardware: *.go
	GOPATH=$(FEELINGS) tinygo build -target rpi3 -o $(NAME).hardware .

clean:
	rm kernel8.img $(NAME) $(NAME).hardware kernel8.img.hardware *.o >/dev/null 2>/dev/null || true

kernel8.img: $(NAME)
	llvm-objcopy -O binary $(NAME) kernel8.img

kernel8.img.hardware: $(NAME).hardware
	llvm-objcopy -O binary $(NAME).hardware kernel8.img.hardware

font.o: font.psf
	ld.lld -m aarch64elf -r -b binary -o font.o font.psf

run:
	qemu-system-aarch64 -M raspi3 -kernel kernel8.img -serial null -serial stdio \
		-semihosting -semihosting-config enable=on,target=native

#-chardev pty,id=char1
#-drive file=outfile.bin,if=sd,format=raw

#-drive file=test.dd,if=sd,format=raw
#-drive file=/Users/iansmith/outfile.bin,if=sd,format=raw,drive=sd0


debug:
	qemu-system-aarch64 \
		-M raspi3 -S -s \
		-kernel timer \
		-semihosting -semihosting-config enable=on,target=native \
		-serial null -serial stdio
