// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

//go:build !linux
// +build !linux

package memory

func GetNodeCount() int {
	return 1
}

func GetCPUToNodeMap() (map[int]int, error) {
	return make(map[int]int), nil
}

func BindMemory(data []byte, node int) error {
	// No-op on non-Linux systems
	return nil
}

func AllocateHugeOnNode(size int, node int) ([]byte, error) {
	return AllocateHuge(size)
}

func GetNodeForCPU(cpuID int) int {
	return 0
}