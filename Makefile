BINARY := thumbr
ARGS ?= obsidian/Main
ARCHIVE_NAME := thumbr-$(shell date +%Y%m%d%H%M%S).tar.gz
VERSION ?= $(shell git describe --tags --dirty --always 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)
GOCACHE := $(CURDIR)/.gocache
GO_TEST_FLAGS ?= -count=1
TEST_PKGS ?= ./...

.PHONY: build run fmt lint test tidy clean check archive

build:
	GOCACHE="$(GOCACHE)" go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/thumbr

run: build
	./$(BINARY) $(ARGS)

fmt:
	gofmt -w cmd internal

lint:
	GOCACHE="$(GOCACHE)" go vet ./...

test:
	GOCACHE="$(GOCACHE)" go test $(GO_TEST_FLAGS) $(TEST_PKGS)

tidy:
	go mod tidy

check: lint test

clean:
	rm -f $(BINARY)

archive: clean
	@echo "Creating archive: $(ARCHIVE_NAME)"
	@tar -czf $(ARCHIVE_NAME) \
		--exclude='$(BINARY)' \
		--exclude='.gocache' \
		--exclude='obsidian' \
		--exclude='*.tar.gz' \
		--exclude='DOCKER.md' \
		--exclude='Dockerfile*' \
		.
