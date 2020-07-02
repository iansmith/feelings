all: clean joy joy.hardware

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

EXCEPTIONASM=exception
KERNELASM=kernel
FONTDIR=$(FEELINGS)/src/lib/upbeat
TINYGO_TINYGO=$(TINYGO)/build/tinygo
TINYGO_LLD=$(LLVM)/bin/ld.lld
TINYGO_CLANG=$(LLVM)/bin/clang

NAME=joy
TINYGO_CLANG=$(LLVM)/bin/clang
TARGET=aarch64-none-elf
GEN=../../../tools/genfuncptr/cmd/genfuncptr/genfuncptr
FUNCPTRS=funcptrs

font.psf: $(FONTDIR)/font.psf
	cp $(FONTDIR)/font.psf .

font.o: font.psf
	$(TINYGO_LLD) -m aarch64elf -r -b binary -o font.o font.psf

$(FUNCPTRS).o: $(FUNCPTRS).S
	$(TINYGO_CLANG) -target $(TARGET) -c -o $(FUNCPTRS).o $(FUNCPTRS).S

$(FUNCPTRS).S: funcnames.list
	$(GEN) funcnames.list funcptrs.S

$(EXCEPTIONASM).o: ../../$(EXCEPTIONASM).S
	$(TINYGO_CLANG) -target $(TARGET) -c -o $(EXCEPTIONASM).o ../../$(EXCEPTIONASM).S

$(KERNELASM).o: ../../$(KERNELASM).S
	$(TINYGO_CLANG) -target $(TARGET) -c -o $(KERNELASM).o ../../$(KERNELASM).S

OBJS=font.o $(EXCEPTIONASM).o $(FUNCPTRS).o $(KERNELASM).o

$(NAME): *.go $(EXCEPTIONASM).o font.o $(FUNCPTRS).o $(KERNELASM).o
	GOPATH=$(FEELINGS) $(TINYGO_TINYGO) build -opt 1 -ldflags='$(OBJS)' -target $(NAME)_qemu.json -o $(NAME) .

$(NAME).hardware: *.go $(EXCEPTIONASM).o font.o $(FUNCPTRS).o
	GOPATH=$(FEELINGS) $(TINYGO_TINYGO) build -ldflags='$(OBJS)' -target $(NAME).json -o $(NAME).hardware .

clean:
	rm kernel8.img $(NAME) $(NAME).hardware *.o >/dev/null 2>/dev/null || true

#run:
#	qemu-system-aarch64 -M raspi3 -kernel $(NAME) -serial null -serial chardev:char0 \
# 		-d unimp,cpu_reset,guest_errors -semihosting -semihosting-config enable=on,target=native \
# 		--chardev pty,path=/dev/ttys009,mux=on,id=char0
#
#debug:
#	qemu-system-aarch64 \
#		-M raspi3 -S -s \
#		-kernel $(NAME) \
#		-semihosting -semihosting-config enable=on,target=native \
#		-serial null -serial stdio
