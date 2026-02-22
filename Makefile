BINARY  := qmdf
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS  := -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: build test lint clean install cross

## build: compile the binary for the current platform
build:
	go build $(LDFLAGS) -o $(BINARY) .

## test: run all tests
test:
	go test ./...

## test-v: run tests with verbose output
test-v:
	go test -v ./...

## lint: run go vet (no extra tools required)
lint:
	go vet ./...

## tidy: tidy and verify go.sum
tidy:
	go mod tidy
	go mod verify

## install: install to GOPATH/bin
install:
	go install $(LDFLAGS) .

## clean: remove build artefacts
clean:
	rm -f $(BINARY)
	rm -f dist/

## cross: cross-compile for common platforms
cross:
	GOOS=linux   GOARCH=amd64  go build $(LDFLAGS) -o dist/$(BINARY)-linux-amd64 .
	GOOS=linux   GOARCH=arm64  go build $(LDFLAGS) -o dist/$(BINARY)-linux-arm64 .
	GOOS=darwin  GOARCH=amd64  go build $(LDFLAGS) -o dist/$(BINARY)-darwin-amd64 .
	GOOS=darwin  GOARCH=arm64  go build $(LDFLAGS) -o dist/$(BINARY)-darwin-arm64 .
	GOOS=windows GOARCH=amd64  go build $(LDFLAGS) -o dist/$(BINARY)-windows-amd64.exe .

## help: show this help
help:
	@grep -E '^## ' Makefile | sed 's/## /  /'
