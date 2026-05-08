// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

package algorithm

import (
	"crypto/sha256"
	"encoding"
	"encoding/binary"
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/rnts08/fairchain-miner/pkg/types"
)

func TestSHANI(t *testing.T) {
	if !HasSHANI {
		t.Skip("SHA-NI not supported on this CPU")
	}

	testCases := [][]byte{
		[]byte(""),
		[]byte("abc"),
		[]byte("The quick brown fox jumps over the lazy dog"),
	}

	for _, tc := range testCases {
		want := sha256.Sum256(tc)
		got := SHA256SHANI(tc)
		if !reflect.DeepEqual(got[:], want[:]) {
			t.Errorf("SHA256SHANI mismatch for %q\ngot:  %x\nwant: %x", tc, got, want)
		}
	}

	// Test Midstate
	header := make([]byte, 80)
	for i := range header {
		header[i] = byte(i)
	}
	want := sha256.Sum256(header)

	h := sha256.New()
	h.Write(header[:64])
	var midstate [32]byte
	marshaled, _ := h.(encoding.BinaryMarshaler).MarshalBinary()
	copy(midstate[:], marshaled[4:36])

	got := SHA256SHANIMidstate(&midstate, header[64:])
	if !reflect.DeepEqual(got[:], want[:]) {
		t.Errorf("SHA256SHANIMidstate mismatch\ngot:  %x\nwant: %x", got, want)
	}
}

func TestSHANISingle(t *testing.T) {
	if !HasSHANI {
		t.Skip("SHA-NI not supported on this CPU")
	}

	header := make([]byte, 80)
	for i := range header {
		header[i] = byte(i)
	}

	want := sha256.Sum256(header)

	h := sha256.New()
	h.Write(header[:64])
	var mid [32]byte
	m, _ := h.(encoding.BinaryMarshaler).MarshalBinary()
	copy(mid[:], m[4:36])

	var state [8]uint32
	for i := 0; i < 8; i++ {
		state[i] = binary.BigEndian.Uint32(mid[i*4:])
	}

	var block [64]byte
	copy(block[:], header[64:])
	block[16] = 0x80
	binary.BigEndian.PutUint64(block[56:], 80*8)

	sha256_compress_single_shani(&state, &block)

	var got [32]byte
	for i := 0; i < 8; i++ {
		binary.BigEndian.PutUint32(got[i*4:], state[i])
	}

	if !reflect.DeepEqual(got[:], want[:]) {
		t.Errorf("Single assembly hash mismatch\ngot:  %x\nwant: %x", got, want)
	}
}

func TestSHANIDual(t *testing.T) {
	if !HasSHANI {
		t.Skip("SHA-NI not supported on this CPU")
	}

	header1 := make([]byte, 80)
	header2 := make([]byte, 80)
	for i := range header1 {
		header1[i] = byte(i)
		header2[i] = byte(i + 1)
	}

	want1 := sha256.Sum256(header1)
	want2 := sha256.Sum256(header2)

	h1 := sha256.New()
	h1.Write(header1[:64])
	var mid1 [32]byte
	m1, _ := h1.(encoding.BinaryMarshaler).MarshalBinary()
	copy(mid1[:], m1[4:36])

	h2 := sha256.New()
	h2.Write(header2[:64])
	var mid2 [32]byte
	m2, _ := h2.(encoding.BinaryMarshaler).MarshalBinary()
	copy(mid2[:], m2[4:36])

	got1, got2 := SHA256SHANIMidstateDual(&mid1, &mid2, header1[64:], header2[64:])

	if !reflect.DeepEqual(got1[:], want1[:]) {
		t.Errorf("Dual hash 1 mismatch\ngot:  %x\nwant: %x", got1, want1)
	}
	if !reflect.DeepEqual(got2[:], want2[:]) {
		t.Errorf("Dual hash 2 mismatch\ngot:  %x\nwant: %x", got2, want2)
	}
}

// TestArxFill tests the non-cryptographic memory fill function for determinism.
func TestArxFill(t *testing.T) {
	var dst, src [32]byte
	// Fill src with a known pattern
	for i := range src {
		src[i] = byte(i)
	}

	ARXFill(&dst, &src, 1234)

	expected, _ := hex.DecodeString("6041bc43e4e540c7698ac54ced2e49d072d3ca55f6774ed97b1cd25effc056e2")
	if !reflect.DeepEqual(dst[:], expected) {
		t.Errorf("arxFill output mismatch\ngot:  %x\nwant: %x", dst[:], expected)
	}
}

// TestCPUvsGPUHasher verifies that the GPU implementation (if available)
// produces identical results to the reference CPU implementation.
func TestCPUvsGPUHasher(t *testing.T) {
	// 1. Get GPU hasher. This will be nil if not built with cuda/opencl tags.
	gpuHasher := getGPUHasher(0)
	if gpuHasher == nil {
		t.Skip("GPU hasher not available in this build (cuda or opencl tags required)")
	}

	cpuHasher := NewCPUHasher()
	ws := NewWorkspace()
	defer ws.Free()

	// Test with an 80-byte header.
	data := make([]byte, 80)
	for i := range data {
		data[i] = byte(i * 7) // Use a non-trivial pattern
	}

	// Test PoWHash
	cpuHash := cpuHasher.PoWHash(data, ws)
	gpuHash := gpuHasher.PoWHash(data, ws)

	if cpuHash != gpuHash {
		t.Errorf("Hasher output mismatch\nCPU: %x\nGPU: %x", cpuHash, gpuHash)
	}

	// Test PoWHashMidstate
	h := sha256.New()
	h.Write(data[:64])
	midstate, err := h.(encoding.BinaryMarshaler).MarshalBinary()
	if err != nil {
		t.Fatalf("failed to marshal midstate: %v", err)
	}

	cpuHashMid := cpuHasher.PoWHashMidstate(data, ws, midstate)
	gpuHashMid := gpuHasher.PoWHashMidstate(data, ws, midstate)

	if cpuHashMid != gpuHashMid {
		t.Errorf("Hasher midstate output mismatch\nCPU: %x\nGPU: %x", cpuHashMid, gpuHashMid)
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

func BenchmarkSHANISingle(b *testing.B) {
	if !HasSHANI {
		b.Skip("SHA-NI not supported")
	}
	var state [8]uint32
	var block [64]byte
	b.SetBytes(64)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sha256_compress_single_shani(&state, &block)
	}
}

func BenchmarkSHANIDual(b *testing.B) {
	if !HasSHANI {
		b.Skip("SHA-NI not supported")
	}
	var state1, state2 [8]uint32
	var block1, block2 [64]byte
	b.SetBytes(128)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sha256_compress_dual_shani(&state1, &state2, &block1, &block2)
	}
}
