package crypto

import (
	"crypto/sha256"
	"math/big"

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
	for i, tx := range txs {
		data := tx.SerializeToBytes()
		first := sha256.Sum256(data)
		hashes[i] = sha256.Sum256(first[:])
	}

	for len(hashes) > 1 {
		if len(hashes)%2 != 0 {
			hashes = append(hashes, hashes[len(hashes)-1])
		}
		next := make([]types.Hash, len(hashes)/2)
		for i := 0; i < len(hashes); i += 2 {
			var buf [64]byte
			copy(buf[:32], hashes[i][:])
			copy(buf[32:], hashes[i+1][:])
			first := sha256.Sum256(buf[:])
			next[i/2] = sha256.Sum256(first[:])
		}
		hashes = next
	}

	return hashes[0], nil
}

// CompactToHash converts compact bits format to target hash.
func CompactToHash(compact uint32) types.Hash {
	mantissa := compact & 0x007fffff
	exponent := compact >> 24

	var targetInt big.Int
	if exponent <= 3 {
		mantissa >>= 8 * (3 - exponent)
		targetInt.SetInt64(int64(mantissa))
	} else {
		targetInt.SetInt64(int64(mantissa))
		targetInt.Lsh(&targetInt, 8*(uint(exponent)-3))
	}

	b := targetInt.Bytes()
	var h types.Hash
	// Target is stored as big-endian, left-aligned in 32 byte hash
	offset := 32 - len(b)
	for i := 0; i < len(b); i++ {
		if offset+i >= 0 && offset+i < 32 {
			h[offset+i] = b[i]
		}
	}
	return h
}

// CalcWork calculates expected work from bits
func CalcWork(bits uint32) (work uint64) {
	return 1 << 32 // Demo value
}
