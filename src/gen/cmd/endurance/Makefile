all: clean endurance

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

NAME=endurance
TINYGO_TINYGO=$(TINYGO)/build/tinygo
TINYGO_CLANG=$(LLVM)/bin/clang

BOOTASM=rpi3_baremetal_boot
BOOTLIB=$(FEELINGS)/src/boot/lib
TARGET=aarch64-none-elf

$(BOOTASM).o: $(BOOTLIB)/$(BOOTASM).S
	$(TINYGO_CLANG) -target $(TARGET) -c -o $(BOOTASM).o $(BOOTLIB)/$(BOOTASM).S


$(NAME): *.go $(BOOTASM).o
	GOPATH=$(FEELINGS) $(TINYGO_TINYGO) build -opt 1 -cflags=-g -ldflags='$(BOOTASM).o' -target rpi3_qemu -o $(NAME) .


$(NAME).hardware: *.go $(BOOTASM).o
	GOPATH=$(FEELINGS) $(TINYGO_TINYGO) build -opt 1 -ldflags='$(BOOTASM).o' -target rpi3 -o $(NAME).hardware .

clean:
	rm $(NAME) $(NAME).hardware kernel8.img *.o >/dev/null 2>/dev/null || true

test:
	GOPATH=$(FEELINGS) $(TINYGO_TINYGO) test .

run:
	$(TOOLS)/bin/qemu-system-aarch64 -M raspi3 -kernel $(NAME) \
	-serial null -serial stdio \
	-chardev pty,id=char1 \
	-semihosting -semihosting-config enable=on,target=native

debug:
	$(TOOLS)/bin/qemu-system-aarch64  \
		-M raspi3 -S -s \
		-kernel $(NAME) \
		-semihosting -semihosting-config enable=on,target=native \
		-serial null -serial stdio
