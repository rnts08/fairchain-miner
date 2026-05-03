# Fairchain Miner — Actionable Task List

> Each task is self-contained and can be assigned to a specialist agent.
> Tasks are ordered by dependency — later tasks may depend on earlier ones.
> Status: `[ ]` = not started, `[~]` = in progress, `[x]` = done

---

## Phase 0 — Project Scaffolding

> **Agent profile:** Go project setup, module management

- [x] P0.1 — Create `fairchain-miner/` directory and README.md
- [x] P0.2 — Create `go.mod` with separate module (`github.com/bams-repo/fairchain/fairchain-miner`), importing `fairchain-src` types via replace directive
- [x] P0.3 — Create Makefile with targets: `build`, `build-asm`, `build-cuda`, `build-opencl`, `test`, `bench`, `lint`, `clean`
- [x] P0.4 — Create CLI entrypoint `cmd/fairchain-miner/main.go` with flag parsing (--rpc, --workers, --benchmark, --gpu, --power-limit, --duration, --stratum, --user, --device)
- [x] P0.5 — Create `pkg/rpc/client.go` — HTTP client for node RPC (getblockchaininfo, getblockbyheight, submitblock)
- [x] P0.6 — Create `pkg/rpc/types.go` — JSON response structs matching the node's REST API (merged into client.go)
- [x] P0.7 — Create `pkg/template/builder.go` — block template construction (coinbase tx, merkle root, header assembly)
- [x] P0.8 — Create `pkg/template/builder_test.go` — verify template output matches reference miner's output
- [x] P0.9 — Create `pkg/metrics/hashrate.go` — EWMA hashrate tracker (port from `internal/miner/`)
- [x] P0.10 — Create `pkg/metrics/reporter.go` — console output with hashrate, blocks found, uptime
- [x] P0.11 — Create `pkg/worker/pool.go` — worker pool manager (spawn, cancel, nonce partitioning)
- [x] P0.12 — Create `pkg/worker/nonce.go` — nonce range partitioning across workers (merged into pool.go)
- [x] P0.13 — Create `pkg/worker/throttle.go` — power limit throttling (merged into pool.go)
- [x] P0.14 — Wire all components: main → rpc → template → worker pool → algorithm → submit
- [x] P0.15 — End-to-end test: mine a block on regtest and submit via RPC

---

## Phase 1 — Reference Algorithm Port & Test Vectors

> **Agent profile:** Crypto algorithm specialist, Go testing

- [x] P1.1 — Copy `sha256mem.go` into `pkg/algorithm/sha256mem.go` as the reference implementation
- [x] P1.2 — Create `testdata/vectors.json` — generated 46 test vectors from reference impl (empty input, 80-byte headers, nonce variants, random data)
- [x] P1.3 — Create `pkg/algorithm/vectors_test.go` — load vectors.json and verify reference impl passes all (46/46 pass)
- [x] P1.4 — Create `pkg/algorithm/sha256mem_test.go` — benchmark reference impl: single-thread, parallel, 80-byte header
- [x] P1.5 — Benchmark baseline: record H/s per core on target hardware (Intel, AMD, ARM), document in `BENCHMARKS.md`
- [x] P1.6 — Profile with `pprof`: identify hotspots (expected: SHA-256 in mix passes ~80% of time; actual: arxFill, memmove, and sha256)
- [x] P1.7 — Port `arxFill` function separately with its own unit test and benchmark

---

## Phase 2 — Pure Go Optimizations (Tier 1)

> **Agent profile:** Go performance optimization, unsafe patterns, compiler hints

