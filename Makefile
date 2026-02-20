BINARY := vecna
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X github.com/shravan20/vecna/cmd.Version=$(VERSION)"

.PHONY: build run clean install lint test

build:
	go build $(LDFLAGS) -o bin/$(BINARY) .

run:
	go run .

clean:
	rm -rf bin/

install:
	go install $(LDFLAGS) .

lint:
	golangci-lint run

test:
	go test -v ./...
