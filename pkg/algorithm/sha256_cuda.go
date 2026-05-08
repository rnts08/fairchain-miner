// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

//go:build cuda
// +build cuda

package algorithm

import (
	"crypto/sha256"
	"fmt"

	"github.com/rnts08/fairchain-miner/pkg/types"
)

type cudaHasher struct {
	deviceID int
}

// getGPUHasher is the build-tagged implementation for CUDA.
// It returns a CUDA-accelerated Hasher.
func getGPUHasher(gpuDeviceID int) Hasher {
	// TODO: In a real scenario, this would initialize CUDA context and kernels for the given deviceID.
	return &cudaHasher{deviceID: gpuDeviceID}
}

func (h *cudaHasher) Name() string { return fmt.Sprintf("sha256mem-cuda-device-%d", h.deviceID) }

func (h *cudaHasher) PoWHash(data []byte, ws *Workspace) types.Hash {
	// TODO: Dispatch to actual CUDA kernel (P6.1)
	return NewCPUHasher().PoWHash(data, ws) // Fallback for now
}

func (h *cudaHasher) PoWHashMidstate(data []byte, ws *Workspace, midstate []byte) types.Hash {
	// TODO: Dispatch to actual CUDA kernel (P6.1)
	return NewCPUHasher().PoWHashMidstate(data, ws, midstate) // Fallback for now
}

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