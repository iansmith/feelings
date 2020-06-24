all: rpi3.sysdec.go rpi3_qemu.sysdec.go
FEELINGS=/Users/iansmith/feelings


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

# on macs gsed is the name of the gnu sed installed by brew, you might need it in
# some cases because gnu sed has more features than the berkeley sed shipped with OSX
SED=sed
PWD=$(shell pwd)
LEAVE=-l
GOFMT=$(HOSTGO)/bin/gofmt

## this assumes that go install puts things somewhere that your PATH will find it
sysdec: generate.go intermediate.go structure.go template.go unmarshal_help.go useropts.go cmd/sysdec/*.go
	rm -f tmp/sysdec_out.go tmp/int_main.go
	rm -f *.sysdec.go
	GO111MODULE=off GOPATH=$(FEELINGS) $(HOSTGO)/bin/go install ./cmd/sysdec

rpi3.sysdec.go: sysdec sys/rpi3.go sys/bcm*.go
	GO111MODULE=off GOPATH=$(FEELINGS) $(FEELINGS)/bin/sysdec -o $(PWD)/tmp/sysdec_out.go $(LEAVE) -t feelings_rpi3 -p machine -b rpi3 sys/rpi3.go
	cat tmp/sysdec_out.go |  $(SED) '/^[[:blank:]]*$$/d' | $(SED) 's/^xxxblankxxx//g' | $(GOFMT) > rpi3.sysdec.go
	rm tmp/sysdec_out.go

rpi3_qemu.sysdec.go: sysdec sys/rpi3_qemu.go sys/bcm*.go
	GO111MODULE=off GOPATH=$(FEELINGS) $(FEELINGS)/bin/sysdec -o $(PWD)/tmp/sysdec_out.go $(LEAVE) -t feelings_rpi3_qemu -p machine -b rpi3_qemu sys/rpi3_qemu.go
	cat tmp/sysdec_out.go |  $(SED) '/^[[:blank:]]*$$/d' | $(SED) 's/^xxxblankxxx//g' | $(GOFMT) > rpi3_qemu.sysdec.go
	rm tmp/sysdec_out.go

clean:
	GO111MODULE=off GOPATH=$(FEELINGS) go clean ./cmd/sysdec
	rm -f tmp/int_main.go tmp/sysdec_out.go
	rm -f rpi3.sysdec.go rpi3.sysdec.go
