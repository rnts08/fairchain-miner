// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

//go:build amd64
// +build amd64

package algorithm

// prefetcht0 preloads memory into L1 cache.
// This function is implemented in assembly in sha256mem_amd64.s.
func prefetcht0(addr uintptr)
