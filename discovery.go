// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

package gpu

import (
	"fmt"
	"sync"
)

// Device represents a discovered GPU hardware device.
type Device struct {
	Index   int    `json:"index"`
	Name    string `json:"name"`
	Vendor  string `json:"vendor"`
	Backend string `json:"backend"` // "CUDA" or "OpenCL"
	Memory  uint64 `json:"memory_bytes"`
}

var (
	discoveryMu    sync.Mutex
	cachedDevices  []Device
)

// DiscoverDevices returns a list of all available GPU devices across all backends.
// It caches results after the first call to avoid repeated driver overhead.
func DiscoverDevices() []Device {
	discoveryMu.Lock()
	defer discoveryMu.Unlock()

	if cachedDevices != nil {
		return cachedDevices
	}

	var devices []Device

	// Implementation note: The following functions are fulfilled by build-tagged
	// files in pkg/gpu/cuda/ and pkg/gpu/opencl/ respectively.
	
	// 1. Discover CUDA Devices (NVIDIA)
	devices = append(devices, discoverCuda()...)

	// 2. Discover OpenCL Devices (AMD/Intel/NVIDIA Fallback)
	devices = append(devices, discoverOpenCL()...)

	cachedDevices = devices
	return devices
}

// String returns a formatted representation of the device.
func (d Device) String() string {
	return fmt.Sprintf("[%s] %d: %s (%s)", d.Backend, d.Index, d.Name, d.Vendor)
}

// discoverCuda and discoverOpenCL are stubs that should be implemented 
// in build-tagged files to avoid compilation errors on systems without toolkits.
func discoverCuda() []Device { return nil }
func discoverOpenCL() []Device { return nil }