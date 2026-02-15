.PHONY: help fmt test lint lint-basic cover check tools

GO ?= go
GOLANGCI_LINT ?= golangci-lint

.DEFAULT_GOAL := help

help:
	@echo "make fmt         - go fmt (gofmt) for all packages"
	@echo "make test        - run unit tests"
	@echo "make lint        - run golangci-lint (requires installation)"
	@echo "make lint-basic  - run go vet"
	@echo "make cover       - run tests with coverage report"
	@echo "make check       - fmt + lint + test"
	@echo "make tools       - install golangci-lint (network required)"

fmt:
	$(GO) fmt ./...

test:
	$(GO) test ./...

lint-basic:
	$(GO) vet ./...

lint:
	@command -v $(GOLANGCI_LINT) >/dev/null 2>&1 || ( \
		echo "$(GOLANGCI_LINT) is not installed. Run: make tools" >&2; \
		exit 1; \
	)
	$(GOLANGCI_LINT) run ./...

cover:
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out

check: fmt lint test

tools:
	$(GO) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
