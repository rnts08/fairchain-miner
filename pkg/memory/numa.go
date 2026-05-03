// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

package memory

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"unsafe"
)

const (
	// NUMA memory policy constants (from linux/mempolicy.h)
	MPOL_DEFAULT    = 0
	MPOL_PREFERRED  = 1
	MPOL_BIND       = 2
	MPOL_INTERLEAVE = 3

	// amd64 syscall numbers
	SYS_MBIND = 237
)

// GetNodeCount returns the number of NUMA nodes available on the system.
func GetNodeCount() int {
	nodes, err := filepath.Glob("/sys/devices/system/node/node*")
	if err != nil || len(nodes) == 0 {
		return 1
	}
	return len(nodes)
}

// GetCPUToNodeMap returns a map from CPU ID to NUMA node ID.
func GetCPUToNodeMap() (map[int]int, error) {
	m := make(map[int]int)
	nodes, err := filepath.Glob("/sys/devices/system/node/node*")
	if err != nil {
		return nil, err
	}

	for _, nodePath := range nodes {
		nodeIDStr := strings.TrimPrefix(filepath.Base(nodePath), "node")
		nodeID, err := strconv.Atoi(nodeIDStr)
		if err != nil {
			continue
		}

		cpulist, err := os.ReadFile(filepath.Join(nodePath, "cpulist"))
		if err != nil {
			continue
		}

		cpus := parseCPULis(strings.TrimSpace(string(cpulist)))
		for _, cpu := range cpus {
			m[cpu] = nodeID
		}
	}

	return m, nil
}

// parseCPULis parses strings like "0-3,6,8-10" into a slice of ints.
func parseCPULis(list string) []int {
	var cpus []int
	if list == "" {
		return cpus
	}
	parts := strings.Split(list, ",")
	for _, part := range parts {
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				continue
			}
			start, _ := strconv.Atoi(rangeParts[0])
			end, _ := strconv.Atoi(rangeParts[1])
			for i := start; i <= end; i++ {
				cpus = append(cpus, i)
			}
		} else {
			cpu, err := strconv.Atoi(part)
			if err == nil {
				cpus = append(cpus, cpu)
			}
		}
	}
	return cpus
}

// BindMemory binds a range of memory to a specific NUMA node.
func BindMemory(data []byte, node int) error {
	if len(data) == 0 {
		return nil
	}

	// Prepare nodemask
	// For node N, we need a bitmask with the N-th bit set.
	// On 64-bit, we can just use an unsigned long.
	mask := uint64(1 << uint(node))
	
	// mbind(void *addr, unsigned long len, int mode, const unsigned long *nodemask, unsigned long maxnode, unsigned int flags)
	_, _, errno := syscall.RawSyscall6(
		SYS_MBIND,
		uintptr(unsafe.Pointer(&data[0])),
		uintptr(len(data)),
		uintptr(MPOL_BIND),
		uintptr(unsafe.Pointer(&mask)),
		uintptr(node+2), // maxnode is max bit + 1. Since we set bit 'node', maxnode is node+1. 
		                 // Actually, some kernels want it to be enough to cover the bit.
		uintptr(0),      // flags
	)

	if errno != 0 {
		return fmt.Errorf("mbind failed: %v", errno)
	}

	return nil
}

// AllocateHugeOnNode attempts to allocate hugepages on a specific NUMA node.
func AllocateHugeOnNode(size int, node int) ([]byte, error) {
	data, err := AllocateHuge(size)
	if err != nil {
		return nil, err
	}

	err = BindMemory(data, node)
	if err != nil {
		// If binding fails, we still have the memory, but it might be on the wrong node.
		// For now, we return the error to let the caller decide.
		return data, err
	}

	return data, nil
}

// GetNodeForCPU returns the NUMA node ID for a given CPU ID.
// Returns -1 if detection fails.
func GetNodeForCPU(cpuID int) int {
	m, err := GetCPUToNodeMap()
	if err != nil {
		return -1
	}
	if node, ok := m[cpuID]; ok {
		return node
	}
	return -1
}