- [x] P2.1 — **Pre-allocated scratchpad per worker**: replace `sync.Pool` with a dedicated `[Slots][32]byte` allocation per goroutine, passed into PoWHash as a parameter
- [x] P2.2 — **Reusable SHA-256 hasher**: replace `sha256.Sum256()` calls with `sha256.New()` + `Reset()` + `Write()` + `Sum(buf[:0])` to eliminate allocation per hash
- [x] P2.3 — **Inline binary.LittleEndian**: replace `binary.LittleEndian.Uint32` / `PutUint32` with direct `unsafe.Pointer` word access on little-endian platforms (build-tagged)
- [x] P2.4 — **Bounds check elimination in ARX fill**: restructure loop to prove to the compiler that all indices are in-bounds; verify with `go build -gcflags="-d=ssa/check_bce/debug=1"`
- [x] P2.5 — **SHA-256 midstate optimization**: for the seed SHA-256, precompute the compression of the first 64 bytes of the 80-byte header once per template; only re-compress the remaining 16 bytes (nonce block) per nonce iteration
- [x] P2.6 — **Batch nonce serialization**: only write the 4-byte nonce into the pre-serialized 80-byte header buffer instead of calling `SerializeInto` on every iteration
- [x] P2.7 — **Unroll ARX fill inner loop**: manually unroll the 8-iteration word loop
- [x] P2.8 — **Prefetch hints in mix passes**: insert `runtime_prefetch` (via `//go:linkname` or asm stub) to prefetch `mem[next_idx]` one round ahead
- [x] P2.9 — **Reduce memory copies in mix passes**: avoid `copy(buf[:32], acc[:])` + `copy(buf[32:], mem[idx][:])` by constructing the 64-byte buffer in-place or using unsafe overlay
- [ ] P2.10 — **Benchmark each optimization individually**: A/B test every change against the previous baseline; reject regressions
- [ ] P2.11 — **Consensus vector regression**: all vectors from P1.2 must still pass after each optimization

---

## Phase 3 — Platform-Specific Assembly (Tier 2, AMD64)

> **Agent profile:** x86-64 assembly, SHA-NI, AVX2/AVX-512, Go plan9 ASM

- [x] P3.1 — **CPUID detection**: create `pkg/algorithm/cpuid_amd64.go` to detect SHA-NI, AVX2, AVX-512, and select the fastest codepath at init time
- [x] P3.2 — **SHA-NI SHA-256 compression**: implemented via `crypto/sha256` which uses hardware SHA-NI on supported AMD64 CPUs
- [x] P3.3 — **SHA-NI SHA-256 midstate**: implemented using Go's internal state injection (`MarshalBinary`/`UnmarshalBinary`) to process only the final block of the header
- [ ] P3.4 — **SHA-NI dual-buffer SHA-256**: implement a 2-way interleaved SHA-256 for the mix pass
- [ ] P3.5 — **AVX2 ARX fill**: vectorize `arxFill` to process 8 × uint32 in a single YMM register pass
- [ ] P3.6 — **AVX-512 ARX fill**: vectorize `arxFill` to process 16 × uint32 per ZMM register
- [x] P3.7 — **Software prefetch in mix passes**: implemented `PREFETCHT0` stubs in `prefetch_amd64.s`
- [x] P3.8 — **Wire ASM codepaths into dispatcher**: `sha256mem_amd64.go` selects the fastest codepath at runtime
- [ ] P3.9 — **Benchmark ASM vs pure Go on Intel (Ice Lake+)**: document results
- [ ] P3.10 — **Benchmark ASM vs pure Go on AMD (Zen 3+)**: document results
- [x] P3.11 — **Consensus vector regression on AMD64**: all vectors pass in `sha256mem_test.go`

---

## Phase 4 — Platform-Specific Assembly (Tier 2, ARM64)

> **Agent profile:** ARM64 assembly, ARM Cryptographic Extensions, Go plan9 ASM

- [ ] P4.1 — **Feature detection**: create `pkg/algorithm/cpuid_arm64.go` to detect SHA-256 Cryptographic Extensions (`ID_AA64ISAR0_EL1.SHA2`)
- [ ] P4.2 — **ARM SHA CE SHA-256 compression**: implement using `SHA256H`, `SHA256H2`, `SHA256SU0`, `SHA256SU1` in Plan9 ASM
- [ ] P4.3 — **ARM SHA CE midstate**: midstate-aware version for the seed phase
- [ ] P4.4 — **NEON ARX fill**: vectorize using NEON `VEOR`, `VSHL`/`VSRI`, `VADD` for 4 × uint32 per pass
- [ ] P4.5 — **Wire ARM64 codepaths**: `sha256mem_arm64.go` dispatcher
- [ ] P4.6 — **Benchmark on Apple M2/M3**: document results
- [ ] P4.7 — **Benchmark on AWS Graviton3/4**: document results
- [ ] P4.8 — **Consensus vector regression on ARM64**: all vectors must pass

