// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

package worker

import "sync/atomic"

var (
	globalPowerLimit atomic.Int32
	affinityEnabled  atomic.Bool
)

func init() {
	globalPowerLimit.Store(100)
	affinityEnabled.Store(true)
}

// SetGlobalPowerLimit updates the CPU power limit (1-100%).
func SetGlobalPowerLimit(limit int) {
	if limit < 1 {
		limit = 1
	}
	if limit > 100 {
		limit = 100
	}
	globalPowerLimit.Store(int32(limit))
}

// GetGlobalPowerLimit returns the current active power limit.
func GetGlobalPowerLimit() int {
	return int(globalPowerLimit.Load())
}

// SetAffinityEnabled toggles CPU core pinning.
func SetAffinityEnabled(b bool) { affinityEnabled.Store(b) }
func IsAffinityEnabled() bool  { return affinityEnabled.Load() }