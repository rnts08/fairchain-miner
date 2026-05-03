package algorithm

// arxFillDualAVX2 processes two blocks using YMM registers.
func arxFillDualAVX2(dst1, dst2, src1, src2 *[32]byte, idx1, idx2 uint32)

// arxFillDualAVX512 processes two blocks using a single ZMM register.
func arxFillDualAVX512(dst1, dst2, src1, src2 *[32]byte, idx1, idx2 uint32)

// ARXFillDualDispatch selects the fastest implementation at runtime.
func ARXFillDualDispatch(dst1, dst2, src1, src2 *[32]byte, idx1, idx2 uint32) {
	if HasAVX512 {
		arxFillDualAVX512(dst1, dst2, src1, src2, idx1, idx2)
	} else if HasAVX2 {
		arxFillDualAVX2(dst1, dst2, src1, src2, idx1, idx2)
	} else {
		arxFillDualGeneric(dst1, dst2, src1, src2, idx1, idx2)
	}
}
