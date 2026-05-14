# Makefile for quickspaces-execution-truenas

.PHONY: all test lint fmt check

all: fmt lint test

test:
	go test ./...

lint:
	go vet ./...

fmt:
	gofmt -w .

check: fmt lint test
