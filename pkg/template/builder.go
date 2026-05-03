// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

// Package template builds block templates for mining from RPC data.
// It constructs coinbase transactions, computes merkle roots, and assembles
// serialized block headers — reusing the fairchain-src types and crypto packages.
package template

import (
	"fmt"

	"github.com/bams-repo/fairchain-miner/pkg/rpc"
	"github.com/bams-repo/fairchain-miner/pkg/crypto"
	"github.com/bams-repo/fairchain-miner/pkg/types"
)

// BlockTemplate holds everything needed to mine a block.
type BlockTemplate struct {
	Height     uint32
	PrevHash   types.Hash
	Bits       uint32
	Timestamp  uint32
	MerkleRoot types.Hash
	Version    uint32
	Subsidy    uint64

	// Pre-serialized header with nonce=0 for workers to stamp nonces into.
	HeaderBytes [types.BlockHeaderSize]byte

	// Target for PoW comparison.
	Target types.Hash

	// Full block data needed for submission.
	CoinbaseTx types.Transaction
	Txs        []types.Transaction
}

// Builder constructs block templates from RPC data.
type Builder struct{}

// NewBuilder creates a new template builder.
func NewBuilder() *Builder {
	return &Builder{}
}

// Build creates a BlockTemplate from chain info and tip block data.
func (b *Builder) Build(info *rpc.ChainInfo, tip *rpc.BlockInfo) (*BlockTemplate, error) {
	prevHash, err := types.HashFromReverseHex(info.BestHash)
	if err != nil {
		return nil, fmt.Errorf("parse prev hash: %w", err)
	}

	var bits uint32
	fmt.Sscanf(info.Bits, "%x", &bits)

	newHeight := info.Height + 1
	subsidy := calcSubsidy(info.Chain, newHeight)

	// Timestamp: parent + 1 (conservative, ensures monotonic).
	blockTimestamp := tip.Timestamp + 1

	// Build coinbase transaction.
	coinbaseTx := makeCoinbaseTx(newHeight, subsidy)

	// Compute merkle root (just coinbase for solo mining without mempool).
	txs := []types.Transaction{coinbaseTx}
	merkle, err := crypto.ComputeMerkleRoot(txs)
	if err != nil {
		return nil, fmt.Errorf("compute merkle root: %w", err)
	}

	// Build header.
	header := types.BlockHeader{
		Version:    1,
		PrevBlock:  prevHash,
		MerkleRoot: merkle,
		Timestamp:  blockTimestamp,
		Bits:       bits,
		Nonce:      0,
	}

	target := crypto.CompactToHash(bits)

	var headerBytes [types.BlockHeaderSize]byte
	header.SerializeInto(headerBytes[:])

	return &BlockTemplate{
		Height:      newHeight,
		PrevHash:    prevHash,
		Bits:        bits,
		Timestamp:   blockTimestamp,
		MerkleRoot:  merkle,
		Version:     1,
		Subsidy:     subsidy,
		HeaderBytes: headerBytes,
		Target:      target,
		CoinbaseTx:  coinbaseTx,
		Txs:         txs,
	}, nil
}

// Assemble creates the final serialized block bytes with the winning nonce.
func (b *Builder) Assemble(tmpl *BlockTemplate, nonce uint32) []byte {
	header := types.BlockHeader{
		Version:    tmpl.Version,
		PrevBlock:  tmpl.PrevHash,
		MerkleRoot: tmpl.MerkleRoot,
		Timestamp:  tmpl.Timestamp,
		Bits:       tmpl.Bits,
		Nonce:      nonce,
	}

	block := &types.Block{
		Header:       header,
		Transactions: tmpl.Txs,
	}

	data, err := block.SerializeToBytes()
	if err != nil {
		// Should never happen with valid template data.
		panic(fmt.Sprintf("failed to serialize block: %v", err))
	}
	return data
}

// makeCoinbaseTx creates a coinbase transaction for the given height and subsidy.
func makeCoinbaseTx(height uint32, subsidy uint64) types.Transaction {
	pushLen := minimalHeightPushLen(height)
	heightBytes := make([]byte, 4)
	types.PutUint32LE(heightBytes, height)

	msg := make([]byte, 0, 1+pushLen+len("fairchain-miner"))
	msg = append(msg, byte(pushLen))
	msg = append(msg, heightBytes[:pushLen]...)
	msg = append(msg, []byte("fairchain-miner")...)

	return types.Transaction{
		Version: 1,
		Inputs: []types.TxInput{{
			PreviousOutPoint: types.CoinbaseOutPoint,
			SignatureScript:  msg,
			Sequence:         0xFFFFFFFF,
		}},
		Outputs: []types.TxOutput{{
			Value:    subsidy,
			PkScript: []byte{0x00}, // OP_0 (anyone-can-spend for solo mining)
		}},
		LockTime: 0,
	}
}

// minimalHeightPushLen returns the minimal number of bytes to encode height
// in the coinbase scriptSig (BIP34).
func minimalHeightPushLen(height uint32) int {
	switch {
	case height <= 0xFF:
		return 1
	case height <= 0xFFFF:
		return 2
	case height <= 0xFFFFFF:
		return 3
	default:
		return 4
	}
}

// calcSubsidy computes the block reward at the given height.
func calcSubsidy(chain string, height uint32) uint64 {
	var initial uint64
	var halving uint32
	switch chain {
	case "testnet":
		initial = 50_0000_00
		halving = 21_000_000
	case "mainnet":
		initial = 50_0000_0000
		halving = 210_000
	default: // regtest
		initial = 50_0000_0000
		halving = 150
	}
	halvings := height / halving
	if halvings >= 64 {
		return 0
	}
	return initial >> halvings
}
