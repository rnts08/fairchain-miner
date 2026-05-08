// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

//go:build opencl
// +build opencl

package algorithm

import (
	"crypto/sha256"
	"fmt"

	"github.com/rnts08/fairchain-miner/pkg/types"
)

type openclHasher struct {
	deviceID int
}

// getGPUHasher is the build-tagged implementation for OpenCL.
// It should return a GPU-accelerated Hasher if available, otherwise nil.
func getGPUHasher(gpuDeviceID int) Hasher {
	// TODO: In a real scenario, this would initialize OpenCL context and kernels for the given deviceID.
	return &openclHasher{deviceID: gpuDeviceID}
}

func (h *openclHasher) Name() string { return fmt.Sprintf("sha256mem-opencl-device-%d", h.deviceID) }

func (h *openclHasher) PoWHash(data []byte, ws *Workspace) types.Hash {
	// TODO: Dispatch to actual OpenCL kernel (P7.1)
	return NewCPUHasher().PoWHash(data, ws) // Fallback for now
}

func (h *openclHasher) PoWHashMidstate(data []byte, ws *Workspace, midstate []byte) types.Hash {
	// TODO: Dispatch to actual OpenCL kernel (P7.1)
	return NewCPUHasher().PoWHashMidstate(data, ws, midstate) // Fallback for now
}

// SHA256OpenCL computes SHA-256 hash using OpenCL GPU acceleration
// Base implementation stub - will be replaced with actual OpenCL kernel calls
func SHA256OpenCL(data []byte) [32]byte {
	return sha256.Sum256(data)
}

// SHA256OpenCLMidstate computes midstate SHA-256 using OpenCL acceleration
func SHA256OpenCLMidstate(midstate *[32]byte, data []byte) [32]byte {
	// Stub implementation - actual OpenCL kernel will be implemented in P7.1
	h := sha256.New()
	state := make([]byte, 4+32+64+8)
	copy(state[4:], midstate[:])
	h.Write(data)
	var res [32]byte
	copy(res[:], h.Sum(nil))
	return res
}

// ARXFillOpenCL performs ARX fill operation on OpenCL device
func ARXFillOpenCL(buf []byte, seed uint64) {
	ARXFillGeneric(buf, seed)
}

// ARXFillDualOpenCL performs dual parallel ARX fill on OpenCL device
func ARXFillDualOpenCL(buf []byte, seed1, seed2 uint64) {
	ARXFillDualGeneric(buf, seed1, seed2)
}