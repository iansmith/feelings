all: clean kernel8.img kernel8.img.hardware

FEELINGS=/Users/iansmith/feelings
NAME=antc
FONTDIR=$(FEELINGS)/src/lib/upbeat

$(FONTDIR)/font.o: $(FONTDIR)/font.psf
	ld.lld -m aarch64elf -r -b binary -o $(FONTDIR)/font.o $(FONTDIR)/font.psf

$(NAME): *.go $(FONTDIR)/font.o
	GOPATH=$(FEELINGS) tinygo build -ldflags='$(FONTDIR)/font.o' -cflags='-g' -target $(NAME)_qemu.json -o $(NAME) .

$(NAME).hardware: *.go $(FONTDIR)/font.o
	GOPATH=$(FEELINGS) tinygo build  -ldflags='$(FONTDIR)/font.o' -target $(NAME).json -o $(NAME).hardware .

clean:
	rm kernel8.img $(NAME) $(NAME).hardware kernel8.img.hardware *.o >/dev/null 2>/dev/null || true

kernel8.img: $(NAME)
	llvm-objcopy -O binary $(NAME) kernel8.img

kernel8.img.hardware: $(NAME).hardware
	llvm-objcopy -O binary $(NAME).hardware kernel8.img.hardware

run:
	qemu-system-aarch64 -M raspi3 -kernel $(NAME) -serial null -serial chardev:char0 \
 		-d unimp,cpu_reset,guest_errors -semihosting -semihosting-config enable=on,target=native \
 		--chardev pty,path=/dev/ttys009,mux=on,id=char0

debug:
	qemu-system-aarch64 \
		-M raspi3 -S -s \
		-kernel $(NAME) \
		-semihosting -semihosting-config enable=on,target=native \
		-serial null -serial stdio