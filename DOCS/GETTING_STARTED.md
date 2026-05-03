# Getting Started with Fairchain Miner

This guide covers how to set up, build, and use the hyper-optimized Fairchain miner for both solo and pool mining.

## 1. Prerequisites

- **Go 1.25+**: Required for building the miner.
- **Linux (Recommended)**: The highest performance optimizations (Hugepages, NUMA) are currently Linux-only.
- **Hardware SHA-NI**: Most modern Intel (Ice Lake+) and AMD (Zen 3+) CPUs support this.
- **Hugepages**: 64MB scratchpad per worker benefits significantly from hugepages.

## 2. Building

The miner supports different build profiles:

```bash
# Standard build (includes AMD64/ARM64 assembly)
make build

# Clean build
make clean && make build
```

The resulting binary will be named `fairchain-miner`.

## 3. Configuration

### System Optimization (Linux)

To enable 2MB Hugepages (drastic performance boost):
```bash
sudo sysctl -w vm.nr_hugepages=1024
```

### Mining Options

| Flag | Description | Default |
| :--- | :--- | :--- |
| `-rpc` | Node RPC URL for solo mining | `http://127.0.0.1:19445` |
| `-stratum` | Stratum pool address | (None) |
| `-user` | Reward address (Solo) or Worker name (Pool) | (None) |
| `-workers` | Number of mining threads | CPU Count |
| `-power-limit`| CPU utilization limit (1-100%) | 100 |

## 4. Mining Modes

### Mode A: Solo Mining (Local Node)
Connect directly to your `fairchaind` node. Ensure the node has RPC enabled.
```bash
./fairchain-miner -rpc http://127.0.0.1:19445 -user YOUR_FAIRCHAIN_ADDRESS
```

### Mode B: Pool Mining (Stratum)
Connect to a public mining pool using the Stratum V1 protocol.
```bash
./fairchain-miner -stratum stratum+tcp://pool.fairchain.org:3333 -user WALLET_ADDRESS.worker1
```

### Mode C: Benchmark
Measure your system's raw hashrate without any network connection.
```bash
./fairchain-miner -benchmark -workers 8 -duration 60s
```

## 5. Performance Tuning

- **Worker Count**: By default, the miner uses all available logical cores. For CPUs with Hyper-Threading/SMT, you may find that using only physical cores (`-workers 8` on a 16-thread CPU) provides better hashrate per worker due to cache contention.
- **NUMA**: On multi-socket servers, the miner automatically detects and pins memory to the local NUMA node of each worker.
- **Hugepages**: Always ensure `vm.nr_hugepages` is set high enough to cover `64MB * workers`.

## 6. Troubleshooting

- **"mbind failed"**: This usually means your kernel doesn't support NUMA or you are running in a restricted container. The miner will fall back to default allocation.
- **"mmap failed (MAP_HUGETLB)"**: Ensure hugepages are allocated in the OS.
- **Stale Shares**: If you get many "job not found" errors on a pool, check your network latency or reduce the number of workers to prevent CPU starvation.
