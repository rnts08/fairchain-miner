# Installation Guide

## Binary Releases

Pre-built binaries are available for all major platforms from the [Releases Page](https://github.com/rnts08/fairchain-miner/releases).

### Linux (amd64 / arm64)
```bash
wget https://github.com/rnts08/fairchain-miner/releases/latest/download/fairchain-miner-linux-amd64
chmod +x fairchain-miner-linux-amd64
sudo mv fairchain-miner-linux-amd64 /usr/local/bin/fairchain-miner
```

### macOS (Intel / Apple Silicon)
```bash
curl -L https://github.com/rnts08/fairchain-miner/releases/latest/download/fairchain-miner-darwin-arm64 -o fairchain-miner
chmod +x fairchain-miner
xattr -d com.apple.quarantine fairchain-miner 2>/dev/null || true
sudo mv fairchain-miner /usr/local/bin/
```

### Windows
Download the latest `.exe` binary from the releases page and run from command prompt.

---

## Docker

```bash
# Pull latest image
docker pull ghcr.io/rnts08/fairchain-miner:latest

# Run miner
docker run ghcr.io/rnts08/fairchain-miner:latest --rpc http://your-node:19445
```

---

## Build from Source

### Prerequisites
- Go 1.25+
- Git

```bash
git clone https://github.com/rnts08/fairchain-miner.git
cd fairchain-miner
make build
```

### Build Targets
```bash
# Pure Go build (works everywhere)
make build

# With assembly optimizations
make build-asm

# With CUDA support
make build-cuda

# With OpenCL support
make build-opencl
```

---

## Post Installation

### Verify Installation
```bash
fairchain-miner --version
```

### First Run
```bash
fairchain-miner --benchmark --workers 1
```

### Configuration
See `README.md` for full command line options and usage examples.