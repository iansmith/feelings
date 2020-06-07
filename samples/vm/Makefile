all: clean kernel8.img kernel8.img.hardware

FEELINGS=/Users/iansmith/feelings
NAME=vm
FONTDIR=$(FEELINGS)/src/lib/upbeat
ASSEMBLY=../../tinygo_conf/assembly/

$(FONTDIR)/font.o: $(FONTDIR)/font.psf
	ld.lld -m aarch64elf -r -b binary -o $(FONTDIR)/font.o $(FONTDIR)/font.psf

$(NAME): *.go $(FONTDIR)/font.o
	clang --target=aarch64-elf -c -o $(ASSEMBLY)/set_regs.o ./set_regs.S
	GOPATH=$(FEELINGS) tinygo build -ldflags='$(FONTDIR)/font.o $(ASSEMBLY)/set_regs.o' -cflags='-g' -target rpi3_qemu -o $(NAME) .

$(NAME).hardware: *.go $(FONTDIR)/font.o
	GOPATH=$(FEELINGS) tinygo build  -ldflags='$(FONTDIR)/font.o $(ASSEMBLY)/set_regs.o' -target rpi3 -o $(NAME).hardware .

clean:
	rm vm kernel8.img *.o kernel8.img.hardware $(NAME) $(NAME).hardware $(ASSEMBLY)/set_regs.o >/dev/null 2>/dev/null || true

kernel8.img: $(NAME)
	llvm-objcopy -O binary $(NAME) kernel8.img

kernel8.img.hardware: $(NAME).hardware
	llvm-objcopy -O binary $(NAME).hardware kernel8.img.hardware


run:
	qemu-system-aarch64 -M raspi3 -kernel $(NAME) -serial null -serial stdio \
 		-d unimp,cpu_reset,guest_errors,mmu -semihosting -semihosting-config enable=on,target=native

#-chardev pty,id=char1
#-drive file=outfile.bin,if=sd,format=raw

#-drive file=test.dd,if=sd,format=raw
#-drive file=/Users/iansmith/outfile.bin,if=sd,format=raw,drive=sd0


debug:
	qemu-system-aarch64  \
		-M raspi3 -S -s \
		-kernel $(NAME) \
		-semihosting -semihosting-config enable=on,target=native \
		-serial null -serial stdio
