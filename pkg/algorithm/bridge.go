//go:build cuda

package cuda

/*
#cgo LDFLAGS: -L/usr/local/cuda/lib64 -lcudart
#include "bridge.h"
*/
import "C"
import (
	"fmt"
	"unsafe"
)

// Sha256MemCUDA performs sha256mem hashing on the GPU.
// It takes the header, a starting nonce, the number of nonces to process,
// and a pre-allocated output buffer for hashes.
func Sha256MemCUDA(header []byte, startNonce uint32, numNonces uint32, outputHashes []byte) error {
	if len(header) != 80 {
		return fmt.Errorf("header must be 80 bytes")
	}
	if len(outputHashes) != int(numNonces)*32 {
		return fmt.Errorf("outputHashes buffer size mismatch: expected %d, got %d", int(numNonces)*32, len(outputHashes))
	}

	// Allocate host memory for header and output (if not already done)
	cHeader := (*C.uchar)(C.CBytes(header))
	defer C.free(unsafe.Pointer(cHeader))

	cOutputHashes := (*C.uchar)(C.CBytes(outputHashes))
	defer C.free(unsafe.Pointer(cOutputHashes))

	// Call the C wrapper function which will launch the CUDA kernel
	ret := C.sha256mem_cuda_kernel_wrapper(cHeader, C.uint(startNonce), C.uint(numNonces), cOutputHashes)
	if ret != 0 {
		return fmt.Errorf("CUDA kernel launch failed with error code: %d", ret)
	}

	// Copy results back from C buffer to Go slice
	C.GoBytes(unsafe.Pointer(cOutputHashes), C.int(len(outputHashes)))
	copy(outputHashes, (*[1 << 30]byte)(unsafe.Pointer(cOutputHashes))[:len(outputHashes)])

	return nil
}

// TODO: Add a function to query CUDA device properties and available memory.
// func GetCUDADeviceInfo() (DeviceInfo, error) { ... }

// TODO: Add a function to initialize CUDA context if necessary.
// func InitCUDA() error { ... }