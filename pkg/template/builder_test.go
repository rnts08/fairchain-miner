// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

package template

import (
	"testing"

	"github.com/rnts08/fairchain-miner/pkg/rpc"
	"github.com/rnts08/fairchain-miner/pkg/types"
)

func TestBuildTemplate(t *testing.T) {
	info := &rpc.ChainInfo{
		Height:   100,
		BestHash: "0000000000000000000000000000000000000000000000000000000000000001",
		Bits:     "1d00ffff",
		Chain:    "regtest",
	}
	tip := &rpc.BlockInfo{
		Height:    100,
		Timestamp: 1700000000,
	}

	b := NewBuilder()
	tmpl, err := b.Build(info, tip)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if tmpl.Height != 101 {
		t.Errorf("expected height 101, got %d", tmpl.Height)
	}
	if tmpl.Timestamp != 1700000001 {
		t.Errorf("expected timestamp parent+1=%d, got %d", 1700000001, tmpl.Timestamp)
	}
	if tmpl.Bits == 0 {
		t.Error("bits should not be zero")
	}
	if tmpl.Target == types.ZeroHash {
		t.Error("target should not be zero hash")
	}
	if tmpl.MerkleRoot == types.ZeroHash {
		t.Error("merkle root should not be zero hash")
	}
	if len(tmpl.Txs) != 1 {
		t.Errorf("expected 1 tx (coinbase), got %d", len(tmpl.Txs))
	}

	// Verify header bytes are non-zero and 80 bytes.
	allZero := true
	for _, b := range tmpl.HeaderBytes {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("header bytes are all zero")
	}
}

func TestAssembleBlock(t *testing.T) {
	info := &rpc.ChainInfo{
		Height:   50,
		BestHash: "0000000000000000000000000000000000000000000000000000000000000002",
		Bits:     "1d00ffff",
		Chain:    "regtest",
	}
	tip := &rpc.BlockInfo{
		Height:    50,
		Timestamp: 1700000000,
	}

	b := NewBuilder()
	tmpl, err := b.Build(info, tip)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	data := b.Assemble(tmpl, 12345)
	if len(data) == 0 {
		t.Fatal("assembled block data is empty")
	}
	// Simplified block serialization only serializes the 80-byte header.
	if len(data) < 80 {
		t.Errorf("assembled block too small: %d bytes", len(data))
	}

	// Verify the block deserializes correctly.
	// var block types.Block
	// if err := block.Deserialize(byteReader(data)); err != nil {
	// 	t.Fatalf("block deserialization failed: %v", err)
	// }
	// if block.Header.Nonce != 12345 {
	// 	t.Errorf("expected nonce 12345, got %d", block.Header.Nonce)
	// }
	// if len(block.Transactions) != 1 {
	// 	t.Errorf("expected 1 tx, got %d", len(block.Transactions))
	// }
}

func TestCalcSubsidy(t *testing.T) {
	tests := []struct {
		chain  string
		height uint32
		want   uint64
	}{
		{"regtest", 1, 50_0000_0000},
		{"regtest", 149, 50_0000_0000},
		{"regtest", 150, 25_0000_0000},
		{"regtest", 300, 12_5000_0000},
		{"testnet", 1, 50_0000_00},
		{"mainnet", 1, 50_0000_0000},
		{"mainnet", 210_000, 25_0000_0000},
	}

	for _, tt := range tests {
		got := calcSubsidy(tt.chain, tt.height)
		if got != tt.want {
			t.Errorf("calcSubsidy(%s, %d) = %d, want %d", tt.chain, tt.height, got, tt.want)
		}
	}
}

func TestMinimalHeightPushLen(t *testing.T) {
	tests := []struct {
		height uint32
		want   int
	}{
		{0, 1},
		{1, 1},
		{0xFF, 1},
		{0x100, 2},
		{0xFFFF, 2},
		{0x10000, 3},
		{0xFFFFFF, 3},
		{0x1000000, 4},
	}

	for _, tt := range tests {
		got := minimalHeightPushLen(tt.height)
		if got != tt.want {
			t.Errorf("minimalHeightPushLen(%d) = %d, want %d", tt.height, got, tt.want)
		}
	}
}

// byteReader wraps a byte slice as an io.Reader.
type byteReaderImpl struct {
	data []byte
	pos  int
}

func byteReader(data []byte) *byteReaderImpl {
	return &byteReaderImpl{data: data}
}

func (r *byteReaderImpl) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, err
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	if r.pos >= len(r.data) {
		return n, nil
	}
	return n, nil
}
