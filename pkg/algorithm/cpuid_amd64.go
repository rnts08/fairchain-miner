// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

//go:build amd64
// +build amd64

package algorithm

import (
	"golang.org/x/sys/cpu"
)

var (
	// HasSHANI indicates support for Intel/AMD SHA Extensions.
	HasSHANI = false
	// HasAVX2 indicates support for Advanced Vector Extensions 2.
	HasAVX2 = false
	// HasAVX512 indicates support for AVX-512 foundation and extensions.
	HasAVX512 = false
)

func cpuid_sha() bool

func init() {
	HasSHANI = cpuid_sha()
	HasAVX2 = cpu.X86.HasAVX2
	HasAVX512 = cpu.X86.HasAVX512
}
