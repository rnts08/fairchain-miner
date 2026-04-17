package common

import (
	"time"

	"github.com/bams-repo/fairchain/internal/crypto"
	"github.com/bams-repo/fairchain/internal/types"
)

// CalculateSubsidy returns block reward for given chain and height
func CalculateSubsidy(chain string, height uint32) uint64 {
	var initial uint64
	var halving uint32
	switch chain {
	case "testnet":
		initial = 50_0000_00
		halving = 21_000_000
	case "mainnet":
		initial = 50_0000_0000
		halving = 210_000
	default:
		initial = 50_0000_0000
		halving = 150
	}
	halvings := height / halving
	if halvings >= 64 {
		return 0
	}
	return initial >> halvings
}

// MakeCoinbaseTx creates a coinbase transaction
func MakeCoinbaseTx(height uint32, subsidy uint64) types.Transaction {
	pushLen := minimalHeightPushLen(height)
	heightBytes := make([]byte, 4)
	types.PutUint32LE(heightBytes, height)

	msg := make([]byte, 0, 1+pushLen+len("fairchain"))
	msg = append(msg, byte(pushLen))
	msg = append(msg, heightBytes[:pushLen]...)
	msg = append(msg, []byte("fairchain")...)

	return types.Transaction{
		Version: 1,
		Inputs: []types.TxInput{{
			PreviousOutPoint: types.CoinbaseOutPoint,
			SignatureScript:  msg,
			Sequence:         0xFFFFFFFF,
		}},
		Outputs: []types.TxOutput{{
			Value:    subsidy,
			PkScript: []byte{0x00},
		}},
		LockTime: 0,
	}
}

// CreateNewBlock creates a new block template for mining
func CreateNewBlock(prevHash types.Hash, height uint32, bits uint32, chain string) (*types.Block, error) {
	subsidy := CalculateSubsidy(chain, height)
	cb := MakeCoinbaseTx(height, subsidy)

	block := &types.Block{
		Header: types.BlockHeader{
			Version:    1,
			PrevBlock:  prevHash,
			Timestamp:  uint32(time.Now().Unix()),
			Bits:       bits,
			Nonce:      0,
		},
		Transactions: []types.Transaction{cb},
	}

	merkle, err := crypto.ComputeMerkleRoot(block.Transactions)
	if err != nil {
		return nil, err
	}
	block.Header.MerkleRoot = merkle

	return block, nil
}

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