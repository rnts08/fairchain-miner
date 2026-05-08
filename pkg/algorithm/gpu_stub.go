// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

//go:build !cuda && !opencl
// +build !cuda,!opencl

package algorithm

// getGPUHasher returns nil when GPU support is not compiled in.
func getGPUHasher(gpuDeviceID int) Hasher { return nil }