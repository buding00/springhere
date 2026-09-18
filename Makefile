GO ?= go
BIN ?= bin/springhere
# 可选：make build VERSION=0.1.0  与 GitHub Release 的 ldflags 一致
VERSION ?=
LDFLAGS := -s -w
ifneq ($(VERSION),)
LDFLAGS += -X github.com/buding00/springhere/internal/version.Version=$(VERSION)
endif

.PHONY: help build test vet fmt run-help

help:
	@echo "make build    编译 bin/springhere"
	@echo "make test     go test ./..."
	@echo "make vet      go vet ./..."
	@echo "make fmt      gofmt -w"

build:
	mkdir -p bin
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN) ./cmd/springhere

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

fmt:
	gofmt -w cmd internal

run-help: build
	./$(BIN) --help
	./$(BIN) new --help
