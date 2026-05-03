package algorithm

import (
	"encoding/binary"
	"math/bits"
)

// ARXFillDual is a reference implementation for processing two blocks.
func ARXFillDual(dst1, dst2, src1, src2 *[32]byte, idx1, idx2 uint32) {
	ARXFill(dst1, src1, idx1)
	ARXFill(dst2, src2, idx2)
}

// arxFillDualGeneric is the pure Go fallback for the dual assembly.
func arxFillDualGeneric(dst1, dst2, src1, src2 *[32]byte, idx1, idx2 uint32) {
	for j := 0; j < 8; j++ {
		s1 := binary.LittleEndian.Uint32(src1[j*4 : j*4+4])
		v1 := s1 ^ (idx1 + uint32(j))
		v1 = bits.RotateLeft32(v1, 13)
		binary.LittleEndian.PutUint32(dst1[j*4:j*4+4], v1+s1)

		s2 := binary.LittleEndian.Uint32(src2[j*4 : j*4+4])
		v2 := s2 ^ (idx2 + uint32(j))
		v2 = bits.RotateLeft32(v2, 13)
		binary.LittleEndian.PutUint32(dst2[j*4:j*4+4], v2+s2)
	}
}
