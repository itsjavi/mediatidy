default: build

build:
	go build ./cmd/mediatidy

test:
	ls /usr/local/src
	go test --cover ./...

$(V).SILENT:
.PHONY: test build
