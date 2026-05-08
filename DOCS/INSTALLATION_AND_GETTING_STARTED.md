# Installation and Getting Started

This guide covers how to set up, build, and use the optimized Fairchain miner.

## 1. Prerequisites

- **Go 1.25+**: Required for building the miner.
- **Linux (Recommended)**: The highest performance optimizations (Hugepages, NUMA) are currently Linux-only.
- **Hardware SHA-NI**: Modern Intel (Ice Lake+) and AMD (Zen 3+) CPUs provide a significant boost via hardware SHA extensions.

## 2. Installation

### Binary Releases
Pre-built binaries for Linux, macOS, and Windows are available on the [Releases Page](https://github.com/rnts08/fairchain-miner/releases).

### Docker
```bash
docker pull ghcr.io/rnts08/fairchain-miner:latest
docker run ghcr.io/rnts08/fairchain-miner:latest --rpc http://your-node:19445
```

### Build from Source
```bash
git clone https://github.com/rnts08/fairchain-miner.git
cd fairchain-miner
make build
```

The build system supports several targets:
- `make build`: Standard build (Pure Go + ASM).
- `make build-cuda`: Enables NVIDIA GPU support.
- `make build-opencl`: Enables OpenCL support.

## 3. Configuration & Optimization

### Hugepages (Linux Only)
Each worker uses a 64MB scratchpad. Enabling 2MB Hugepages reduces TLB misses.
```bash
sudo sysctl -w vm.nr_hugepages=1024
```

### Core Mining Options
| Flag | Description | Default |
| :--- | :--- | :--- |
| `-rpc` | Node RPC URL for solo mining | `http://127.0.0.1:19445` |
| `-stratum` | Stratum pool address | (None) |
| `-user` | Reward address (Solo) or Worker name (Pool) | (None) |
| `-workers` | Number of mining threads | CPU Count |
| `-power-limit`| CPU utilization limit (1-100%) | 100 |

## 4. Mining Modes

### Solo Mining
Connect directly to your local node:
```bash
./fairchain-miner -rpc http://127.0.0.1:19445 -user YOUR_ADDRESS
```

### Pool Mining
Connect to a Stratum V1 compatible pool:
```bash
./fairchain-miner -stratum stratum+tcp://pool.example.com:3333 -user ADDRESS.worker1
```

### Benchmark
Verify raw performance without a network connection:
```bash
./fairchain-miner -benchmark -duration 60s
```

## 5. Performance Tuning

- **Physical Cores**: On CPUs with Hyper-Threading/SMT, you may get a higher hashrate by limiting `-workers` to the number of physical cores.
- **NUMA**: On multi-socket servers, the miner automatically attempts to pin memory to the local NUMA node.
- **Apple Silicon**: You may find better results by setting `-workers` to the number of Performance (P) cores.

## 6. Troubleshooting

- **"mmap failed (MAP_HUGETLB)"**: Ensure you have allocated enough hugepages in the OS via `sysctl`.
- **"mbind failed"**: This indicates the miner couldn't set NUMA affinity. It will fallback to standard allocation.
- **Stale Shares**: If you get many "job not found" errors on a pool, check your network latency or try reducing the `-workers` count if the CPU is over-saturated.

## 7. Post Installation

Verify your installation and version:
```bash
./fairchain-miner --version
```

Run your first benchmark:
```bash
./fairchain-miner --benchmark --workers 1
```

For further details on hardware specifics, see TUNING.md.