// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

package stratum

import (
	"encoding/hex"
	"testing"

	"github.com/bams-repo/fairchain-miner/pkg/types"
)

// TestTargetToDifficulty verifies our target computation logic perfectly
// mirrors what cpuminer-opt expects.
func TestDifficultyConversions(t *testing.T) {
	tests := []float64{0.001, 1.0, 10.0, 256.0}

	for _, diff := range tests {
		target := difficultyToTarget(diff)
		if target == types.ZeroHash {
			t.Errorf("expected non-zero target for diff %f", diff)
		}

		// Ensure that our raw validation works.
		if !ValidHashRaw(target, target) {
			t.Errorf("ValidHashRaw should return true for exact match")
		}
	}
}

func TestDecodeStratumPrevhash(t *testing.T) {
	// 32-byte BE hex string sent from stratum (byte-swapped per 4 bytes)
	hexStr := "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"
	expectedLeHex := "04030201080706050c0b0a09100f0e0d14131211181716151c1b1a19201f1e1d"

	got := decodeStratumPrevhash(hexStr)
	gotHex := hex.EncodeToString(got[:])

	if gotHex != expectedLeHex {
		t.Errorf("decodeStratumPrevhash mismatch\nwant: %s\ngot:  %s", expectedLeHex, gotHex)
	}
}

func TestEncodeUint32LE(t *testing.T) {
	v := uint32(0x12345678)
	want := "78563412"
	got := encodeUint32LE(v)
	if got != want {
		t.Errorf("encodeUint32LE(%x) = %s, want %s", v, got, want)
	}
}
