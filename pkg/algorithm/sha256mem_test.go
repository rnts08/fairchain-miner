// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

package algorithm

import (
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/bams-repo/fairchain-miner/pkg/types"
)

// TestArxFill tests the non-cryptographic memory fill function for determinism.
func TestArxFill(t *testing.T) {
	var dst, src [32]byte
	// Fill src with a known pattern
	for i := range src {
		src[i] = byte(i)
	}

	arxFill(&dst, &src, 1234)

	expected, _ := hex.DecodeString("6041bc43e4e540c7698ac54ced2e49d072d3ca55f6774ed97b1cd25effc056e2")
	if !reflect.DeepEqual(dst[:], expected) {
		t.Errorf("arxFill output mismatch\ngot:  %x\nwant: %x", dst[:], expected)
	}
}

// TestPoWHashKnownVector verifies the miner's implementation matches the
// reference node implementation bit-for-bit. This vector is consensus-critical.
func TestPoWHashKnownVector(t *testing.T) {
	h := New()
	ws := NewWorkspace()

	// Empty input known vector — must match internal/algorithms/sha256mem exactly.
	// BE output: c44266989d33a18aeefde8a63588433ff51d61a0afddff88fe9b533bc2d19469
	// PoWHash returns LE (reversed):
	input := []byte{}
	want, _ := hex.DecodeString("6994d1c23b539bfe88ffddafa0611df53f438835a6e8fdee8aa1339d986642c4")
	var expected types.Hash
	copy(expected[:], want)

	got := h.PoWHash(input, ws)
	if got != expected {
		t.Fatalf("known vector mismatch\n  expected %x\n  got      %x", expected, got)
	}
}

// TestPoWHashDeterministic verifies identical inputs produce identical outputs.
func TestPoWHashDeterministic(t *testing.T) {
	h := New()
	ws := NewWorkspace()
	input := []byte("test vector for sha256mem pow hash")
	got1 := h.PoWHash(input, ws)
	got2 := h.PoWHash(input, ws)

	if got1 == types.ZeroHash {
		t.Fatal("PoWHash returned zero hash")
	}
	if got1 != got2 {
		t.Fatal("PoWHash is not deterministic")
	}
}

// TestPoWHashDifferentInputs verifies different inputs produce different hashes.
func TestPoWHashDifferentInputs(t *testing.T) {
	h := New()
	ws := NewWorkspace()
	a := h.PoWHash([]byte("input A"), ws)
	b := h.PoWHash([]byte("input B"), ws)

	if a == b {
		t.Fatal("different inputs produced the same hash")
	}
}

// TestPoWHash80ByteHeader tests with a realistic 80-byte block header input.
func TestPoWHash80ByteHeader(t *testing.T) {
	h := New()
	ws := NewWorkspace()
	var header [80]byte
	for i := range header {
		header[i] = byte(i)
	}

	got := h.PoWHash(header[:], ws)
	if got == types.ZeroHash {
		t.Fatal("PoWHash of 80-byte header returned zero hash")
	}

	// Verify deterministic.
	got2 := h.PoWHash(header[:], ws)
	if got != got2 {
		t.Fatal("80-byte header hash is not deterministic")
	}
}

// TestConcurrentSafety verifies the hasher is safe for concurrent use.
func TestConcurrentSafety(t *testing.T) {
	h := New()
	ws := NewWorkspace()
	input := []byte("concurrent test data")
	expected := h.PoWHash(input, ws)

	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func() {
			ws := NewWorkspace()
			defer func() { done <- struct{}{} }()
			for j := 0; j < 5; j++ {
				got := h.PoWHash(input, ws)
				if got != expected {
					t.Errorf("concurrent PoWHash mismatch")
					return
				}
			}
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}

// BenchmarkPoWHash benchmarks single-threaded hash throughput.
func BenchmarkPoWHash(b *testing.B) {
	h := New()
	ws := NewWorkspace()
	input := []byte("benchmark input for sha256mem")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.PoWHash(input, ws)
	}
}

// BenchmarkPoWHashParallel benchmarks multi-threaded hash throughput.
func BenchmarkPoWHashParallel(b *testing.B) {
	h := New()
	input := []byte("benchmark input for sha256mem")
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		ws := NewWorkspace()
		for pb.Next() {
			h.PoWHash(input, ws)
		}
	})
}

// BenchmarkPoWHash80Byte benchmarks with realistic 80-byte header input.
func BenchmarkPoWHash80Byte(b *testing.B) {
	h := New()
	ws := NewWorkspace()
	var header [80]byte
	for i := range header {
		header[i] = byte(i)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.PoWHash(header[:], ws)
	}
}

// BenchmarkArxFill benchmarks the arxFill function.
func BenchmarkArxFill(b *testing.B) {
	var dst, src [32]byte
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		arxFill(&dst, &src, uint32(i))
	}
}
