// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

//go:build amd64 || arm64
// +build amd64 arm64

package algorithm

//go:noescape
func prefetcht0(addr uintptr)