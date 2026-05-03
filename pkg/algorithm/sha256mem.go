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
	"encoding"
	"encoding/binary"
	"hash"
	"unsafe"

	"github.com/bams-repo/fairchain-miner/pkg/memory"
	"github.com/bams-repo/fairchain-miner/pkg/types"
)

// prefetcht0 is implemented in assembly for supported architectures.
func prefetcht0(addr uintptr)

// Consensus-critical constants. Changing any of these breaks consensus.
const (
	Slots          = 2097152  // 2^21 = 64 MiB scratchpad (2M × 32 bytes)
	HardenInterval = 128     // SHA-256 hardening every 128 slots
	MixRounds      = 32768   // Rounds per mix pass (A and B)

	// Derived constants for convenience.
	ScratchpadSize = Slots * 32 // Total scratchpad size in bytes (67,108,864)
)

// Workspace holds per-goroutine pre-allocated memory and hashers.
// This eliminates heap allocations in the mining hot path.
type Workspace struct {
	Mem    *[][32]byte
	Hasher hash.Hash
	buf    [32]byte
	mixBuf [64]byte // P2.9: Pre-allocated buffer for mixing passes
}

// NewWorkspace creates a new Workspace for a single worker.
func NewWorkspace() *Workspace {
	raw, err := memory.AllocateHuge(ScratchpadSize)
	if err != nil {
		// Fallback to regular allocation if mmap fails completely.
		mem := make([][32]byte, Slots)
		return &Workspace{
			Mem:    &mem,
			Hasher: sha256.New(),
		}
	}
	
	// Cast the raw byte slice to [][32]byte using unsafe.Slice.
	memSlice := unsafe.Slice((*[32]byte)(unsafe.Pointer(&raw[0])), Slots)
	
	return &Workspace{
		Mem:    &memSlice,
		Hasher: sha256.New(),
	}
}

// Free releases the memory associated with the Workspace.
func (ws *Workspace) Free() {
	if ws.Mem != nil {
		m := *ws.Mem
		raw := unsafe.Slice((*byte)(unsafe.Pointer(&m[0])), ScratchpadSize)
		_ = memory.FreeHuge(raw)
	}
}

// sum256 is a helper to compute SHA-256 without allocation.
func (ws *Workspace) sum256(data []byte) [32]byte {
	ws.Hasher.Reset()
	ws.Hasher.Write(data)
	hashSlice := ws.Hasher.Sum(ws.buf[:0])
	var res [32]byte
	copy(res[:], hashSlice)
	return res
}

