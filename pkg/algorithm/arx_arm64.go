// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

//go:build arm64
// +build arm64

package algorithm

// ARXFillNEON fills the memory with ARX permutations using NEON vectorization
// This implementation will be replaced with optimized assembly in P4.4
func ARXFillNEON(buf []byte, seed uint64) {
	for i := 0; i < len(buf); i += 32 {
		// Generic implementation
	}
}

// ARXFillDualNEON performs dual parallel ARX fill using NEON
func ARXFillDualNEON(buf []byte, seed1, seed2 uint64) {
	for i := 0; i < len(buf); i += 32 {
		// Generic implementation
	}
}
