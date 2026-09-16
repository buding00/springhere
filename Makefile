GO ?= go
BIN ?= bin/springhere

.PHONY: help build test vet fmt run-help

help:
	@echo "make build    编译 bin/springhere"
	@echo "make test     go test ./..."
	@echo "make vet      go vet ./..."
	@echo "make fmt      gofmt -w"

build:
	mkdir -p bin
	$(GO) build -o $(BIN) ./cmd/springhere

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

fmt:
	gofmt -w cmd internal

run-help: build
	./$(BIN) --help
	./$(BIN) new --help
