// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

//go:build linux
// +build linux

package memory

import (
	"fmt"
	"syscall"
	"unsafe"
)

const (
	// MAP_HUGETLB is a Linux-specific flag for mmap to use huge pages.
	// It is defined as 0x40000 in the Linux kernel.
	MAP_HUGETLB = 0x40000

	// MAP_HUGE_SHIFT is the shift for the huge page size in flags.
	MAP_HUGE_SHIFT = 26
	
	// Page size identifiers for MAP_HUGETLB.
	MAP_HUGE_2MB = 21 << MAP_HUGE_SHIFT
	MAP_HUGE_1GB = 30 << MAP_HUGE_SHIFT
)

// AllocateHuge attempts to allocate size bytes using 2MB Linux huge pages.
func AllocateHuge(size int) ([]byte, error) {
	return AllocateHugeCustom(size, 2*1024*1024)
}

// AllocateHuge1GB attempts to allocate size bytes using 1GB Linux huge pages.
func AllocateHuge1GB(size int) ([]byte, error) {
	return AllocateHugeCustom(size, 1024*1024*1024)
}

// AllocateHugeCustom allocates memory using a specific huge page size.
// pageSize must be either 2MB (2097152) or 1GB (1073741824).
func AllocateHugeCustom(size int, pageSize int) ([]byte, error) {
	// Round up size to pageSize boundary.
	size = (size + pageSize - 1) &^ (pageSize - 1)

	flags := syscall.MAP_PRIVATE | syscall.MAP_ANONYMOUS | MAP_HUGETLB
	if pageSize == 1024*1024*1024 {
		flags |= MAP_HUGE_1GB
	} else if pageSize == 2*1024*1024 {
		flags |= MAP_HUGE_2MB
	}

	// Try with MAP_HUGETLB.
	data, err := syscall.Mmap(
		-1, 0, size,
		syscall.PROT_READ|syscall.PROT_WRITE,
		flags,
	)
	if err == nil {
		return data, nil
	}

	// Fallback to regular mmap for 2MB request.
	if pageSize == 2*1024*1024 {
		data, err = syscall.Mmap(
			-1, 0, size,
			syscall.PROT_READ|syscall.PROT_WRITE,
			syscall.MAP_PRIVATE|syscall.MAP_ANONYMOUS,
		)
		if err != nil {
			return nil, fmt.Errorf("mmap fallback failed: %v", err)
		}

		// Try to advise the kernel to use transparent huge pages.
		// MADV_HUGEPAGE is 14 on Linux.
		_, _, _ = syscall.Syscall(syscall.SYS_MADVISE, uintptr(unsafe.Pointer(&data[0])), uintptr(size), uintptr(14))
		return data, nil
	}

	return nil, fmt.Errorf("hugepage allocation (size %d) failed: %v", pageSize, err)
}

// FreeHuge releases memory allocated with AllocateHuge.
func FreeHuge(data []byte) error {
	return syscall.Munmap(data)
}
