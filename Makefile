BINARY  := nodeval
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build build-windows build-all test vet check clean

## build: compile for the current platform
build:
	go build $(LDFLAGS) -o $(BINARY) .

## build-windows: cross-compile for Windows amd64
build-windows:
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY).exe .

## build-all: compile for Linux (amd64, arm64) and Windows (amd64)
build-all:
	GOOS=linux  GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-linux-amd64 .
	GOOS=linux  GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY)-linux-arm64 .
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-windows-amd64.exe .

## test: run all tests
test:
	go test ./... -v

## vet: run static analysis
vet:
	go vet ./...

## check: vet + test (run before committing)
check: vet test

## clean: remove built binaries
clean:
	rm -f $(BINARY) $(BINARY).exe
	rm -rf dist/

## help: list available targets
help:
	@grep -E '^## ' Makefile | sed 's/^## //'
