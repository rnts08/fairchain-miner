# Fairchain Miner — Roadmap & Task List

> Status: `[ ]` = not started, `[~]` = in progress, `[x]` = done
> For performance metrics and benchmarking instructions, see [BENCHMARKING.md](BENCHMARKING.md).

---

## REMAINING TASKS

### Phase 1: Proper CI/CD With multi-arch build releases
- [ ] P1.1: Automatic linting, format, race and testing on push to main 
- [ ] P1.2: Automatic base benchmarking if possible with github CI or external provider
- [ ] P1.3: Automatic building of multi-arch releases for different platforms, packages and with shasum documets and release notes.
- [ ] P1.4: Update Readme me with status badges for security scan, code quality and test coverage
 
### Phase 4: ARM64 Assembly (Apple Silicon / Graviton)
- [~] P4.2: ARM Cryptographic Extensions (SHA2) implementation (base stubs complete)
- [~] P4.4: NEON ARX fill vectorization (base stubs complete)
- [ ] P4.5: Full in depth documenation and discussion in differnt implementation details, catches and optimization parts.
---

### Phase 6 & 7: GPU Acceleration (Future)
- [~] P6.1: CUDA kernel implementation (base stubs complete)
- [~] P7.1: OpenCL kernel implementation (base stubs complete)
- [ ] P7.6: Multi-vendor GPU support (AMD/Intel/NVIDIA)
- [ ] P7.7: Full in depth documenation and discussion in differnt implementation details, catches and optimization parts.
---

### Phase 10: In TUI Documentation, attribution and share implementation
- [ ] P10.1: Implement the dev fee properlu
- [ ] P10.2: Tuning information for operating systems and the application on specific hardware
- [ ] P10.3: Supporting the project and Licensing
- [ ] P10.4: General polish, look and feel and logic of the TUI must be intuitive to work with and look nice but not drain too much resources. Such as minimize to tray with mouseover with hashrate etc.

## Pages 11: Cleanup for final 1.0 release
- [ ] P11.1: Remove, consolidate and finalize documentation - no development docs, no multiple docs and only clear instructions for how to use the application on different systems.
- [ ] P11.2: Verify that github action CI/CD build successfully build for linux-amd64/arm64, mac-arm64/mac-silicon and windows-amd64
- [ ] P11.3: Check how to make the miner a drop-in replacement miner for srbminer/ocminer on linux mining os systems
- [ ] P11.4: Umbrel and other similar services with basic webui click and run for home users.