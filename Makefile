# --- Variables ---
.DEFAULT_GOAL := help

BINARY_NAME := thumbr
BUILD_DIR   := bin
BINARY      := $(BUILD_DIR)/$(BINARY_NAME)
ARGS        ?= samples/notes

# Versioning
VERSION     ?= $(shell git describe --tags --dirty --always 2>/dev/null || echo dev)
BUILD_DATE  := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

# LDFLAGS: Linker flags (inject variables)
LDFLAGS     := -X main.version=$(VERSION) -X main.buildDate=$(BUILD_DATE)

# Go Configuration
GO            := go
GOFLAGS       ?=
GO_BUILD_FLAGS:= -trimpath $(GOFLAGS)
GOCACHE       ?= $(CURDIR)/.gocache
GO_TEST_FLAGS ?= -count=1
TEST_PKGS     ?= ./...

# Archive
ARCHIVE_NAME := $(BINARY_NAME)-$(VERSION).tar.gz

# --- Main Targets ---

.PHONY: all build run fmt vet lint test race cover tidy deps clean clean-cache check archive help

all: check build

## build: Build the binary to the bin/ directory
build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	GOCACHE="$(GOCACHE)" $(GO) build $(GO_BUILD_FLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/thumbr

## run: Build and run the binary (pass args with ARGS="...")
run: build
	@echo "Running..."
	@./$(BINARY) $(ARGS)

## fmt: Format all go files
fmt:
	@echo "Formatting..."
	@gofmt -w .

## vet: Run go vet
vet:
	@echo "Vet..."
	GOCACHE="$(GOCACHE)" $(GO) vet ./...

## lint: Alias for vet (backcompat)
lint: vet

## test: Run unit tests
test:
	@echo "Testing..."
	GOCACHE="$(GOCACHE)" $(GO) test $(GO_TEST_FLAGS) $(TEST_PKGS)

## race: Run unit tests with race detector
race:
	@echo "Testing with race detector..."
	GOCACHE="$(GOCACHE)" $(GO) test -race $(GO_TEST_FLAGS) $(TEST_PKGS)

## cover: Run tests with coverage profile (cover.out)
cover:
	@echo "Testing with coverage..."
	GOCACHE="$(GOCACHE)" $(GO) test $(GO_TEST_FLAGS) -coverprofile=cover.out $(TEST_PKGS)

## tidy: Tidy go.mod dependencies
tidy:
	@echo "Tidying modules..."
	$(GO) mod tidy

## deps: Download Go module dependencies
deps:
	@echo "Downloading modules..."
	$(GO) mod download

## check: Run formatting, linting, and testing
check: fmt vet test

## clean: Remove binary and build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f $(ARCHIVE_NAME) cover.out

## clean-cache: Remove local build cache (when GOCACHE is under repo)
clean-cache:
	@echo "Cleaning cache..."
	@rm -rf "$(GOCACHE)"

## archive: Create a source archive using git
archive:
	@echo "Creating archive: $(ARCHIVE_NAME)"
	@git archive --format=tar.gz --prefix=$(BINARY_NAME)-$(VERSION)/ -o $(ARCHIVE_NAME) HEAD

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@awk 'sub(/^## /,"") { split($$0, A, ":"); printf "  %-15s %s\n", A[1], A[2] }' $(MAKEFILE_LIST)
