#ifndef FAIRCHAIN_MINER_CUDA_BRIDGE_H
#define FAIRCHAIN_MINER_CUDA_BRIDGE_H

#include <stdint.h>

// Function to launch the CUDA kernel.
// header_base: 80-byte block header
// start_nonce: starting nonce for this batch
// num_nonces: number of nonces to process
// out_hashes: buffer to store the 32-byte resulting hashes
int sha256mem_cuda_kernel_wrapper(const uint8_t* header_base, uint32_t start_nonce, uint32_t num_nonces, uint8_t* out_hashes);

#endif // FAIRCHAIN_MINER_CUDA_BRIDGE_H