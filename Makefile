SHELL := /usr/bin/env bash
GOBIN := $(shell go env GOPATH)/bin
GOLANGCI_LINT := $(GOBIN)/golangci-lint

.PHONY: all help tidy fmt check-format vet test lint lint-fix ci

all: help

help:
	@echo "Available targets:"
	@echo "  make tidy         # ensure go.mod and go.sum are up to date"
	@echo "  make fmt          # format Go source files"
	@echo "  make check-format # verify Go files are formatted"
	@echo "  make vet          # run go vet"
	@echo "  make test         # run Go unit tests"
	@echo "  make lint         # run golangci-lint"
	@echo "  make lint-fix     # apply golangci-lint automatic fixes"
	@echo "  make ci           # run the full CI validation"

tidy:
	go mod tidy

fmt:
	gofmt -w .

check-format:
	unformatted=$(gofmt -l .) && if [[ -n "$${unformatted}" ]]; then \
	  echo "The following Go files are not formatted:"; \
	  echo "$${unformatted}"; \
	  echo; \
	  echo "Run make fmt to fix formatting."; \
	  exit 1; \
	fi

vet:
	go vet ./...

test:
	go test ./...

install-tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

lint: install-tools
	$(GOLANGCI_LINT) run ./...

lint-fix: install-tools
	$(GOLANGCI_LINT) run --fix ./...

ci: tidy check-format vet test lint
