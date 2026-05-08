// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

//go:build !linux
// +build !linux

package memory

func AllocateHuge(size int) ([]byte, error) {
	// Generic fallback - use standard heap allocation
	buf := make([]byte, size)
	return buf, nil
}

func AllocateHuge1GB(size int) ([]byte, error) {
	return AllocateHuge(size)
}

func AllocateHugeCustom(size int, pageSize int) ([]byte, error) {
	return AllocateHuge(size)
}

func FreeHuge(data []byte) error {
	// No-op for heap allocated memory
	return nil
}