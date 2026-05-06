// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

//go:build !arm64 && !amd64
// +build !arm64,!amd64

package algorithm

import "crypto/sha256"

// SHA256Best automatically selects the fastest available SHA256 implementation
func SHA256Best(data []byte) [32]byte {
	return sha256.Sum256(data)
}

// SHA256MidstateBest automatically selects the fastest available midstate implementation
func SHA256MidstateBest(midstate *[32]byte, data []byte) [32]byte {
	// Fallback full hash calculation
	full := make([]byte, 64 + len(data))
	copy(full[0:64], midstate[:])
	copy(full[64:], data)
	return sha256.Sum256(full)
}

// ARXFillBest automatically selects the fastest available ARX fill implementation
func ARXFillBest(buf []byte, seed uint64) {
	ARXFillGeneric(buf, seed)
}

// ARXFillDualBest automatically selects the fastest available dual ARX fill implementation
func ARXFillDualBest(buf []byte, seed1, seed2 uint64) {
	ARXFillDualGeneric(buf, seed1, seed2)
}
