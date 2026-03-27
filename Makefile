BINARY_NAME=bp
BUILD_DIR=bin
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X github.com/bitpanda-labs/bitpanda-cli/internal/cli.Version=$(VERSION)"

.PHONY: build install test lint clean

build:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/bp

install:
	go install $(LDFLAGS) ./cmd/bp

test:
	go test ./... -v

lint:
	golangci-lint run ./...

clean:
	rm -rf $(BUILD_DIR)
