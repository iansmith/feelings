all: clean kernel8.img kernel8.img.hardware

FEELINGS=/Users/iansmith/feelings
NAME=antc

# we use the bare metal linker script for the bootloader
$(NAME): *.go
	GOPATH=$(FEELINGS) tinygo build -cflags='-g' -target rpi3_qemu -o $(NAME) .
#	clang --target=aarch64-elf -mno-unaligned-access -c -o ../../../../tinygo_conf/assembly/baremetal_boot.o ../../../hardware/arm-cortex-a53/barmetal_boot.S
#	tinygo build -opt s -cflags='-mno-unaligned-access' -target  ../../../../tinygo_conf/rpi3.json -ldflags='../../../../tinygo_conf/assembly/baremetal_boot.o -T ../../../../tinygo_conf/rpi3.ld' -o antc .

$(NAME).hardware: *.go
	GOPATH=$(FEELINGS) tinygo build -target antc.json -o $(NAME).hardware .

clean:
	rm kernel8.img $(NAME) $(NAME).hardware kernel8.img.hardware *.o >/dev/null 2>/dev/null || true

kernel8.img: $(NAME)
	llvm-objcopy -O binary $(NAME) kernel8.img

kernel8.img.hardware: $(NAME).hardware
	llvm-objcopy -O binary $(NAME).hardware kernel8.img.hardware

# why is all this stuff here?
#run-old:
#	qemu-system-aarch64 -semihosting -semihosting-config enable=on,target=native,chardev=char0 \
#		--chardev pty,path=/dev/ttys009,mux=on,id=char0 \
#		-M raspi3\
#		-kernel kernel8.img \
#		-semihosting -semihosting-config enable=on,target=native,chardev=char0 \
#		-serial null -serial chardev:char0
run:
	qemu-system-aarch64 -M raspi3 -kernel kernel8.img -serial null -serial stdio \
		-d in_asm -semihosting -semihosting-config enable=on,target=native


#debug-old:
#	qemu-system-aarch64 -semihosting -semihosting-config enable=on,target=native,chardev=char0 \
#		--chardev pty,mux=on,id=char0 \
#		-M raspi3 -S -s \
#		-kernel kernel8.img \
#		-semihosting -semihosting-config enable=on,target=native,chardev=char0 \
#		-serial null -serial chardev:char0

debug:
	qemu-system-aarch64 \
		-M raspi3 -S -s \
		-kernel kernel8.img \
		-semihosting -semihosting-config enable=on,target=native \
		-serial null -serial stdio
