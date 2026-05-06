# Fairchain Miner — Benchmark Results

> All benchmarks are run against the reference Go sha256mem implementation.
> Hardware and Go version are recorded for reproducibility.
> Each optimization tier adds entries below its baseline.

---

## Baseline (Reference Go Implementation)

**Date:** 2026-05-02

**Hardware:**
- CPU: 12th Gen Intel Core i7-1265U (12 threads)
- Architecture: amd64
- OS: Linux

**Go Version:** 1.25.8

### Single-thread

| Benchmark | ns/op | ops/sec (H/s) | allocs/op |
|-----------|-------|----------------|-----------|
| `BenchmarkPoWHash` | 79,829,199 | **~12.5 H/s** | 0 |
| `BenchmarkPoWHash80Byte` | 109,664,688 | **~9.1 H/s** | 0 |

### Multi-thread (12 threads)

| Benchmark | ns/op | ops/sec (H/s) | allocs/op |
|-----------|-------|----------------|-----------|
| `BenchmarkPoWHashParallel-12` | 22,041,268 | **~45.4 H/s** | 0 |

### Analysis

- **Single-thread:** ~12.5 H/s with generic input, ~9.1 H/s with 80-byte header
- **Multi-thread scaling:** 45.4 / 12.5 = **3.6x** across 12 threads (imperfect scaling due to memory bandwidth contention on the 64 MB scratchpad)
- **Per-thread in parallel:** ~3.8 H/s (reduced from 12.5 due to L3 cache pressure)
- **Zero allocations** in the hot path thanks to `sync.Pool` scratchpad reuse

### Bottleneck Breakdown (Estimated)

| Phase | SHA-256 Calls | Est. Time Share |
|-------|--------------|-----------------|
| Seed | 1 | <0.1% |
| Fill (hardening) | 16,384 | ~20% |
| Fill (ARX) | N/A | ~2% |
| Mix Pass A | 32,768 | ~38% |
| Mix Pass B | 32,768 | ~38% |
| Finalize | 1 | <0.1% |
| **Total** | **~81,922** | **100%** |

**Primary optimization target:** SHA-256 compression — accounts for ~96% of compute time.
SHA-NI hardware acceleration should provide 3–5x improvement in SHA-256 throughput.

---

## Tier 1 — Pure Go Optimizations

*Not yet measured. Expected: 2–3x improvement over baseline.*

---

## Tier 2 — Assembly (SHA-NI / AVX2 / ARM SHA CE)

*Not yet measured. Expected: 5–10x improvement over baseline.*

### ARM64 Benchmark Targets

| Hardware | Optimization | Expected H/s |
|----------|--------------|--------------|
| Apple M1 Pro | ARM Crypto + NEON | ~50+ H/s |
| Apple M2 Pro | ARM Crypto + NEON | ~65+ H/s |
| Apple M3 Pro | ARM Crypto + NEON | ~75+ H/s |
| AWS Graviton 3 | ARM Crypto + NEON | ~40+ H/s |

---

## Tier 3 — GPU (CUDA / OpenCL)

*Not yet measured. Expected: TBD (research phase).*

### GPU Benchmark Targets

| Hardware | Acceleration | Expected H/s |
|----------|--------------|--------------|
| NVIDIA RTX 3090 | CUDA | ~500+ H/s |
| NVIDIA RTX 4090 | CUDA | ~800+ H/s |
| NVIDIA A100 | CUDA | ~1200+ H/s |
| AMD RX 7900 XTX | OpenCL | ~600+ H/s |
| Intel Arc A770 | OpenCL | ~350+ H/s |
