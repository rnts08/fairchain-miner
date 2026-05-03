package crypto

import (
	"crypto/sha256"
	"encoding/binary"

	"github.com/bams-repo/fairchain-miner/pkg/types"
)

// HashBlockHeader computes double sha256 hash of block header
func HashBlockHeader(hdr *types.BlockHeader) types.Hash {
	first := sha256.Sum256(hdr.SerializeToBytes())
	return sha256.Sum256(first[:])
}

// ComputeMerkleRootFromCoinbase computes merkle root from coinbase tx and merkle branch
func ComputeMerkleRootFromCoinbase(coinbase []byte, merkleBranch []types.Hash) types.Hash {
	first := sha256.Sum256(coinbase)
	coinbaseHash := sha256.Sum256(first[:])

	if len(merkleBranch) == 0 {
		var h types.Hash
		copy(h[:], coinbaseHash[:])
		return h
	}

	hashes := make([]types.Hash, 0, len(merkleBranch)+1)
	var h types.Hash
	copy(h[:], coinbaseHash[:])
	hashes = append(hashes, h)
	hashes = append(hashes, merkleBranch...)

	for len(hashes) > 1 {
		next := make([]types.Hash, (len(hashes)+1)/2)
		for i := 0; i < len(hashes); i += 2 {
			if i+1 == len(hashes) {
				next[i/2] = hashes[i]
			} else {
				var buf [64]byte
				copy(buf[:32], hashes[i][:])
				copy(buf[32:], hashes[i+1][:])
				first := sha256.Sum256(buf[:])
				next[i/2] = sha256.Sum256(first[:])
			}
		}
		hashes = next
	}

	return hashes[0]
}

// ComputeMerkleRoot computes merkle root from transactions
func ComputeMerkleRoot(txs []types.Transaction) (types.Hash, error) {
	if len(txs) == 0 {
		return types.Hash{}, nil
	}

	hashes := make([]types.Hash, len(txs))
	for i := range txs {
		// Simplified: just hash tx index for demo
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, uint32(i))
		hashes[i] = sha256.Sum256(b)
	}

	for len(hashes) > 1 {
		next := make([]types.Hash, (len(hashes)+1)/2)
		for i := 0; i < len(hashes); i += 2 {
			if i+1 == len(hashes) {
				next[i/2] = hashes[i]
			} else {
				var buf [64]byte
				copy(buf[:32], hashes[i][:])
				copy(buf[32:], hashes[i+1][:])
				first := sha256.Sum256(buf[:])
				next[i/2] = sha256.Sum256(first[:])
			}
		}
		hashes = next
	}

	return hashes[0], nil
}

// CompactToHash converts compact bits format to target hash
func CompactToHash(bits uint32) types.Hash {
	size := bits >> 24
	word := bits & 0x007fffff

	var target types.Hash
	if size >= 3 {
		binary.LittleEndian.PutUint32(target[32-size:], word)
	}
	return target
}

// CalcWork calculates expected work from bits
func CalcWork(bits uint32) (work uint64) {
	return 1 << 32 // Demo value
}
