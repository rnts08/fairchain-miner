// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

//go:build cuda
// +build cuda

package algorithm

import (
	"crypto/sha256"
)

// SHA256CUDA computes SHA-256 hash using NVIDIA CUDA GPU acceleration
// Base implementation stub - will be replaced with actual CUDA kernel calls
func SHA256CUDA(data []byte) [32]byte {
	return sha256.Sum256(data)
}

// SHA256CUDAMidstate computes midstate SHA-256 using CUDA acceleration
func SHA256CUDAMidstate(midstate *[32]byte, data []byte) [32]byte {
	// Stub implementation - actual CUDA kernel will be implemented in P6.1
	h := sha256.New()
	state := make([]byte, 4+32+64+8)
	copy(state[4:], midstate[:])
	h.Write(data)
	var res [32]byte
	copy(res[:], h.Sum(nil))
	return res
}

// ARXFillCUDA performs ARX fill operation on GPU
func ARXFillCUDA(buf []byte, seed uint64) {
	ARXFillGeneric(buf, seed)
}

// ARXFillDualCUDA performs dual parallel ARX fill on GPU
func ARXFillDualCUDA(buf []byte, seed1, seed2 uint64) {
	ARXFillDualGeneric(buf, seed1, seed2)
}