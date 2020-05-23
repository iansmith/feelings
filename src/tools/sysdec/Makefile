all: rpi3.sysdec.go
FEELINGS=/Users/iansmith/feelings

# on macs gsed is the name of the gnu sed installed by brew, you might need it in
# some cases because gnu sed has more features than the berkeley sed shipped with OSX
SED=sed
PWD=$(shell pwd)
LEAVE=-l

## this assumes that go install puts things somewhere that your PATH will find it
sysdec: generate.go intermediate.go structure.go template.go unmarshal_help.go useropts.go cmd/sysdec/*.go
	rm -f tmp/sysdec_out.go tmp/int_main.go
	rm -f *.sysdec.go
	GO111MODULE=off GOPATH=$(FEELINGS) go install ./cmd/sysdec

rpi3.sysdec.go: sysdec
	GO111MODULE=off GOPATH=$(FEELINGS) $(FEELINGS)/bin/sysdec -o $(PWD)/tmp/sysdec_out.go $(LEAVE) -t feelings_rpi3 -p machine -b rpi3 sys/rpi3.go
	cat tmp/sysdec_out.go |  $(SED) '/^[[:blank:]]*$$/d' | $(SED) 's/^xxxblankxxx//g' | gofmt > rpi3.sysdec.go
	rm tmp/sysdec_out.go
