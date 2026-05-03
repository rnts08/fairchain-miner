//go:build !amd64
// +build !amd64

package algorithm

// ARXFill computes the ARX fill for a scratchpad slot using the generic method.
func ARXFill(dst, src *[32]byte, index uint32) {
	arxFillGeneric(dst, src, index)
}
