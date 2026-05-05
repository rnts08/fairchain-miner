// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

package memory

import "sync/atomic"

var (
	numaEnabled      atomic.Bool
	hugepagesEnabled atomic.Bool
)

func init() {
	numaEnabled.Store(true)
	hugepagesEnabled.Store(true)
}

func SetNumaEnabled(b bool)      { numaEnabled.Store(b) }
func IsNumaEnabled() bool       { return numaEnabled.Load() }
func SetHugepagesEnabled(b bool) { hugepagesEnabled.Store(b) }
func IsHugepagesEnabled() bool  { return hugepagesEnabled.Load() }