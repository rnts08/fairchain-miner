# Fairchain Miner User Guide

## Installation
1. Download the latest binary from the [Releases](https://github.com/rnts08/fairchain-miner/releases) page.
2. Linux: `chmod +x fairchain-miner-linux-amd64`

## Usage (Standard)
```bash
./fairchain-miner -stratum stratum+tcp://pool_url:port -user wallet_address.worker
```

## Usage (Mining OS / SRBMiner Drop-in)
This miner supports SRBMiner-style flags for compatibility with HiveOS and RaveOS:
```bash
./fairchain-miner -a sha256mem -o pool_url:port -u wallet_address.worker -p x
```

## JSON API
The miner exposes a local API (default port 4040) for monitoring:
`http://localhost:4040/stats`

## Performance Tuning

### Hugepages (Linux)
Significant performance boost for the 64MB scratchpad:
```bash
sudo sysctl -w vm.nr_hugepages=1024
```

### CPU Affinity
For high-core count CPUs, the miner performs better if workers are pinned to physical cores to avoid L3 cache contention. This can be toggled in the TUI (Settings) or configured via persistent config.

### Power Limit
Use `-power-limit 80` to keep your system cool during extended mining sessions.

## Troubleshooting
- **mbind failed**: Usually means NUMA is not supported or restricted (common in Docker). The miner will fall back to standard memory allocation.
- **Low Hashrate**: Check if your CPU supports SHA-NI extensions. Check if Hugepages are enabled.