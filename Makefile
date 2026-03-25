BINARY        := gitflow-tui
MODULE        := github.com/Polqt/gitflowtui
CMD           := ./cmd/gitflow-tui
GOLANGCI_LINT ?= $(shell go env GOPATH)/bin/golangci-lint

VERSION  := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT   := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE     := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS  := -s -w \
            -X main.buildVersion=$(VERSION) \
            -X main.buildCommit=$(COMMIT) \
            -X main.buildDate=$(DATE)

.PHONY: all build install test lint lint-fix clean release snapshot

all: build

## build: compile the binary for the current platform
build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BINARY) $(CMD)

## install: install the binary into GOPATH/bin
install:
	CGO_ENABLED=0 go install -ldflags "$(LDFLAGS)" $(CMD)

## test: run all tests with race detector and coverage
test:
	go test -race -cover ./...

## lint: run golangci-lint
lint:
	$(GOLANGCI_LINT) run --timeout=5m

## lint-fix: run golangci-lint with autofix
lint-fix:
	$(GOLANGCI_LINT) run --fix --timeout=5m

## clean: remove build artifacts
clean:
	rm -f $(BINARY)
	rm -rf dist/

## release: create a tagged GitHub release via goreleaser (requires GITHUB_TOKEN)
release:
	goreleaser release --clean

## snapshot: build all platform binaries locally without publishing
snapshot:
	goreleaser release --snapshot --clean --skip=publish

## help: list available targets
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## //' | column -t -s ':'
