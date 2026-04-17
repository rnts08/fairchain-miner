# Fairchain Miner

VERSION := 1.0.0
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")

GO_BUILD := go build -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT)"

all: cli tui

cli:
	$(GO_BUILD) -o fairchain-miner ./cmd

tui:
	$(GO_BUILD) -tags tui -o fairchain-miner-tui ./cmd

debug:
	$(GO_BUILD) -gcflags "all=-N -l" -o fairchain-miner ./cmd

debug-tui:
	$(GO_BUILD) -tags tui -gcflags "all=-N -l" -o fairchain-miner-tui ./cmd

clean:
	rm -f fairchain-miner fairchain-miner-tui

install: all
	cp fairchain-miner fairchain-miner-tui /usr/local/bin/

test:
	go test ./pkg/miner

.PHONY: all cli tui debug debug-tui clean install test
