// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

package algorithm

import (
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/rnts08/fairchain-miner/pkg/types"
)

type testVector struct {
	Name   string `json:"name"`
	Input  string `json:"input_hex"`
	Output string `json:"output_hex"`
}

// TestVectorRegression loads frozen test vectors from testdata/vectors.json
// and verifies every one matches bit-for-bit. Any optimized codepath that
// changes the hash output will fail this test.
func TestVectorRegression(t *testing.T) {
	vectors := loadVectors(t)
	h := New()
	ws := NewWorkspace()

	for _, v := range vectors {
		t.Run(v.Name, func(t *testing.T) {
			input, err := hex.DecodeString(v.Input)
			if err != nil {
				t.Fatalf("bad input hex: %v", err)
			}

			wantBytes, err := hex.DecodeString(v.Output)
			if err != nil {
				t.Fatalf("bad output hex: %v", err)
			}
			var want types.Hash
			copy(want[:], wantBytes)

			got := h.PoWHash(input, ws)
			if got != want {
				t.Errorf("vector %s mismatch\n  input:    %s\n  expected: %x\n  got:      %x",
					v.Name, v.Input, want, got)
			}
		})
	}
}

func loadVectors(t *testing.T) []testVector {
	t.Helper()

	// Find testdata relative to this source file.
	_, thisFile, _, _ := runtime.Caller(0)
	vecPath := filepath.Join(filepath.Dir(thisFile), "..", "..", "testdata", "vectors.json")

	data, err := os.ReadFile(vecPath)
	if err != nil {
		t.Skipf("test vectors not found at %s: %v (run gen_vectors.go to generate)", vecPath, err)
		return nil
	}

	var vectors []testVector
	if err := json.Unmarshal(data, &vectors); err != nil {
		t.Fatalf("parse vectors.json: %v", err)
	}

	if len(vectors) == 0 {
		t.Fatal("vectors.json is empty")
	}

	t.Logf("loaded %d test vectors", len(vectors))
	return vectors
}
