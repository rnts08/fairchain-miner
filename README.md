# Fairchain Miner — Hyper-Optimized SHA256-Mem Mining Engine

[![CI Pipeline](https://github.com/rnts08/fairchain-miner/actions/workflows/ci.yml/badge.svg)](https://github.com/rnts08/fairchain-miner/actions/workflows/ci.yml)
[![Code Coverage](https://codecov.io/gh/rnts08/fairchain-miner/branch/main/graph/badge.svg)](https://codecov.io/gh/rnts08/fairchain-miner)
[![Go Report Card](https://goreportcard.com/badge/github.com/rnts08/fairchain-miner)](https://goreportcard.com/report/github.com/rnts08/fairchain-miner)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/rnts08/fairchain-miner)](https://github.com/rnts08/fairchain-miner/releases)

> A standalone, performance-obsessed miner for Fairchain's `sha256mem` proof-of-work
> algorithm. Extracts and optimizes the critical mining hot path for maximum
> hashrate on server CPUs and GPUs.

---

## Table of Contents

- [Mission](#mission)
- [Algorithm Deep Dive — SHA256-Mem](#algorithm-deep-dive--sha256-mem)
  - [Overview](#overview)
  - [Phase 1: Seed](#phase-1-seed)
  - [Phase 2: Memory Fill (Scratchpad Build)](#phase-2-memory-fill-scratchpad-build)
  - [Phase 3: Mix Pass A](#phase-3-mix-pass-a)
  - [Phase 4: Mix Pass B](#phase-4-mix-pass-b)
  - [Phase 5: Finalize](#phase-5-finalize)
  - [Constants](#constants)
- [Why This Exists](#why-this-exists)
- [Architecture](#architecture)
- [Optimization Strategy](#optimization-strategy)
  - [Tier 1 — Pure Go Micro-Optimizations](#tier-1--pure-go-micro-optimizations)
  - [Tier 2 — Assembly & Intrinsics (CPU)](#tier-2--assembly--intrinsics-cpu)
  - [Tier 3 — GPU Compute (CUDA / OpenCL)](#tier-3--gpu-compute-cuda--opencl)
  - [Tier 4 — Multi-Node & Pool Mining](#tier-4--multi-node--pool-mining)
- [Hardware Targets](#hardware-targets)
- [Relationship to fairchain-src](#relationship-to-fairchain-src)
- [Build](#build)
- [User Guide & Tuning](#user-guide--tuning)
- [License](#license)

---

## Mission

Achieve the highest possible hashrate for Fairchain's `sha256mem` algorithm
across a range of hardware — from commodity laptops to high-core-count server
CPUs and NVIDIA/AMD GPUs — while maintaining **bit-exact consensus correctness**
at all times.

Every optimization must produce the identical hash output as the reference
Go implementation in `internal/algorithms/sha256mem/sha256mem.go`.

---

## Algorithm Deep Dive — SHA256-Mem

### Overview

SHA256-Mem is a memory-hard, CPU-favoring proof-of-work algorithm. It is
designed to give CPUs with large L3 caches and strong single-threaded SHA-256
throughput an economic advantage over GPUs, which suffer from:

1. **Serial SHA-256 dependency chains** — no ILP across mix rounds
2. **Data-dependent memory access** — poor occupancy, unpredictable cache behavior
3. **Large per-thread memory footprint** — 64 MB scratchpad limits SM occupancy

The algorithm processes an 80-byte block header through five phases:

```
Input (80 bytes)
    │
    ▼
┌─────────────────┐
│  Phase 1: Seed  │  SHA-256(header) → 32-byte seed
└────────┬────────┘
         │
         ▼
┌─────────────────────────────┐
│  Phase 2: Memory Fill       │  Build 2,097,152 × 32-byte scratchpad (64 MB)
│  • ARX fill (fast, cheap)   │  • Slots 1..N: ARX(prev_slot, index)
│  • SHA-256 hardening        │  • Every 128th slot: SHA-256(prev_slot)
└────────────┬────────────────┘
             │
             ▼
┌───────────────────────────────┐
│  Phase 3: Mix Pass A          │  32,768 rounds
│  • Data-dependent indexing    │  idx = acc[0:4] % Slots
│  • SHA-256(acc || mem[idx])   │  Each round depends on previous SHA-256
└────────────┬──────────────────┘
             │
             ▼
┌───────────────────────────────┐
│  Phase 4: Mix Pass B          │  32,768 rounds
│  • Rotating offset indexing   │  off = (i % 7) * 4; idx = acc[off:off+4] % Slots
│  • SHA-256(acc || mem[idx])   │  More scattered reads than Pass A
└────────────┬──────────────────┘
             │
             ▼
┌───────────────────────────┐
│  Phase 5: Finalize        │  SHA-256(acc) → reverse byte order → PoW hash
└───────────────────────────┘
```

### Phase 1: Seed

```
seed = SHA-256(header_80_bytes)
```

A single SHA-256 on the 80-byte serialized block header. This is the initial
32-byte state from which the scratchpad is built.

### Phase 2: Memory Fill (Scratchpad Build)

Allocates a **2,097,152-slot array** of 32-byte entries (64 MiB total):

- `mem[0] = seed`
- For `i = 1` to `2,097,151`:
  - If `i % 128 == 0`: `mem[i] = SHA-256(mem[i-1])` — **serial hardening**
  - Otherwise: `mem[i] = ARX_fill(mem[i-1], i)` — **fast non-crypto fill**

**ARX fill** (`arxFill`): For each of 8 × 32-bit words in the slot:
```
v = LE32(src[w*4:])
v ^= (index + w)
v = ROTL(v, 13)
v += LE32(src[w*4:])
→ LE32(dst[w*4:], v)
```

The SHA-256 hardening every 128 slots creates **16,384 SHA-256 dependency
barriers** during the fill. These are the most expensive operations in this phase
and are the primary optimization target (SHA-NI, AVX-512).

### Phase 3: Mix Pass A

Starting with `acc = mem[Slots-1]`:

```
for i = 0 to 32,767:
    idx = LE32(acc[0:4]) % 2,097,152
    buf = acc || mem[idx]          // 64 bytes
    acc = SHA-256(buf)
```

Each round reads a **random** 32-byte slot (data-dependent on the previous hash
output), concatenates with the accumulator, and hashes. This is a **serial chain
of 32,768 SHA-256 operations** with unpredictable 32-byte memory reads.

### Phase 4: Mix Pass B

Identical structure to Pass A, but with a **rotating index offset**:

```
for i = 0 to 32,767:
    off = (i % 7) * 4              // cycles through offsets 0, 4, 8, 12, 16, 20, 24
    idx = LE32(acc[off:off+4]) % 2,097,152
    buf = acc || mem[idx]
    acc = SHA-256(buf)
```

The rotating offset increases the entropy of access patterns, making
prefetching even harder.

### Phase 5: Finalize

```
result = SHA-256(acc)
return byte_reverse(result)        // LE internal byte order
```

### Constants

| Constant | Value | Memory Impact |
|----------|-------|---------------|
| `Slots` | 2,097,152 | 64 MiB scratchpad per hash |
| `HardenInterval` | 128 | SHA-256 every 128th fill slot |
| `MixRounds` | 32,768 | Rounds per mix pass (65,536 SHA-256 total for both passes) |

**Total SHA-256 invocations per hash:**
- Seed: 1
- Fill hardening: 2,097,152 / 128 = 16,384
- Mix Pass A: 32,768
- Mix Pass B: 32,768
- Finalize: 1
- **Total: ~81,922 SHA-256 calls per PoW hash**

---

## Why This Exists

The reference `internal/miner/` is tightly coupled to the full node's chain
state, mempool, consensus engine, and P2P layer. It is designed for correctness
and integration — not raw speed. This miner separates concerns:

| Aspect | `internal/miner/` | `fairchain-miner/` |
|--------|--------------------|--------------------|
| **Coupling** | Chain, mempool, P2P, consensus | RPC client only |
| **Hash impl** | Reference Go (`crypto/sha256`) | Platform-optimized (ASM, SIMD, GPU) |
| **Memory mgmt** | `sync.Pool` shared with node | Dedicated, NUMA-aware allocators |
| **Thread model** | Node's goroutine pool | Pinned workers, CPU affinity |
| **Build** | Part of node binary | Standalone binary, cgo optional |
| **GPU** | Not supported | CUDA / OpenCL backends |

---

## Architecture

```
fairchain-miner/
├── README.md                   # This file
├── TODO.md                     # Actionable task list
├── Makefile                    # Build targets (go, asm, cuda, opencl)
├── go.mod                      # Separate module (imports fairchain-src types)
│
├── cmd/
│   └── fairchain-miner/
│       └── main.go             # CLI entrypoint (flags, RPC setup, worker launch)
│
├── pkg/
│   ├── algorithm/
│   │   ├── sha256mem.go        # Baseline Go port (from internal/algorithms/sha256mem)
│   │   ├── sha256mem_test.go   # Consensus vector tests (bit-exact match)
│   │   ├── sha256mem_opt.go    # Pure Go optimizations (bounds elim, inlining)
│   │   ├── sha256mem_amd64.go  # AMD64 dispatcher (SHA-NI, AVX2, AVX-512)
│   │   ├── sha256mem_amd64.s   # Hand-tuned AMD64 assembly
│   │   ├── sha256mem_arm64.go  # ARM64 dispatcher (SHA CE extensions)
│   │   ├── sha256mem_arm64.s   # Hand-tuned ARM64 assembly
│   │   └── bench_test.go       # Comprehensive benchmarks per codepath
│   │
│   ├── gpu/
│   │   ├── cuda/
│   │   │   ├── sha256mem.cu    # CUDA kernel
│   │   │   ├── bridge.go       # cgo bridge
│   │   │   └── bridge.h        # C header for cgo
│   │   └── opencl/
│   │       ├── sha256mem.cl    # OpenCL kernel
│   │       └── bridge.go       # cgo bridge
│   │
│   ├── worker/
│   │   ├── pool.go             # Worker pool with CPU affinity, NUMA awareness
│   │   ├── nonce.go            # Nonce range partitioning
│   │   └── throttle.go         # Power limit / thermal throttling
│   │
│   ├── rpc/
│   │   ├── client.go           # HTTP client for getblockchaininfo, submitblock, etc.
│   │   └── types.go            # JSON response types
│   │
│   ├── template/
│   │   ├── builder.go          # Block template construction (coinbase, merkle)
│   │   └── builder_test.go     # Template correctness tests
│   │
│   ├── memory/
│   │   ├── allocator.go        # NUMA-aware scratchpad allocator
│   │   ├── pool.go             # Lock-free scratchpad pool
│   │   └── hugepages.go        # Linux hugepage support (2MB / 1GB)
│   │
│   └── metrics/
│       ├── hashrate.go         # EWMA hashrate tracker
│       └── reporter.go         # Console / JSON metrics output
│
└── testdata/
    └── vectors.json            # Known test vectors (input → expected hash)
```

---

## Optimization Strategy

### Tier 1 — Pure Go Micro-Optimizations

**Goal:** 2–3x improvement over reference with zero cgo.

| Optimization | Rationale |
|-------------|-----------|
| Eliminate `sync.Pool` per-hash | Pre-allocate one scratchpad per worker thread |
| Remove bounds checks in ARX fill | `//go:nosplit`, manual bounds elimination |
| Inline `binary.LittleEndian` calls | Avoid function call overhead in tight loops |
| Use `unsafe.Pointer` for word access | Skip `binary.LittleEndian` decoding entirely |
| Pre-compute SHA-256 midstate | The 80-byte header shares 64 bytes across nonce iterations; compute the first SHA-256 block once, iterate only the second block |
| Avoid `sha256.Sum256` allocation | Use `sha256.New()` + `Reset()` + `Sum(buf[:0])` to reuse hasher state |
| Batch nonce serialization | Only the last 4 bytes of the header change per nonce; avoid re-serializing the full 80 bytes |
| Prefetch next memory slot | `runtime.Prefetch` or manually schedule reads ahead of consumption |

### Tier 2 — Assembly & Intrinsics (CPU)

**Goal:** 3–8x over reference, leveraging hardware SHA and SIMD.

| Target | Feature | Expected Benefit |
|--------|---------|-----------------|
| AMD64 (Intel) | SHA-NI extensions | 3–5x SHA-256 throughput |
| AMD64 (AMD) | SHA-NI + AVX2 | 3–5x SHA-256, vectorized ARX fill |
| AMD64 | AVX-512 | 8-wide ARX fill, potential 2-way SHA interleave |
| ARM64 (Apple M-series, Graviton) | SHA CE (Cryptographic Extensions) | 3–4x SHA-256 throughput |
| Both | SIMD ARX fill | 4–8 slots per cycle for non-hardened fill |
| Both | Software prefetch | `PREFETCHT0` / `PRFM` for mix pass reads |

**SHA-256 midstate optimization is the single largest win.** The 80-byte header
means two SHA-256 compression blocks (64 + 16 bytes). The first 64 bytes
(version through bits) are constant across nonce iterations. Compute the
midstate once, then only compress the second 16-byte block (last 12 bytes of
bits padding + 4-byte nonce + SHA padding).

### Tier 3 — GPU Compute (CUDA / OpenCL)

**Goal:** Explore GPU mining despite the algorithm's CPU-favoring design.

The sha256mem algorithm is intentionally hostile to GPUs:
- 64 MB per thread limits SM occupancy on even high-VRAM GPUs
- Serial SHA-256 chains prevent warp-level parallelism
- Data-dependent reads cause warp divergence and cache thrashing

However, high-end GPUs (A100/H100, RTX 4090) have massive memory bandwidth
and reasonable SHA-256 throughput. A carefully tuned kernel might achieve
competitive hashrates despite the design penalty:

| Approach | Tradeoff |
|----------|----------|
| One thread per SM, 64 MB shared/global per thread | Low occupancy, high bandwidth per thread |
| Scratchpad in shared memory (where possible) | SM limit ~128 KB shared, need tiling |
| Scratchpad in L2/HBM with prefetch hints | Latency-tolerant if SHA chain is long enough |
| Hybrid: ARX fill on GPU, mix passes on CPU | PCIe transfer cost may negate benefit |

**This tier is exploratory** — the algorithm was designed to make GPUs uneconomical,
but the margins should be measured, not assumed.

### Tier 4 — Multi-Node & Pool Mining

**Goal:** Scale beyond single-machine limits.

| Feature | Description |
|---------|-------------|
| Stratum V1 client | Connect to fairchain-src's built-in stratum server |
| Multi-GPU support | Distribute work across multiple GPU devices |
| NUMA-aware allocation | Pin workers and memory to the same NUMA node |
| Remote hashrate aggregation | JSON metrics endpoint for monitoring |
| Failover RPC | Automatic reconnect on node failure |

---

## Hardware Targets

| Platform | Optimization Level | Expected Hashrate Multiplier |
|----------|-------------------|------------------------------|
| Generic x86-64 (Go only) | Tier 1 | 2–3x baseline |
| Intel Xeon (Ice Lake+, SHA-NI) | Tier 2 | 5–10x baseline |
| AMD EPYC/Ryzen (Zen 3+, SHA-NI) | Tier 2 | 5–10x baseline |
| Apple M2/M3 (ARM SHA CE) | Tier 2 | 4–8x baseline |
| AWS Graviton3/4 (ARM SHA CE) | Tier 2 | 4–8x baseline |
| NVIDIA RTX 4090 (CUDA) | Tier 3 | TBD (research) |
| NVIDIA A100/H100 (CUDA) | Tier 3 | TBD (research) |
| AMD Radeon (OpenCL) | Tier 3 | TBD (research) |

---

- **Single-thread hashrate** (H/s per core)
- **Multi-thread scaling** (H/s vs worker count)
- **Memory bandwidth utilization** (bytes/s vs theoretical peak)
- **SHA-256 throughput** (compressions/s with and without hardware acceleration)
- **Per-phase breakdown** (fill time, mix A time, mix B time, finalize time)

```bash
# Full benchmark suite
make bench
```

## User Guide & Tuning

For detailed instructions on installation, configuration, performance tuning, and benchmarking, please refer to the USER_GUIDE.md.

## Relationship to fairchain-src

This miner is a **consumer** of fairchain-src, not a fork of it:

- **Imports** `internal/types`, `internal/crypto`, and `internal/coinparams` as
  Go packages (or copies their logic where the module boundary requires it)
- **Never imports** `internal/chain`, `internal/mempool`, `internal/p2p`, or
  `internal/consensus` — the miner talks to the node exclusively via RPC
- **Test vectors** are generated from the reference implementation and frozen
  in `testdata/vectors.json` — any optimization that changes the output fails
  the tests

The only shared code paths are:
1. Block header serialization (`types.BlockHeader.SerializeInto`)
2. SHA-256 (standard library — but we'll replace with hardware-accelerated versions)
3. Compact bits ↔ target conversion (`crypto.CompactToHash`)
4. Merkle root computation (`crypto.ComputeMerkleRoot`)

---

## Installation & Getting Started

Detailed setup instructions, hardware requirements, and troubleshooting steps can be found in the [Installation and Getting Started Guide](DOCS/INSTALLATION_AND_GETTING_STARTED.md).

### Quick Start

```bash
# Build for your current architecture
make build

# Run a local benchmark to check hashrate
./fairchain-miner -benchmark -workers $(nproc)

# Start solo mining (requires a running fairchaind node)
./fairchain-miner -rpc http://127.0.0.1:19445 -user <YOUR_ADDRESS>
```

### Performance Tip (Linux)
Enable 2MB Hugepages for a significant hashrate boost:
```bash
sudo sysctl -w vm.nr_hugepages=1024
```

For more advanced configuration, see [TUNING.md](DOCS/TUNING.md).

## License

Source avaiable for non-commercial use — see [LICENSE](../LICENSE).

# Developer Fee Design

## 1. Objectives
- Provide a professional, interactive TUI for real-time monitoring and configuration.
- Implement a sustainable developer fee mechanism (percentage-based).
- Support binary-only releases with secure credential management.

## 2. Configuration Storage (SQLite)
- Use a local `config.sqlite` to persist settings across restarts.
- **Tables:**
    - `settings`: Global configuration (Pool URL, User, Pass, Reward Address, DevFee%).
    - `stats`: Persistent history for hashrate visualization.
    - `history`: Found blocks and submission results.

## 3. User Interface (Bubble Tea)
- Use `charmbracelet/bubbletea` for the Elm-style state management.
- **Components:**
    - `Header`: Version info, current block height, network difficulty.
    - `Stats`: Real-time EWMA hashrate, efficiency (H/W), and uptime.
    - `Graph`: Lipgloss-styled sparklines for hashrate history.
    - `WorkerTable`: Detailed per-thread utilization and core pinning status.
    - `ConfigForm`: Interactive field editing with validation.

## 4. Developer Fee Mechanism (Time-Slicing)
- **Logic:** Instead of a complex multi-identity Stratum client, we use a simple time-slice scheduler.
- **Execution:**
    - For every 100 minutes of mining, `Fee%` minutes are dedicated to the developer's address.
    - The worker pool is paused, Stratum identity is hot-swapped (via `mining.authorize`), and resumes.
    - Once the slice is finished, it reverts to the user's credentials.
- **Transparency:** The TUI will clearly indicate "Dev Fee Mining Active" to ensure user trust.

## 5. Binary Release Strategy
- **Static Linking:** Ensure `sqlite3` and `libusb` (for future GPU) are statically linked for "portable" binaries.
- **Stripping:** Remove debug symbols to reduce binary size and hinder reverse engineering of proprietary optimizations.
- **Packaging:** Distribute as single-binary executables for Linux/Windows/macOS.

