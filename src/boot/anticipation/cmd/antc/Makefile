all: clean kernel8.img kernel8.img.hardware

ifndef FEELINGS
$(error FEELINGS variable is not set, see enable-feelings.sample)
endif
ifndef TINYGO
$(error TINYGO variable is not set, see enable-feelings.sample)
endif
ifndef LLVM
$(error LLVM variable is not set, see enable-feelings.sample)
endif
ifndef HOSTGO
$(error HOSTGO variable is not set, see enable-feelings.sample)
endif

NAME=antc
TINYGO_CLANG=$(LLVM)/bin/clang
TINYGO_OBJCOPY=$(LLVM)/bin/llvm-objcopy
TINYGO_LLD=$(LLVM)/bin/ld.lld
TINYGO_TINYGO=$(TINYGO)/build/tinygo

FONTDIR=$(FEELINGS)/src/lib/upbeat
BOOTASM=rpi3_baremetal_boot
MMUASM=mmu
SET_REGS=set_regs
BOOTLIB=$(FEELINGS)/src/boot/lib
LOADERLIB=$(FEELINGS)/src/lib/loader
TARGET=aarch64-none-elf

IMAGEFILE=$(HOME)/mnt/feelings.img

$(BOOTASM).o: $(BOOTLIB)/$(BOOTASM).S
	$(TINYGO_CLANG) -target $(TARGET) -c -o $(BOOTASM).o $(BOOTLIB)/$(BOOTASM).S

$(MMUASM).o: $(LOADERLIB)/$(MMUASM).S
	$(TINYGO_CLANG) -target $(TARGET) -c -o $(MMUASM).o $(LOADERLIB)/$(MMUASM).S

$(SET_REGS).o: $(SET_REGS).S
	$(TINYGO_CLANG) -target $(TARGET) -c -o $(SET_REGS).o $(SET_REGS).S

ALL_ASM_OBJS=$(BOOTASM).o $(MMUASM).o $(SET_REGS).o

font.psf: $(FONTDIR)/font.psf
	cp $(FONTDIR)/font.psf .

font.o: font.psf
	$(TINYGO_LLD) -m aarch64elf -r -b binary -o font.o font.psf

$(NAME): *.go font.o  $(ALL_ASM_OBJS)
	GOPATH=$(FEELINGS) $(TINYGO_TINYGO) build -ldflags='font.o $(ALL_ASM_OBJS)' -cflags='-g' -target $(NAME)_qemu.json -o $(NAME) .

$(NAME).hardware: *.go font.o $(ALL_ASM_OBJS)
	GOPATH=$(FEELINGS) $(TINYGO_TINYGO) build  -ldflags='font.o $(ALL_ASM_OBJS)' -target $(NAME).json -o $(NAME).hardware .

clean:
	rm $(NAME) $(NAME).hardware kernel8.img kernel8.img.hardware *.o >/dev/null 2>/dev/null || true

kernel8.img: $(NAME)
	$(TINYGO_OBJCOPY) -O binary $(NAME) kernel8.img

kernel8.img.hardware: $(NAME).hardware
	$(TINYGO_OBJCOPY) -O binary $(NAME).hardware kernel8.img.hardware

run:
	$(TOOLS)/bin/qemu-system-aarch64 -M raspi3 -kernel $(NAME) \
		-serial null -serial chardev:char0 \
 		-d unimp,cpu_reset,guest_errors \
 		-semihosting -semihosting-config enable=on,target=native \
 		--chardev pty,path=/dev/ttys009,mux=on,id=char0 \
		-drive file=$(IMAGEFILE),if=sd,format=raw
# 		 -serial mon:stdio \

debug:
	$(TOOLS)/bin/qemu-system-aarch64 -M raspi3 -S -s \
		-kernel $(NAME) \
		-semihosting -semihosting-config enable=on,target=native \
		-serial null -serial chardev:char0 \
		-d unimp,cpu_reset,guest_errors \
		-drive file=$(IMAGEFILE),if=sd,format=raw \
		--chardev pty,path=/dev/ttys009,mux=on,id=char0