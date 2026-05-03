// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

//go:build !amd64
// +build !amd64

package algorithm

var (
	HasSHANI  = false
	HasAVX2   = false
	HasAVX512 = false
)
