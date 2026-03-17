GO ?= go
BINARY ?= post
MAIN_PACKAGE ?= ./cmd/post
GO_PACKAGES ?= ./...
VERSION_FILE ?= VERSION
VERSION ?= $(shell cat $(VERSION_FILE))
COMMIT ?= $(shell git rev-parse HEAD)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS ?= -s -w \
	-X github.com/mirtle/post-cli/internal/buildinfo.Version=$(VERSION) \
	-X github.com/mirtle/post-cli/internal/buildinfo.Commit=$(COMMIT) \
	-X github.com/mirtle/post-cli/internal/buildinfo.BuildDate=$(BUILD_DATE)
all: rebuild

.PHONY: help build rebuild clean test smoke-local fmt

help:
	@printf '%s\n' \
		'Available targets:' \
		'  make build        Build the CLI binary' \
		'  make rebuild      Remove the old binary and build from scratch' \
		'  make clean        Remove build output' \
		'  make test         Run all Go tests' \
		'  make smoke-local  Run local smoke tests (requires POST_HOST and POST_TOKEN)' \
		'  make fmt          Format Go source files'

build:
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BINARY) $(MAIN_PACKAGE)

rebuild: clean build

clean:
	rm -f $(BINARY)

test:
	$(GO) test $(GO_PACKAGES)

smoke-local:
	./scripts/smoke_local.sh

fmt:
	$(GO) fmt $(GO_PACKAGES)
