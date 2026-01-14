.DEFAULT_GOAL := build

BINARY ?= bin/linear
PKG ?= ./...
TOOLS_DIR ?= $(CURDIR)/.tools
GOLANGCI_LINT := $(TOOLS_DIR)/golangci-lint

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT := $(shell git rev-parse --short=12 HEAD 2>/dev/null || echo "")
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -X github.com/duailibe/linear-cli/internal/cli.version=$(VERSION) -X github.com/duailibe/linear-cli/internal/cli.commit=$(COMMIT) -X github.com/duailibe/linear-cli/internal/cli.date=$(DATE)

.PHONY: build test fmt tidy install tools lint

build:
	@mkdir -p $(dir $(BINARY))
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/linear

test:
	go test $(PKG)

fmt:
	go fmt $(PKG)

tidy:
	go mod tidy

install:
	go install ./cmd/linear

tools:
	@mkdir -p $(TOOLS_DIR)
	@GOBIN=$(TOOLS_DIR) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.1.6

lint: tools
	@$(GOLANGCI_LINT) run
