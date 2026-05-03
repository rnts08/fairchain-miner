// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

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
)

// AllocateHuge attempts to allocate size bytes using Linux huge pages.
// If huge pages are not available or the allocation fails, it falls back
// to a standard mmap allocation.
func AllocateHuge(size int) ([]byte, error) {
	// Round up size to 2MB boundary if we want to be sure about hugepage alignment.
	// 2MB = 2 * 1024 * 1024
	const hugePageSize = 2 * 1024 * 1024
	size = (size + hugePageSize - 1) &^ (hugePageSize - 1)

	// Try with MAP_HUGETLB first.
	data, err := syscall.Mmap(
		-1, 0, size,
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_PRIVATE|syscall.MAP_ANONYMOUS|MAP_HUGETLB,
	)
	if err == nil {
		return data, nil
	}

	// Fallback to regular mmap.
	data, err = syscall.Mmap(
		-1, 0, size,
		syscall.PROT_READ|syscall.PROT_WRITE,
		syscall.MAP_PRIVATE|syscall.MAP_ANONYMOUS,
	)
	if err != nil {
		return nil, fmt.Errorf("mmap failed: %v", err)
	}

	// Try to advise the kernel to use transparent huge pages if enabled.
	// MADV_HUGEPAGE is 14 on Linux.
	_, _, _ = syscall.Syscall(syscall.SYS_MADVISE, uintptr(unsafe.Pointer(&data[0])), uintptr(size), uintptr(14))

	return data, nil
}

// FreeHuge releases memory allocated with AllocateHuge.
func FreeHuge(data []byte) error {
	return syscall.Munmap(data)
}
