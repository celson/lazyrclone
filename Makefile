BINARY_NAME=lazyrclone
BUILD_DIR=bin
VERSION=0.1.0
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-s -w -X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.Date=$(DATE)"

.PHONY: all build test clean run demo install

all: test build

build:
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/lazyrclone

run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

demo: build
	./$(BUILD_DIR)/$(BINARY_NAME) --demo

test:
	go test -v ./...

install:
	go install $(LDFLAGS) ./cmd/lazyrclone

clean:
	rm -rf $(BUILD_DIR)
