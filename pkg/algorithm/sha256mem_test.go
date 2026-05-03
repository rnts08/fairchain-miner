// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

package algorithm

import (
	"crypto/sha256"
	"encoding"
	"encoding/hex"
	"reflect"
	"testing"

	"github.com/bams-repo/fairchain-miner/pkg/types"
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
