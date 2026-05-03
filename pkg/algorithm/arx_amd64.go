//go:build amd64
// +build amd64

package algorithm

// arxFillAVX2 is implemented in arx_amd64.s
func arxFillAVX2(dst, src *[32]byte, index uint32)

// ARXFill computes the ARX fill for a scratchpad slot using the best available method.
func ARXFill(dst, src *[32]byte, index uint32) {
	if HasAVX2 {
		arxFillAVX2(dst, src, index)
	} else {
		arxFillGeneric(dst, src, index)
	}
}
