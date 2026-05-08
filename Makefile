.PHONY: all build build-asm build-cuda build-opencl test bench lint fmt tidy clean

MODULE := $(shell grep '^module' go.mod | awk '{print $$2}')
BINDIR := bin
BINARY := fairchain-miner
GO     ?= go

# Architecture and feature detection for SHA-NI
HAS_SHA_NI := $(shell grep -q sha_ni /proc/cpuinfo 2>/dev/null && echo "sha_ni" || echo "")

GOFLAGS := -v
ifneq ($(HAS_SHA_NI),)
GOFLAGS += -tags $(HAS_SHA_NI)
endif

# Ensure Go assembler recognizes SHA-NI instructions on AMD64
ifeq ($(GOARCH),amd64)
GOFLAGS += -gcflags="-m -l -N" -asmflags="-trimpath=$(shell pwd) -f -s -spectre=all -go-version=$(shell $(GO) version | awk '{print $$3}') -shared -compress-debug-info=false -go-build-id="
endif


# --- Default target ---
all: build

# --- Build targets ---
build:
	mkdir -p $(BINDIR)
	rm -f pkg/algorithm/prefetch_amd64.go pkg/algorithm/prefetch_amd64.s
	$(GO) build $(GOFLAGS) -o $(BINDIR)/$(BINARY) ./cmd/fairchain-miner

build-tui:
	$(GO) build $(GOFLAGS) -o $(BINDIR)/$(BINARY)-tui ./cmd/fairchain-miner

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
	rm -f pkg/algorithm/prefetch_amd64.go pkg/algorithm/prefetch_amd64.s
	$(GO) clean ./...
