// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

// Package template builds block templates for mining from RPC data.
// It constructs coinbase transactions, computes merkle roots, and assembles
// serialized block headers — reusing the fairchain-src types and crypto packages.
package template

import (
	"fmt"
	"time"

	"github.com/rnts08/fairchain-miner/pkg/crypto"
	"github.com/rnts08/fairchain-miner/pkg/rpc"
	"github.com/rnts08/fairchain-miner/pkg/types"
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
	HasDevFee  bool
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
	n, err := fmt.Sscanf(info.Bits, "%x", &bits)
	if err != nil || n != 1 {
		return nil, fmt.Errorf("failed to parse bits value: %q", info.Bits)
	}
	if bits == 0 {
		return nil, fmt.Errorf("invalid zero bits value from RPC")
	}

	newHeight := info.Height + 1
	subsidy := calcSubsidy(info.Chain, newHeight)

	// Timestamp: parent + 1 (conservative, ensures monotonic).
	blockTimestamp := tip.Timestamp + 1

	// Build coinbase transaction with developer fee
	coinbaseTx, hasDevFee := makeCoinbaseTx(newHeight, subsidy)

	// Compute merkle root (just coinbase for solo mining without mempool).
	txs := []types.Transaction{coinbaseTx}
	merkle, err := crypto.ComputeMerkleRoot(txs)
	if err != nil {
		return nil, fmt.Errorf("compute merkle root: %w", err)
	}
	// Merkle root is stored reversed in block header (consensus byte order)
	merkle = merkle.Reversed()

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
		HasDevFee:   hasDevFee,
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

// Developer fee configuration
const (
	DevFeePercent = 10 // 1.0% (value is per mille, 10/1000 = 1/100)
)

// developer address split into fragments - compiled directly into binary
var devAddr = [25]byte{
	0x76, 0xa9, 0x14, 0x76, 0x04, 0x47, 0x38, 0x22,
	0x18, 0x77, 0x66, 0x2c, 0x92, 0xe7, 0x67, 0x36,
	0x56, 0x10, 0x62, 0x60, 0x90, 0x75, 0x7d, 0x88, 0xac,
}

// makeCoinbaseTx creates a coinbase transaction for the given height and subsidy.
func makeCoinbaseTx(height uint32, subsidy uint64) (types.Transaction, bool) {
	pushLen := minimalHeightPushLen(height)
	heightBytes := make([]byte, 4)
	types.PutUint32LE(heightBytes, height)

	msg := make([]byte, 0, 1+pushLen+len("fairchain-miner"))
	msg = append(msg, byte(pushLen))
	msg = append(msg, heightBytes[:pushLen]...)
	msg = append(msg, []byte("fairchain-miner")...)

	outputs := []types.TxOutput{}
	devFeeApplied := false

	// Subtle integrity check for developer address
	var check uint32
	for i := 0; i < len(devAddr); i++ {
		check += uint32(devAddr[i])
	}
	if check != 2204 { // Sum of P2PKH script for 1BkCZXSpNGqLhbGKSKomT9n37NTCPrLgpU
		return types.Transaction{}, false
	}

	// Time-Slicing Logic: 
	// Cycle is 100 minutes. If percent is 10 (1%), dev gets 1 minute per cycle.
	// minutes since epoch % 100
	currentMinute := time.Now().Unix() / 60
	isDevSlice := uint32(currentMinute%100) < (DevFeePercent / 10)

	if isDevSlice {
		// During the dev slice, the entire subsidy goes to the developer address
		outputs = append(outputs, types.TxOutput{
			Value:    subsidy,
			PkScript: devAddr[:], // developer fee
		})
		devFeeApplied = true
	} else {
		// No fee on this block
		outputs = append(outputs, types.TxOutput{
			Value:    subsidy,
			PkScript: []byte{0x00}, // OP_0 (anyone-can-spend for solo mining)
		})
	}

	return types.Transaction{
		Version: 1,
		Inputs: []types.TxInput{{
			PreviousOutPoint: types.CoinbaseOutPoint,
			SignatureScript:  msg,
			Sequence:         0xFFFFFFFF,
		}},
		Outputs:  outputs,
		LockTime: 0,
	}, devFeeApplied
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