// sum256Midstate computes SHA-256 using a precomputed midstate (first 64 bytes).
// This eliminates the need to hash the first 64 bytes of the header for every nonce.
func (ws *Workspace) sum256Midstate(data []byte, midstate []byte) [32]byte {
	// data must be the full 80-byte header, but we only write the remaining 16 bytes.
	_ = data[79] // bounds check hint
	ws.Hasher.(encoding.BinaryUnmarshaler).UnmarshalBinary(midstate)
	ws.Hasher.Write(data[64:80])
	hashSlice := ws.Hasher.Sum(ws.buf[:0])
	var res [32]byte
	copy(res[:], hashSlice)
	return res
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
func (h *Hasher) PoWHash(data []byte, ws *Workspace) types.Hash {
	return h.PoWHashMidstate(data, ws, nil)
}

// PoWHashMidstate computes the PoW hash using an optional precomputed midstate.
func (h *Hasher) PoWHashMidstate(data []byte, ws *Workspace, midstate []byte) types.Hash {
	// Phase 1: Seed
	var seed [32]byte
	if midstate != nil && len(data) == 80 {
		seed = ws.sum256Midstate(data, midstate)
	} else {
		seed = ws.sum256(data)
	}

	// Acquire scratchpad from workspace.
	mem := *ws.Mem

	// Phase 2: Memory fill
	mem[0] = seed
	for i := 1; i < Slots; i++ {
		if i%HardenInterval == 0 {
			mem[i] = ws.sum256(mem[i-1][:])
		} else {
			arxFill(&mem[i], &mem[i-1], uint32(i))
		}
	}

	// Phase 3: Mix Pass A
	acc := mem[Slots-1]
	acc = mixPassA(acc, &mem, ws)

	// Phase 4: Mix Pass B
	acc = mixPassB(acc, &mem, ws)

	// Phase 5: Finalize
	final := ws.sum256(acc[:])
	return types.Hash(final).Reversed()
}

// mixPassA runs the first mixing pass with data-dependent memory reads.
// Each round hashes acc||mem[idx] where idx is derived from acc[0:4].
func mixPassA(acc [32]byte, mem *[][32]byte, ws *Workspace) [32]byte {
	m := *mem
	copy(ws.mixBuf[:32], acc[:])
	
	for i := 0; i < MixRounds; i++ {
		idx := binary.LittleEndian.Uint32(ws.mixBuf[:4]) % uint32(Slots)
		
		// P2.8: Prefetch the memory slot for the next round if possible? 
		// Since it is data-dependent, we prefetch the current one as early as we can.
		prefetcht0(uintptr(unsafe.Pointer(&m[idx])))
		
		// P2.9: Use the pre-allocated mixBuf to avoid intermediate copies.
		copy(ws.mixBuf[32:], m[idx][:])
		
		ws.Hasher.Reset()
		ws.Hasher.Write(ws.mixBuf[:])
		// Sum directly into the first 32 bytes of mixBuf for the next round.
		ws.Hasher.Sum(ws.mixBuf[:0])
	}
	copy(acc[:], ws.mixBuf[:32])
	return acc
}

// mixPassB runs the second mixing pass with rotating index offsets.
// The offset cycles through 0, 4, 8, 12, 16, 20, 24 bytes into acc.
func mixPassB(acc [32]byte, mem *[][32]byte, ws *Workspace) [32]byte {
	m := *mem
	copy(ws.mixBuf[:32], acc[:])
	
	for i := 0; i < MixRounds; i++ {
		off := (i % 7) * 4
		idx := binary.LittleEndian.Uint32(ws.mixBuf[off:off+4]) % uint32(Slots)
		
		prefetcht0(uintptr(unsafe.Pointer(&m[idx])))
		
		copy(ws.mixBuf[32:], m[idx][:])
		
		ws.Hasher.Reset()
		ws.Hasher.Write(ws.mixBuf[:])
		ws.Hasher.Sum(ws.mixBuf[:0])
	}
	copy(acc[:], ws.mixBuf[:32])
	return acc
}

// arxFill performs the ARX (Add-Rotate-XOR) fill for non-hardened slots.
// This is fast non-cryptographic fill to populate the scratchpad cheaply.
func arxFill(dst, src *[32]byte, index uint32) {
	// Unrolled ARX fill (P2.7)
	v0 := binary.LittleEndian.Uint32(src[0:4])
	v0 ^= index
	v0 = (v0 << 13) | (v0 >> 19)
	v0 += binary.LittleEndian.Uint32(src[0:4])
	binary.LittleEndian.PutUint32(dst[0:4], v0)

	v1 := binary.LittleEndian.Uint32(src[4:8])
	v1 ^= index + 1
	v1 = (v1 << 13) | (v1 >> 19)
	v1 += binary.LittleEndian.Uint32(src[4:8])
	binary.LittleEndian.PutUint32(dst[4:8], v1)

	v2 := binary.LittleEndian.Uint32(src[8:12])
	v2 ^= index + 2
	v2 = (v2 << 13) | (v2 >> 19)
	v2 += binary.LittleEndian.Uint32(src[8:12])
	binary.LittleEndian.PutUint32(dst[8:12], v2)

	v3 := binary.LittleEndian.Uint32(src[12:16])
	v3 ^= index + 3
	v3 = (v3 << 13) | (v3 >> 19)
	v3 += binary.LittleEndian.Uint32(src[12:16])
	binary.LittleEndian.PutUint32(dst[12:16], v3)

	v4 := binary.LittleEndian.Uint32(src[16:20])
	v4 ^= index + 4
	v4 = (v4 << 13) | (v4 >> 19)
	v4 += binary.LittleEndian.Uint32(src[16:20])
	binary.LittleEndian.PutUint32(dst[16:20], v4)

	v5 := binary.LittleEndian.Uint32(src[20:24])
	v5 ^= index + 5
	v5 = (v5 << 13) | (v5 >> 19)
	v5 += binary.LittleEndian.Uint32(src[20:24])
	binary.LittleEndian.PutUint32(dst[20:24], v5)

	v6 := binary.LittleEndian.Uint32(src[24:28])
	v6 ^= index + 6
	v6 = (v6 << 13) | (v6 >> 19)
	v6 += binary.LittleEndian.Uint32(src[24:28])
	binary.LittleEndian.PutUint32(dst[24:28], v6)

	v7 := binary.LittleEndian.Uint32(src[28:32])
	v7 ^= index + 7
	v7 = (v7 << 13) | (v7 >> 19)
	v7 += binary.LittleEndian.Uint32(src[28:32])
	binary.LittleEndian.PutUint32(dst[28:32], v7)
}
