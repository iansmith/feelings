all: clean test

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

GENNY=$(FEELINGS)/bin/genny
DBL_LINKED_SRC=doubly_linked.go
POOL_SRC=fixed_pool.go
FIXED_DL_SRC=fixed_dl.go
STRINGISH_FIXED_DL=cmd/endurance/stringish_fixed_dl.go
STRINGISH_DL=cmd/endurance/stringish_dl.go
STRINGISH_POOL=cmd/endurance/stringish_pool.go

$(STRINGISH_DL): $(DBL_LINKED_SRC)
	GOPATH=$(FEELINGS) $(GENNY) -in $(DBL_LINKED_SRC) -out $(STRINGISH_DL) -pkg main gen "Generic=Stringish"

$(STRINGISH_POOL): $(POOL_SRC)
	GOPATH=$(FEELINGS) $(GENNY) -in $(POOL_SRC) -out $(STRINGISH_POOL) -pkg main gen "Generic=Stringish"

$(STRINGISH_FIXED_DL): $(FIXED_DL_SRC)
	GOPATH=$(FEELINGS) $(GENNY) -in $(FIXED_DL_SRC) -out $(STRINGISH_FIXED_DL) -pkg main gen "Generic=Stringish"

test: $(STRINGISH_DL) $(STRINGISH_POOL) $(STRINGISH_FIXED_DL)
	GOPATH=$(FEELINGS) $(TINYGO)/build/tinygo test   .

clean:
	rm -f $(STRINGISH_DL) $(STRINGISH_POOL) $(STRINGISH_FIXED_DL)
