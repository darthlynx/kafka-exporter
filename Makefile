# Makefile for a Go project
# Usage: make <target>
# Override with: make BINARY=myname GO=go1.24

GO ?= go
BINARY ?= kafka-exporter
BUILD_DIR ?= bin
CMD_DIR ?= ./cmd/$(BINARY)

VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo 'v0.0.0')
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo 'unknown')
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

PLATFORMS ?= linux/amd64 darwin/amd64 linux/arm64 windows/amd64

.DEFAULT_GOAL := help

help:
	@echo "Makefile targets:"
	@echo "  make build         Build $(BINARY) for the host OS"
	@echo "  make build-all     Cross-compile for common platforms"
	@echo "  make build-linux   Build linux/amd64 binary"
	@echo "  make test          Run unit tests"
	@echo "  make fmt           Run go fmt"
	@echo "  make vet           Run go vet"
	@echo "  make deps          Download module dependencies"
	@echo "  make tidy          Run go mod tidy"
	@echo "  make install       go install the binary"
	@echo "  make lint          Run golangci-lint (if installed)"
	@echo "  make clean         Remove build artifacts"

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

tidy:
	$(GO) mod tidy

deps:
	$(GO) mod download

build: deps
	@mkdir -p $(BUILD_DIR)
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY) $(CMD_DIR)

build-linux:
	@$(MAKE) build GOOS=linux GOARCH=amd64

build-all:
	@mkdir -p $(BUILD_DIR)
	@set -e; \
	for p in $(PLATFORMS); do \
		os=$$(echo $$p | cut -d/ -f1); arch=$$(echo $$p | cut -d/ -f2); \
		out=$(BUILD_DIR)/$(BINARY)-$$os-$$arch; \
		if [ "$$os" = "windows" ]; then out=$$out.exe; fi; \
		echo "building $$out ..."; \
		GOOS=$$os GOARCH=$$arch $(GO) build -ldflags "$(LDFLAGS)" -o $$out $(CMD_DIR); \
	done

test:
	$(GO) test ./...

run:
	$(GO) run $(CMD_DIR)

install: deps
	$(GO) install -ldflags "$(LDFLAGS)" $(CMD_DIR)

lint:
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "golangci-lint not found; install from https://golangci-lint.run/"; exit 1; \
	fi
	golangci-lint run

clean:
	@rm -rf $(BUILD_DIR)
	$(GO) clean

.PHONY: help fmt vet tidy deps build build-linux build-all test run install lint clean
