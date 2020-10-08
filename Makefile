default: build

build:
	go build ./

install:
	go install

$(V).SILENT:
.PHONY: tests builds
