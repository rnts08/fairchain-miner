# Fairchain Miner — Roadmap & Task List

> Status: `[ ]` = not started, `[~]` = in progress, `[x]` = done
> For performance metrics and benchmarking instructions, see [BENCHMARKING.md](BENCHMARKING.md).

### Documentation
- [x] Consolidate installation and getting started documentation into `DOCS/INSTALLATION_AND_GETTING_STARTED.md`

### Core Development
- [ ] Implement Tier 1 Pure Go micro-optimizations (bounds elimination, manual inlining)
- [ ] Implement Tier 2 Assembly kernels (SHA-NI, AVX-512, ARM Cryptographic Extensions)
- [ ] Add NUMA-aware allocation and worker pinning
- [ ] Implement Stratum V1 client for pool mining

### Benchmarking & Research
- [ ] Benchmark Tier 1 vs Reference Baseline
- [ ] Measure Hugepage impact on different L3 cache sizes
- [ ] Research CUDA/OpenCL feasibility for High-VRAM GPUs (Tier 3)

---
