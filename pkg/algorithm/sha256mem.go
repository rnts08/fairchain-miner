// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

// Package algorithm implements the sha256mem PoW hash function.
// This is the starting point — a direct port of the reference implementation
// from internal/algorithms/sha256mem/sha256mem.go. Optimized variants will
// be added as separate files behind build tags and runtime feature detection.
package algorithm

import (
	"crypto/sha256"
	"encoding/binary"
	"sync"

	"github.com/bams-repo/fairchain/internal/types"
)

// Consensus-critical constants. Changing any of these breaks consensus.
const (
	Slots          = 2097152  // 2^21 = 64 MiB scratchpad (2M × 32 bytes)
	HardenInterval = 128     // SHA-256 hardening every 128 slots
	MixRounds      = 32768   // Rounds per mix pass (A and B)

	// Derived constants for convenience.
	ScratchpadSize = Slots * 32 // Total scratchpad size in bytes (67,108,864)
)

// memPool provides reusable scratchpad allocations for concurrent hashing.
// TODO(P2.1): Replace with per-worker dedicated allocation for zero contention.
var memPool = sync.Pool{
	New: func() any {
		buf := make([][32]byte, Slots)
		return &buf
	},
}

// Hasher implements the sha256mem PoW hash algorithm.
// This is the reference Go implementation — bit-exact match with the node's
// internal/algorithms/sha256mem package.
type Hasher struct{}

// New creates a new sha256mem Hasher.
func New() *Hasher { return &Hasher{} }

// Name returns the algorithm identifier.
func (h *Hasher) Name() string { return "sha256mem" }

// PoWHash computes the sha256mem proof-of-work hash.
// Input is typically the 80-byte serialized block header.
// Output is a 32-byte hash in little-endian (reversed) byte order.
//
// This reference implementation matches internal/algorithms/sha256mem exactly.
// Optimized variants (ASM, GPU) must produce identical output for all inputs.
func (h *Hasher) PoWHash(data []byte) types.Hash {
	// Phase 1: Seed
	seed := sha256.Sum256(data)

	// Acquire scratchpad from pool.
	memPtr := memPool.Get().(*[][32]byte)
	mem := *memPtr

	// Phase 2: Memory fill
	mem[0] = seed
	for i := 1; i < Slots; i++ {
		if i%HardenInterval == 0 {
			mem[i] = sha256.Sum256(mem[i-1][:])
		} else {
			arxFill(&mem[i], &mem[i-1], uint32(i))
		}
	}

	// Phase 3: Mix Pass A
	acc := mem[Slots-1]
	acc = mixPassA(acc, &mem)

	// Phase 4: Mix Pass B
	acc = mixPassB(acc, &mem)

	// Return scratchpad to pool.
	memPool.Put(memPtr)

	// Phase 5: Finalize
	final := sha256.Sum256(acc[:])
	return types.Hash(final).Reversed()
}

// mixPassA runs the first mixing pass with data-dependent memory reads.
// Each round hashes acc||mem[idx] where idx is derived from acc[0:4].
func mixPassA(acc [32]byte, mem *[][32]byte) [32]byte {
	m := *mem
	var buf [64]byte
	for i := 0; i < MixRounds; i++ {
		idx := binary.LittleEndian.Uint32(acc[:4]) % uint32(Slots)
		copy(buf[:32], acc[:])
		copy(buf[32:], m[idx][:])
		acc = sha256.Sum256(buf[:])
	}
	return acc
}

// mixPassB runs the second mixing pass with rotating index offsets.
// The offset cycles through 0, 4, 8, 12, 16, 20, 24 bytes into acc.
func mixPassB(acc [32]byte, mem *[][32]byte) [32]byte {
	m := *mem
	var buf [64]byte
	for i := 0; i < MixRounds; i++ {
		off := (i % 7) * 4
		idx := binary.LittleEndian.Uint32(acc[off:off+4]) % uint32(Slots)
		copy(buf[:32], acc[:])
		copy(buf[32:], m[idx][:])
		acc = sha256.Sum256(buf[:])
	}
	return acc
}

// arxFill performs the ARX (Add-Rotate-XOR) fill for non-hardened slots.
// This is fast non-cryptographic fill to populate the scratchpad cheaply.
func arxFill(dst, src *[32]byte, index uint32) {
	for w := 0; w < 8; w++ {
		v := binary.LittleEndian.Uint32(src[w*4:])
		v ^= index + uint32(w)
		v = (v << 13) | (v >> 19) // ROTL 13
		v += binary.LittleEndian.Uint32(src[w*4:])
		binary.LittleEndian.PutUint32(dst[w*4:], v)
	}
}
