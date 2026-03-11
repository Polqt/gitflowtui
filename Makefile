GOLANGCI_LINT ?= $(shell go env GOPATH)/bin/golangci-lint

.PHONY: lint lint-fix

lint: ## Run golangci-lint
	$(GOLANGCI_LINT) run --timeout=5m

lint-fix: ## Run golangci-lint with autofix
	$(GOLANGCI_LINT) run --fix --timeout=5m
