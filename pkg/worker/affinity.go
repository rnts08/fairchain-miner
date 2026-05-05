// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

//go:build linux
// +build linux

package worker

import (
	"runtime"
	"syscall"
	"unsafe"
)

// SetAffinity pins the current goroutine to the specified CPU core.
// It locks the goroutine to the OS thread and then sets the thread's CPU affinity.
func SetAffinity(cpuID int) error {
	runtime.LockOSThread()

	var mask [1024 / 64]uintptr
	mask[cpuID/64] |= 1 << (uint(cpuID) % 64)

	_, _, err := syscall.RawSyscall(
		syscall.SYS_SCHED_SETAFFINITY,
		0, // current thread
		uintptr(len(mask)*8),
		uintptr(unsafe.Pointer(&mask[0])),
	)
	if err != 0 {
		return syscall.Errno(err)
	}
	return nil
}

// UnsetAffinity removes core pinning for the current thread.
func UnsetAffinity() error {
	var mask [1024 / 64]uintptr
	// Set all bits to allow the thread to run on any CPU.
	for i := range mask {
		mask[i] = ^uintptr(0)
	}

	_, _, err := syscall.RawSyscall(
		syscall.SYS_SCHED_SETAFFINITY,
		0,
		uintptr(len(mask)*8),
		uintptr(unsafe.Pointer(&mask[0])),
	)
	if err != 0 {
		return syscall.Errno(err)
	}
	return nil
}
