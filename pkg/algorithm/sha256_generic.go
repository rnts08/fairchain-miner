// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

//go:build !amd64
// +build !amd64

package algorithm

import "crypto/sha256"

func SHA256SHANI(data []byte) [32]byte {
	return sha256.Sum256(data)
}

func SHA256SHANIMidstate(midstate []byte, data []byte) [32]byte {
	// Fallback doesn't actually use midstate optimization for simplicity
	// since this is only called when HasSHANI is true.
	return [32]byte{}
}
