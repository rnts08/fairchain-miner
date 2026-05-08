# Hardware and OS Tuning Guide

To achieve maximum performance with the `sha256mem` algorithm, your system needs to be tuned to handle high memory throughput and low-latency SHA-256 operations.

## 1. Operating System Optimizations (Linux)

### Hugepages (Crucial)
Each mining worker uses a 64MB scratchpad. Using standard 4KB pages results in frequent TLB misses. Enabling 2MB Hugepages can improve hashrate by 10-15%.

**Temporary Enable:**
```bash
sudo sysctl -w vm.nr_hugepages=1024
```

**Permanent Enable:**
Add `vm.nr_hugepages=1024` to `/etc/sysctl.conf`.

### CPU Governor
Ensure your CPU is not down-clocking during mining.
```bash
echo performance | sudo tee /sys/devices/system/cpu/cpu*/cpufreq/scaling_governor
```

### NUMA (Non-Uniform Memory Access)
On multi-socket systems (Threadripper, EPYC, Xeon), memory access latency varies depending on which CPU socket is accessing which RAM slot. The miner is NUMA-aware and will attempt to bind memory to the local node of the CPU worker. Ensure `numactl` is installed on your system.

## 2. Hardware Specifics

### Intel / AMD (x86_64)
- **SHA-NI**: Ensure your CPU supports SHA Extensions. This provides a ~4x boost in the hardening and mixing phases.
- **SMT / Hyper-Threading**: Mining is cache-heavy. Often, using only physical cores (e.g., `-workers 8` on a 16-thread CPU) yields a higher total hashrate because logical pairs share L1/L2 cache, causing contention.

### Apple Silicon (M1/M2/M3)
- **Performance vs Efficiency Cores**: The miner will automatically use all cores. However, the Efficiency (E) cores contribute significantly less to the hashrate while consuming some memory bandwidth. You may find better results by setting `-workers` to the number of Performance (P) cores.
- **Unified Memory**: Apple's architecture has extremely low latency, making it very efficient for the data-dependent reads in the Mix Passes.

### ARM Server (Graviton)
- **L3 Cache**: `sha256mem` benefits from large L3 caches. High-core-count ARM chips often have distributed caches; ensure you are not over-subscribing the memory bandwidth.

## 3. Application Tuning

### The Power Limit Flag
If you are mining on a laptop or in a thermally constrained environment, use `-power-limit 70` to reduce heat. This introduces small sleeps between batches to allow the hardware to cool, preventing aggressive thermal throttling by the OS which can lead to "choppy" hashrate.
```