include $(GOROOT)/src/Make.inc

TARG=launchpad.net/gommap

GOFILES=\
	gommap.go\
	consts.go\
	mmap_$(GOOS)_$(GOARCH).go\

include $(GOROOT)/src/Make.pkg

# consts.go isn't cleaned up so that goinstall works.
CLEANFILES+=\
	_consts.out\

%.go: %.c
	$(HOST_CC) -Wall -pedantic $< -o _$*.out
	./_$*.out > $@

GOFMT=gofmt
BADFMT=$(shell $(GOFMT) -l $(GOFILES) $(wildcard *_test.go))

gofmt: $(BADFMT)
	@for F in $(BADFMT); do $(GOFMT) -w $$F && echo $$F; done

ifneq ($(BADFMT),)
ifneq ($(MAKECMDGOALS),gofmt)
$(warning WARNING: make gofmt: $(BADFMT))
endif
endif
