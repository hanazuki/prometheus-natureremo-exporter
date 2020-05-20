all: build man

build:
	go build

man:
	asciidoctor -b manpage man/*.adoc

.PHONY: all build man
