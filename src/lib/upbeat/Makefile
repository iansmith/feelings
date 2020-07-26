all: buddy_pool.go buddy_fixed_dl.go buddy_dl.go

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
DBL_LINKED_SRC=$(FEELINGS)/src/gen/doubly_linked.go
POOL_SRC=$(FEELINGS)/src/gen/managed_pool.go
FIXED_DL_SRC=$(FEELINGS)/src/gen/fixed_dl.go
BUDDY_FIXED_DL=buddy_fixed_dl.go
BUDDY_DL=buddy_dl.go
BUDDY_POOL=buddy_pool.go

$(BUDDY_DL): $(DBL_LINKED_SRC) buddy.go
	GOPATH=$(FEELINGS) $(GENNY) -in $(DBL_LINKED_SRC) -out $(BUDDY_DL) -pkg upbeat gen "Generic=Buddy"

$(BUDDY_POOL): $(POOL_SRC) buddy.go
	GOPATH=$(FEELINGS) $(GENNY) -in $(POOL_SRC) -out $(BUDDY_POOL) -pkg upbeat gen "Generic=Buddy"

$(BUDDY_FIXED_DL): $(FIXED_DL_SRC) buddy.go
	GOPATH=$(FEELINGS) $(GENNY) -in $(FIXED_DL_SRC) -out $(BUDDY_FIXED_DL) -pkg upbeat gen "Generic=Buddy"
