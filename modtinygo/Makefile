all: install_rpi3

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

install_rpi3:
	cp -R ./rpi3-files/* $(TINYGO)
	cd ../src/tools/sysdec && make && cp rpi3.sysdec.go rpi3_qemu.sysdec.go $(TINYGO)/src/machine
