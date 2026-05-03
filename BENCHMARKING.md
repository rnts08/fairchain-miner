# Fairchain Miner Benchmarking Guide

This document describes how to benchmark the Fairchain miner's various optimization levels and hardware-specific codepaths. These benchmarks allow developers and users to verify performance gains on different architectures (Intel, AMD, ARM) and configurations (Hugepages, NUMA).

## 1. Running Automated Benchmarks

The miner uses the standard Go testing tool for fine-grained benchmarks.

### Core Hashing Performance
To benchmark the core `sha256mem` algorithm (including mix passes and ARX fill):
```bash
go test ./pkg/algorithm/... -bench PoWHash -benchmem
```

### Hugepage Impact
Compare performance with and without hugepages:
```bash
go test ./pkg/algorithm/... -bench PoWHash -benchmem
```
*Note: `BenchmarkPoWHash` uses hugepages by default if available, while `BenchmarkPoWHashRegular` uses standard OS allocation.*

### Assembly Kernels (AMD64)
Benchmark specific assembly optimizations:
```bash
# ARX Fill (AVX2 vs AVX-512 vs Generic)
go test ./pkg/algorithm/... -bench ARXFill -benchmem

# SHA-NI (Single-buffer vs Dual-buffer interleaved)
go test ./pkg/algorithm/... -bench SHANI -benchmem
```

## 2. Full Miner Benchmarking Mode

To measure the overall hashrate in a realistic scenario (multiple workers, full mining loop):
```bash
# Benchmark with 8 workers
./fairchain-miner -benchmark -workers 8 -duration 30s
```

## 3. Configuration Profiles

### High Performance (Server/Rig)
Ensure hugepages are enabled in the kernel:
```bash
sudo sysctl -w vm.nr_hugepages=1024
```
Run with NUMA affinity and hugepages:
```bash
./fairchain-miner -workers $(nproc) -power-limit 100
```

### Efficiency (Laptop/Dev)
```bash
./fairchain-miner -workers 4 -power-limit 50
```

## 4. Hardware Results (Reference)

| Architecture | CPU | Optimization | Hashrate (per core) |
| :--- | :--- | :--- | :--- |
| Intel Alder Lake | i7-1265U | Pure Go | ~17 H/s |
| Intel Alder Lake | i7-1265U | SHA-NI + AVX2 | ~28 H/s |
| Intel Alder Lake | i7-1265U | SHA-NI Dual + AVX-512 + Hugepages | ~42 H/s |
| AMD Zen 3 | - | - | TBD |
| ARM64 (Apple M3) | - | - | TBD |

## 5. Performance Profiling

To generate a CPU profile for further optimization:
```bash
go test ./pkg/algorithm/... -bench PoWHash -cpuprofile cpu.prof
go tool pprof -http=:8080 cpu.prof
```
