package algorithm

import (
	"reflect"
	"testing"
)

func TestARXFillDual(t *testing.T) {
	var dst1, dst2, src1, src2 [32]byte
	for i := range src1 {
		src1[i] = byte(i)
		src2[i] = byte(i + 1)
	}

	idx1 := uint32(100)
	idx2 := uint32(200)

	// Compute reference
	var want1, want2 [32]byte
	ARXFill(&want1, &src1, idx1)
	ARXFill(&want2, &src2, idx2)

	// Compute dual
	ARXFillDualDispatch(&dst1, &dst2, &src1, &src2, idx1, idx2)

	if !reflect.DeepEqual(dst1[:], want1[:]) {
		t.Errorf("Dual ARX mismatch block 1\ngot:  %x\nwant: %x", dst1, want1)
	}
	if !reflect.DeepEqual(dst2[:], want2[:]) {
		t.Errorf("Dual ARX mismatch block 2\ngot:  %x\nwant: %x", dst2, want2)
	}
}

func BenchmarkARXFillSingle(b *testing.B) {
	var dst, src [32]byte
	b.SetBytes(32)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		arxFillAVX2(&dst, &src, uint32(i))
	}
}

func BenchmarkARXFillDual(b *testing.B) {
	var dst1, dst2, src1, src2 [32]byte
	b.SetBytes(64)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ARXFillDualDispatch(&dst1, &dst2, &src1, &src2, uint32(i), uint32(i+1))
	}
}

func BenchmarkARXFillDualAVX2(b *testing.B) {
	if !HasAVX2 {
		b.Skip("AVX2 not supported")
	}
	var dst1, dst2, src1, src2 [32]byte
	b.SetBytes(64)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		arxFillDualAVX2(&dst1, &dst2, &src1, &src2, uint32(i), uint32(i+1))
	}
}
