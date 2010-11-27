include $(GOROOT)/src/Make.inc

TARG=gommap
GOFMT=gofmt -spaces=true -tabindent=false -tabwidth=4

GOFILES=\
	gommap.go\
	_consts.go\

include $(GOROOT)/src/Make.pkg

CLEANFILES+=\
	_consts.go\
	_consts.out

format:
	${GOFMT} -w gommap.go
	${GOFMT} -w gommap_test.go

_%.go: %.c
	$(HOST_CC) -Wall -pedantic $< -o _$*.out
	./_$*.out > $@
