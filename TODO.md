# Fairchain Miner — Roadmap & Task List

> Status: `[ ]` = not started, `[~]` = in progress, `[x]` = done
> For performance metrics and benchmarking instructions, see [BENCHMARKING.md](BENCHMARKING.md).

---

## ✅ Completed Phases

### Phase 0: Project Scaffolding
- [x] P0.1–P0.15: Core infrastructure, CLI, RPC client, worker pool, and end-to-end regtest mining.

### Phase 1: Reference Algorithm & Verification
- [x] P1.1–P1.4: Ported `sha256mem.go`, established 46 consensus test vectors, and verified reference implementation.
- [x] P1.6–P1.7: Hotspot profiling and isolated `arxFill` implementation.

### Phase 2: Pure Go Optimizations
- [x] P2.1–P2.2: Pre-allocated scratchpads and reusable SHA-256 hashers (zero-allocation hot path).
- [x] P2.3–P2.4: Unsafe word access and bounds check elimination.
- [x] P2.5–P2.6: SHA-256 midstate optimization and batch nonce serialization.
- [x] P2.7–P2.9: Loop unrolling, prefetch hints, and memory copy reduction.
- [x] P2.11: Continuous consensus regression verification.

### Phase 3: AMD64 Assembly Kernels
- [x] P3.1: Runtime CPUID detection (SHA-NI, AVX2, AVX-512).
- [x] P3.2–P3.3: SHA-NI compression and midstate state injection.
- [x] P3.4: **Dual-buffer SHA-NI** (2-way interleaved hashing).
- [x] P3.5–P3.6: **AVX2 & AVX-512 ARX Fill** vectorization.
- [x] P3.7–P3.8: Software prefetch stubs and runtime dispatcher integration.
- [x] P3.11: Consensus verification for all ASM codepaths.

### Phase 5: Memory Subsystem
- [x] P5.1: 2MB Hugepage allocator (`mmap` with `MAP_HUGETLB`).
- [x] P5.2: 1GB Hugepage support for server workloads.
- [x] P5.3: **NUMA-aware allocation** (mbind memory to local CPU node).
- [x] P5.4: CPU affinity/pinning for worker goroutines.
- [x] P5.5: Lock-free scratchpad pool.

### Phase 8: Stratum Protocol (Core)
- [x] P8.1–P8.2: Stratum V1 client and job manager.
- [x] P8.3: **Extranonce rolling logic** (unique search space per worker).
- [x] P8.4–P8.5: Share submission and Vardiff support.
- [x] P8.6: **Automatic Reconnection** with exponential backoff.
- [x] P8.7: Integration testing against fairchain-src pool.

---

## 🚀 Remaining Tasks

### Phase 8: Stratum Protocol (Finalization)
- [x] P8.8 — Fixed critical extranonce2 panic bug

### Phase T1: Interactive TUI (High Priority)
- [x] T1.1 — **Bubble Tea Integration**: bootstrap dashboard with `charmbracelet/bubbletea`.
- [x] T1.2 — **Hashrate Sparklines**: real-time visualization of performance.
- [x] T1.3 — **Interactive Config**: form-based editing for pool/address saved to `config.sqlite`.
- [x] T1.4 — **Hardware Control**: TUI toggles for NUMA/Hugepages/Power limits.

### Phase F1: Sustainable Development (Fee)
- [x] F1.1: Implemented statistical probability based developer fee system
- [x] F1.2: Fee logic integrated into critical code paths
- [x] F1.3: Configurable address and percentage values ready for deployment
- [x] F1.4: Add TUI transparency indicator

### Phase 4: ARM64 Assembly (Apple Silicon / Graviton)
- [ ] P4.1–P4.3: Feature detection and ARM Cryptographic Extensions (SHA2) support.
- [ ] P4.4–P4.5: NEON ARX fill vectorization and ARM64 dispatcher.
- [ ] P4.8: Consensus regression on ARM64.

### Phase 6 & 7: GPU Acceleration (Future)
- [ ] P6.1–P6.11: CUDA implementation and RTX/A100 benchmarking.
- [ ] P7.1–P7.6: OpenCL implementation for AMD/Intel GPUs.

### Phase 10: Release & Deployment
- [ ] P10.1–P10.3: GitHub Actions with cross-compilation matrix.
- [ ] P10.4–P10.6: Versioned binary releases and Dockerization.
