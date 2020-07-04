all: clean sdcard sdcard.hardware

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

NAME=sdcard
TINYGO_CLANG=$(LLVM)/bin/clang
TINYGO_OBJCOPY=$(LLVM)/bin/llvm-objcopy
TINYGO_LLD=$(LLVM)/bin/ld.lld
TINYGO_TINYGO=$(TINYGO)/build/tinygo

BOOTASM=rpi3_baremetal_boot
BOOTLIB=$(FEELINGS)/src/boot/lib
TARGET=aarch64-none-elf

IMAGEFILE=/Users/iansmith/feelings/samples/emmc/test/image.img

$(BOOTASM).o: $(BOOTLIB)/$(BOOTASM).S
	$(TINYGO_CLANG) -target $(TARGET) -c -o $(BOOTASM).o $(BOOTLIB)/$(BOOTASM).S


$(NAME): *.go $(BOOTASM).o
	GOPATH=$(FEELINGS) $(TINYGO_TINYGO) build -opt 1 -ldflags='$(BOOTASM).o' -target rpi3_qemu -o $(NAME) .

$(NAME).hardware: *.go $(BOOTASM).o
	GOPATH=$(FEELINGS) $(TINYGO_TINYGO) build -opt 1 -ldflags='$(BOOTASM).o' -target rpi3 -o $(NAME).hardware .

clean:
	rm $(NAME) $(NAME).hardware kernel8.img *.o >/dev/null 2>/dev/null || true

kernel8.img: sdcard
	$(TINYGO_OBJCOPY) -O binary $(NAME) kernel8.img

run:
	$(TOOLS)/bin/qemu-system-aarch64 -M raspi3 -kernel sdcard \
	-serial null -serial stdio \
	-chardev pty,id=char1 \
	 --trace events=/tmp/events \
	-drive file=$(IMAGEFILE),if=sd,format=raw

#-drive file=/Users/iansmith/outfile.bin,if=sd,format=raw,drive=sd0

debug:
	$(TOOLS)/bin/qemu-system-aarch64  \
		-M raspi3 -S -s \
		-kernel sdcard \
		-semihosting -semihosting-config enable=on,target=native \
		-drive file=$(IMAGEFILE),if=sd,format=raw \
		-serial null -serial stdio
