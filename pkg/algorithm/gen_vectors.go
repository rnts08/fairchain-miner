// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

//go:build ignore

// gen_vectors generates frozen test vectors for the sha256mem PoW hash.
// Run: go run gen_vectors.go > ../testdata/vectors.json
package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/rnts08/fairchain-miner/pkg/algorithm"
)

type vector struct {
	Name   string `json:"name"`
	Input  string `json:"input_hex"`
	Output string `json:"output_hex"`
}

func main() {
	h := algorithm.New()
	ws := algorithm.NewWorkspace()

	var vectors []vector

	// 1. Empty input.
	empty := h.PoWHash([]byte{}, ws)
	vectors = append(vectors, vector{
		Name:   "empty_input",
		Input:  "",
		Output: hex.EncodeToString(empty[:]),
	})

	// 2. Single byte inputs.
	for i := 0; i < 5; i++ {
		input := []byte{byte(i)}
		out := h.PoWHash(input, ws)
		vectors = append(vectors, vector{
			Name:   fmt.Sprintf("single_byte_%d", i),
			Input:  hex.EncodeToString(input),
			Output: hex.EncodeToString(out[:]),
		})
	}

	// 3. Known strings.
	knownStrings := []string{
		"fairchain",
		"sha256mem",
		"test vector for sha256mem pow hash",
		"benchmark input for sha256mem",
	}
	for _, s := range knownStrings {
		input := []byte(s)
		out := h.PoWHash(input, ws)
		vectors = append(vectors, vector{
			Name:   fmt.Sprintf("string_%s", s),
			Input:  hex.EncodeToString(input),
			Output: hex.EncodeToString(out[:]),
		})
	}

	// 4. 80-byte block headers (sequential fill).
	for i := 0; i < 10; i++ {
		var header [80]byte
		for j := range header {
			header[j] = byte(j + i)
		}
		out := h.PoWHash(header[:], ws)
		vectors = append(vectors, vector{
			Name:   fmt.Sprintf("header_80byte_seq_%d", i),
			Input:  hex.EncodeToString(header[:]),
			Output: hex.EncodeToString(out[:]),
		})
	}

	// 5. 80-byte headers with only nonce varying (bytes 76-79).
	var baseHeader [80]byte
	for j := range baseHeader {
		baseHeader[j] = byte(j * 3)
	}
	for nonce := uint32(0); nonce < 20; nonce++ {
		header := baseHeader
		header[76] = byte(nonce)
		header[77] = byte(nonce >> 8)
		header[78] = byte(nonce >> 16)
		header[79] = byte(nonce >> 24)
		out := h.PoWHash(header[:], ws)
		vectors = append(vectors, vector{
			Name:   fmt.Sprintf("header_nonce_%d", nonce),
			Input:  hex.EncodeToString(header[:]),
			Output: hex.EncodeToString(out[:]),
		})
	}

	// 6. Random inputs of various sizes.
	sizes := []int{16, 32, 64, 80, 128, 256}
	for _, size := range sizes {
		input := make([]byte, size)
		rand.Read(input)
		out := h.PoWHash(input, ws)
		vectors = append(vectors, vector{
			Name:   fmt.Sprintf("random_%d_bytes", size),
			Input:  hex.EncodeToString(input),
			Output: hex.EncodeToString(out[:]),
		})
	}

	data, err := json.MarshalIndent(vectors, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "json marshal error: %v\n", err)
		os.Exit(1)
	}
	os.Stdout.Write(data)
	fmt.Fprintln(os.Stdout)

	fmt.Fprintf(os.Stderr, "generated %d test vectors\n", len(vectors))
}