---

## Phase 5 — Memory Subsystem Optimization

> **Agent profile:** Systems programming, NUMA, Linux kernel, memory management

- [x] P5.1 — **Hugepage allocator**: create `pkg/memory/hugepages.go` — allocate scratchpads on 2 MB hugepages (`mmap` with `MAP_HUGETLB`) to reduce TLB misses (64 MB / 4 KB pages = 16,384 TLB entries vs 32 with 2 MB pages)
- [ ] P5.2 — **1 GB hugepage support**: optional 1 GB hugepage allocation for server workloads
- [ ] P5.3 — **NUMA-aware allocation**: create `pkg/memory/numa.go` — detect NUMA topology, allocate scratchpad memory on the same node as the worker's CPU core
- [x] P5.4 — **CPU affinity**: create `pkg/worker/affinity.go` — pin each worker goroutine to a specific CPU core using `sched_setaffinity` (Linux) or equivalent
- [x] P5.5 — **Lock-free scratchpad pool**: create `pkg/memory/pool.go` — pre-allocate N scratchpads, one per worker, no contention on allocation
- [ ] P5.6 — **Benchmark with/without hugepages**: measure TLB miss rate reduction and hashrate impact
- [ ] P5.7 — **Benchmark with/without NUMA pinning**: measure cross-node memory access penalty
- [ ] P5.8 — **Memory bandwidth profiling**: use `perf stat` to measure LLC misses, bandwidth utilization during mix passes

---

## Phase 6 — GPU Mining (Tier 3, CUDA)

> **Agent profile:** CUDA programming, GPU memory management, kernel optimization

- [ ] P6.1 — **Research feasibility**: calculate theoretical throughput — 64 MB per thread × SM count × occupancy vs HBM bandwidth; document expected H/s ceiling
- [ ] P6.2 — **CUDA SHA-256 kernel**: implement single-block SHA-256 compression as a device function
- [ ] P6.3 — **CUDA ARX fill kernel**: implement scratchpad fill with shared memory or L2 cache hints
- [ ] P6.4 — **CUDA mix pass kernel**: implement mix pass A and B with data-dependent global memory reads
- [ ] P6.5 — **CUDA full sha256mem kernel**: chain all phases into a single kernel launch (minimize host↔device synchronization)
- [ ] P6.6 — **cgo bridge**: create `pkg/gpu/cuda/bridge.go` — transfer header data to GPU, receive (nonce, hash) results
- [ ] P6.7 — **Multi-GPU support**: device enumeration, work distribution across GPUs
- [ ] P6.8 — **Benchmark on RTX 4090**: document H/s and compare to optimized CPU
- [ ] P6.9 — **Benchmark on A100/H100**: document H/s (HBM advantage for 64 MB scratchpad)
- [ ] P6.10 — **Consensus vector regression on CUDA**: all vectors must pass
- [ ] P6.11 — **Cost analysis**: H/s per watt and H/s per dollar vs CPU mining

---

## Phase 7 — GPU Mining (Tier 3, OpenCL)

> **Agent profile:** OpenCL programming, cross-vendor GPU compute

- [ ] P7.1 — **OpenCL SHA-256 kernel**: port CUDA kernel to OpenCL
- [ ] P7.2 — **OpenCL ARX fill + mix kernel**: port to OpenCL
- [ ] P7.3 — **cgo bridge for OpenCL**: create `pkg/gpu/opencl/bridge.go`
- [ ] P7.4 — **Benchmark on AMD Radeon (RDNA3)**: document H/s
- [ ] P7.5 — **Benchmark on Intel Arc**: document H/s (if available)
- [ ] P7.6 — **Consensus vector regression on OpenCL**: all vectors must pass

---

## Phase 8 — Stratum Client

> **Agent profile:** Network protocol, mining pools, stratum V1

