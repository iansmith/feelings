all: clean release

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

GOCOMP=$(HOSTGO)/bin/go

#
# NOTE: Built with system go compiler, probably should be go1.14. Targeted for the
# NOTE: host system to push a file to the device over serial.
#
release: *.go
	GO111MODULE=off GOPATH=$(FEELINGS) $(GOCOMP) build -o release .

clean:
	rm release *.o >/dev/null 2>/dev/null || true

##
## NOTE: This /dev/ttys004 depends on your system state when you run
## NOTE: antc in another window. Look at the first line of output from
## NOTE: antc to see what to change this to
##
deploy: release
	./release -v 2 -p /dev/ttys005 ../../../../joy/cmd/joy/joy



