.PHONY: all build build-asm build-cuda build-opencl test bench lint fmt tidy clean

MODULE := $(shell grep '^module' go.mod | awk '{print $$2}')
BINDIR := bin
BINARY := fairchain-miner
GO     ?= go

# --- Default target ---
all: build

# --- Build targets ---
build:
	mkdir -p $(BINDIR)
	$(GO) build -v -o $(BINDIR)/$(BINARY) ./cmd/fairchain-miner

# Optimized build for specific architectures
build-amd64:
	GOOS=linux GOARCH=amd64 $(GO) build -v -o $(BINDIR)/$(BINARY)-linux-amd64 ./cmd/fairchain-miner

build-arm64:
	GOOS=linux GOARCH=arm64 $(GO) build -v -o $(BINDIR)/$(BINARY)-linux-arm64 ./cmd/fairchain-miner

build-cuda:
	$(GO) build -v -tags cuda -o $(BINDIR)/$(BINARY)-cuda ./cmd/fairchain-miner

build-opencl:
	$(GO) build -v -tags opencl -o $(BINDIR)/$(BINARY)-opencl ./cmd/fairchain-miner

# --- Test / Bench / Lint ---
test:
	$(GO) test ./... -v -count=1

test-short:
	$(GO) test ./... -count=1

bench:
	$(GO) test ./pkg/algorithm/ -bench=. -benchmem -count=3

bench-all:
	$(GO) test ./... -bench=. -benchmem

lint:
	$(GO) vet ./...

fmt:
	gofmt -w .

tidy:
	$(GO) mod tidy

# --- Run targets ---
run-benchmark:
	$(BINDIR)/$(BINARY) --benchmark --workers 0 --duration 30s

run-regtest:
	$(BINDIR)/$(BINARY) --rpc http://127.0.0.1:19445

run-testnet:
	$(BINDIR)/$(BINARY) --rpc http://127.0.0.1:19445

# --- Clean ---
clean:
	rm -rf $(BINDIR)
	$(GO) clean ./...
