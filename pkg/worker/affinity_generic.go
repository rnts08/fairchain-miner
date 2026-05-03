// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

//go:build !linux
// +build !linux

package worker

// SetAffinity is a no-op on non-Linux platforms.
func SetAffinity(cpuID int) error {
	return nil
}
