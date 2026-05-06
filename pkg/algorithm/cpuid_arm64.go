// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

//go:build arm64
// +build arm64

package algorithm

import "runtime"

var (
	HasSHANI  = false
	HasAVX2   = false
	HasAVX512 = false
	HasARMCRYPTO = false
	HasNEON = false
)

func init() {
	// ARM64 feature detection
	// Will be expanded with proper CPUID register checking
	HasNEON = true
	
	// Check for ARM Cryptographic Extensions
	// For now default to enabled on Apple Silicon / Graviton
	switch runtime.GOARCH {
	case "arm64":
		HasARMCRYPTO = true
	}
}