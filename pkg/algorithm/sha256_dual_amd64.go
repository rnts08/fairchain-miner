//go:build amd64
// +build amd64

package algorithm

import (
	"crypto/sha256"
	"encoding/binary"
)

// sha256_compress_dual_shani is implemented in sha256_dual_amd64.s
func sha256_compress_dual_shani(state1, state2 *[8]uint32, block1, block2 *[64]byte)

// sha256_compress_single_shani is implemented in sha256_single_amd64.s
func sha256_compress_single_shani(state *[8]uint32, block *[64]byte)

// SHA256SHANIDual computes the SHA-256 hash of two data blocks in parallel using SHA-NI.
func SHA256SHANIDual(data1, data2 []byte) ([32]byte, [32]byte) {
	if !HasSHANI {
		// Fallback to single-way standard library
		return sha256.Sum256(data1), sha256.Sum256(data2)
	}

	// For simplicity in this implementation, we assume they are the same length.
	// In the miner, they are always 80 bytes (header) or 64 bytes (mixing).
	if len(data1) != len(data2) {
		return sha256.Sum256(data1), sha256.Sum256(data2)
	}

	// This is a simplified version. A full parallel SHA-256 would handle arbitrary lengths.
	// For our miner, we'll focus on the fixed-size blocks used in the hot path.
	return sha256.Sum256(data1), sha256.Sum256(data2) // Placeholder for now
}

// SHA256SHANIMidstateDual computes two hashes (midstate + data) in parallel.
func SHA256SHANIMidstateDual(mid1, mid2 *[32]byte, data1, data2 []byte) ([32]byte, [32]byte) {
	if !HasSHANI {
		return SHA256SHANIMidstate(mid1, data1), SHA256SHANIMidstate(mid2, data2)
	}

	// Extract states
	var state1, state2 [8]uint32
	for i := 0; i < 8; i++ {
		state1[i] = binary.BigEndian.Uint32(mid1[i*4:])
		state2[i] = binary.BigEndian.Uint32(mid2[i*4:])
	}

	// Prepare final blocks (assuming len(data) <= 55 for simplicity, which is true for header[64:80])
	var block1, block2 [64]byte
	copy(block1[:], data1)
	copy(block2[:], data2)

	// Add padding
	block1[len(data1)] = 0x80
	block2[len(data2)] = 0x80
	
	// Add length (64 + len(data)) in bits
	binary.BigEndian.PutUint64(block1[56:], (64+uint64(len(data1)))*8)
	binary.BigEndian.PutUint64(block2[56:], (64+uint64(len(data2)))*8)

	// Parallel compression
	sha256_compress_dual_shani(&state1, &state2, &block1, &block2)

	// Convert back to [32]byte
	var res1, res2 [32]byte
	for i := 0; i < 8; i++ {
		binary.BigEndian.PutUint32(res1[i*4:], state1[i])
		binary.BigEndian.PutUint32(res2[i*4:], state2[i])
	}
	return res1, res2
}
