// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

//go:build arm64
// +build arm64

package algorithm

import (
	"crypto/sha256"
	"encoding"
	"encoding/binary"
	"fmt"
)

// SHA256ARMCRYPTO computes the SHA-256 hash of data using ARM Cryptographic Extensions.
// This implementation will be replaced with optimized assembly in P4.2
func SHA256ARMCRYPTO(data []byte) [32]byte {
	return sha256.Sum256(data)
}

// SHA256ARMCRYPTOMidstate computes the SHA-256 hash of (midstate + data) using ARM hardware acceleration.
// midstate is the 32-byte state after processing the first 64 bytes of the header.
// This implementation will be replaced with optimized assembly in P4.2
func SHA256ARMCRYPTOMidstate(midstate *[32]byte, data []byte) [32]byte {
	h := sha256.New()
	
	// Use MarshalBinary/UnmarshalBinary to set the state
	// Current Go sha256 marshaled format:
	// magic(4) + state(8*4) + buffer(64) + len(8)
	
	state := make([]byte, 4+32+64+8)
	binary.BigEndian.PutUint32(state[0:], 0x73686103) // magic
	copy(state[4:], midstate[:])                     // state
	// buffer (64 bytes) stays zero
	binary.BigEndian.PutUint64(state[4+32+64:], 64)   // len
	
	unmarshaler := h.(encoding.BinaryUnmarshaler)
	if err := unmarshaler.UnmarshalBinary(state); err != nil {
		panic(fmt.Sprintf("UnmarshalBinary failed: %v", err))
	}
	
	h.Write(data)
	var res [32]byte
	copy(res[:], h.Sum(nil))
	return res
}

// SHA256NEON computes SHA-256 using NEON vectorization for parallel hashing
func SHA256NEON(data []byte) [32]byte {
	return sha256.Sum256(data)
}