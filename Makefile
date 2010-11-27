include $(GOROOT)/src/Make.inc

TARG=gommap
GOFMT=gofmt -spaces=true -tabindent=false -tabwidth=4

GOFILES=\
	gommap.go\
	consts.go\

include $(GOROOT)/src/Make.pkg

# consts.go isn't cleaned up so that goinstall works.
CLEANFILES+=\
	_consts.out\

format:
	${GOFMT} -w gommap.go
	${GOFMT} -w gommap_test.go

%.go: %.c
	$(HOST_CC) -Wall -pedantic $< -o _$*.out
	./_$*.out > $@