- [x] P8.1 — **Stratum V1 client**: create `pkg/stratum/client.go` — connect to stratum server, handle subscribe/authorize/notify/submit
- [x] P8.2 — **Job manager**: parse stratum jobs into block templates, handle clean_jobs and job rotation
- [ ] P8.3 — **Extranonce handling**: manage extranonce1/extranonce2 assignment and nonce space partitioning (Needs rolling logic per worker)
- [x] P8.4 — **Share submission**: submit valid shares and handle accept/reject responses
- [x] P8.5 — **Vardiff support**: respond to `mining.set_difficulty` notifications
- [ ] P8.6 — **Reconnect logic**: automatic reconnection with exponential backoff
- [x] P8.7 — **Integration test**: connect to fairchain-src's built-in stratum server and mine shares
- [ ] P8.8 — **Failover**: support multiple stratum endpoints with automatic failover

---

## Phase 9 — TUI & Monitoring

> **Agent profile:** Terminal UI, metrics, monitoring

- [ ] P9.1 — **Rich console output**: hashrate, worker status, block count, uptime, temperature (similar to existing fairchain-qt miner TUI)
- [ ] P9.2 — **JSON metrics endpoint**: HTTP server on configurable port for external monitoring
- [ ] P9.3 — **Prometheus metrics**: optional `/metrics` endpoint for Prometheus scraping
- [ ] P9.4 — **Per-worker stats**: individual hashrate, nonce range, shares submitted per worker
- [ ] P9.5 — **GPU stats**: temperature, utilization, memory usage per GPU device

---

## Phase 10 — CI/CD & Release

> **Agent profile:** DevOps, CI/CD, cross-compilation

- [ ] P10.1 — **GitHub Actions CI**: build + test on Linux amd64, Linux arm64, macOS arm64
- [ ] P10.2 — **ASM build matrix**: test SHA-NI and non-SHA-NI codepaths in CI
- [ ] P10.3 — **CUDA build (optional)**: CI step for CUDA compilation (requires GPU runner or cross-compile only)
- [ ] P10.4 — **Release binaries**: cross-compile for Linux amd64/arm64, macOS arm64, Windows amd64
- [ ] P10.5 — **Docker image**: multi-stage build with optional CUDA support
- [ ] P10.6 — **Version tagging**: integrate with git tags, embed version in binary

---

## Dependency Graph

```
P0 (scaffolding)
 └── P1 (reference port + vectors)
      ├── P2 (pure Go optimizations)
      │    ├── P3 (AMD64 ASM)
      │    ├── P4 (ARM64 ASM)
      │    └── P5 (memory subsystem)
      │         ├── P6 (CUDA)
      │         └── P7 (OpenCL)
      └── P8 (stratum client)
           └── P9 (TUI/monitoring)
                └── P10 (CI/release)
```

> **Parallel tracks:** P3, P4, P5 can proceed independently after P2.
> P6 and P7 can proceed independently after P5.
> P8 is independent of P3–P7.
---

## Phase T1 — Interactive TUI (Side Quest)

> **Agent profile:** Frontend/TUI design, Bubble Tea (Elm architecture), SQLite

- [ ] T1.1 — **Bubble Tea Integration**: bootstrap a dashboard using `charmbracelet/bubbletea` and `lipgloss` for styling
- [ ] T1.2 — **Hashrate Sparklines**: implement real-time hashrate visualization with historical window
- [ ] T1.3 — **Interactive Configuration**: form-based UI to edit reward address, pool URL, and credentials; persist to `config.sqlite`
- [ ] T1.4 — **Hardware Control**: UI toggles for NUMA pinning, GPU selection, and power limits
- [ ] T1.5 — **Log Viewport**: scrollable, color-coded log output integrated into the TUI

---

## Phase F1 — Developer Fee Mechanism (Side Quest)

> **Agent profile:** Systems logic, Stratum protocol, scheduler

- [ ] F1.1 — **Dual Identity Support**: modify `stratum.Client` to support hot-swapping credentials without closing the connection (if supported) or handle quick reconnects
- [ ] F1.2 — **Time-Sliced Scheduler**: implement a robust timer that switches mining to the dev address for `(Fee % * CycleTime)`
- [ ] F1.3 — **Fee Metrics**: separate tracking for dev-fee shares to ensure transparency
- [ ] F1.4 — **Stealth Mode**: optional TUI indicator when dev-fee mining is active
