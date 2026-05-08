// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

//go:build arm64
// +build arm64

package algorithm

func prefetcht0(addr uintptr) {
	_ = addr
}