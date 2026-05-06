// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

//go:build !amd64 && !arm64
// +build !amd64,!arm64

package algorithm

// prefetcht0 preloads memory into L1 cache
// This is a no-op fallback implementation, overridden by assembly on supported architectures
func prefetcht0(addr uintptr) {
	_ = addr
}