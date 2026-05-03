// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

package algorithm

import (
	"testing"
)

func BenchmarkARXFillGeneric(b *testing.B) {
	var dst, src [32]byte
	for i := 0; i < b.N; i++ {
		arxFillGeneric(&dst, &src, uint32(i))
	}
}

func BenchmarkARXFillAVX2(b *testing.B) {
	if !HasAVX2 {
		b.Skip("AVX2 not supported")
	}
	var dst, src [32]byte
	for i := 0; i < b.N; i++ {
		arxFillAVX2(&dst, &src, uint32(i))
	}
}

func BenchmarkPoWHash(b *testing.B) {
	h := New()
	ws := NewWorkspace()
	data := make([]byte, 80)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.PoWHash(data, ws)
	}
}

func BenchmarkPoWHashRegular(b *testing.B) {
	h := New()
	ws := NewWorkspaceRegular()
	data := make([]byte, 80)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.PoWHash(data, ws)
	}
}
