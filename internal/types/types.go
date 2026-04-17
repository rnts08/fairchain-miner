package types

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
)

// Hash represents a 32-byte sha256 hash
type Hash [32]byte

// BlockHeader is the block header structure
type BlockHeader struct {
	Version    uint32
	PrevBlock  Hash
	MerkleRoot Hash
	Timestamp  uint32
	Bits       uint32
	Nonce      uint32
}

// Transaction represents a Bitcoin-style transaction
type Transaction struct {
	Version  uint32
	Inputs   []TxInput
	Outputs  []TxOutput
	LockTime uint32
}

// TxInput is a transaction input
type TxInput struct {
	PreviousOutPoint OutPoint
	SignatureScript  []byte
	Sequence         uint32
}

// TxOutput is a transaction output
type TxOutput struct {
	Value    uint64
	PkScript []byte
}

// OutPoint references a transaction output
type OutPoint struct {
	Hash  Hash
	Index uint32
}

// Block represents a full block
type Block struct {
	Header       BlockHeader
	Transactions []Transaction
}

// CoinbaseOutPoint is the special outpoint for coinbase inputs
var CoinbaseOutPoint = OutPoint{
	Hash:  Hash{},
	Index: 0xffffffff,
}

// PutUint32LE writes a uint32 in little endian
func PutUint32LE(b []byte, v uint32) {
	binary.LittleEndian.PutUint32(b, v)
}

// HashFromReverseHex parses a reversed hex string (Bitcoin RPC format) to Hash
func HashFromReverseHex(s string) (Hash, error) {
	var h Hash
	b, err := hex.DecodeString(s)
	if err != nil {
		return h, err
	}
	if len(b) != 32 {
		return h, errors.New("invalid hash length")
	}
	// reverse bytes
	for i := 0; i < 16; i++ {
		b[i], b[31-i] = b[31-i], b[i]
	}
	copy(h[:], b)
	return h, nil
}

// LessOrEqual returns true if this hash is <= target hash
func (h Hash) LessOrEqual(other Hash) bool {
	for i := 31; i >= 0; i-- {
		if h[i] < other[i] {
			return true
		}
		if h[i] > other[i] {
			return false
		}
	}
	return true
}

// ReverseString returns the hash as reversed hex string
func (h Hash) ReverseString() string {
	var rev [32]byte
	for i := 0; i < 32; i++ {
		rev[i] = h[31-i]
	}
	return hex.EncodeToString(rev[:])
}

// SerializeToBytes serializes the block header
func (h *BlockHeader) SerializeToBytes() []byte {
	buf := make([]byte, 80)
	binary.LittleEndian.PutUint32(buf[0:4], h.Version)
	copy(buf[4:36], h.PrevBlock[:])
	copy(buf[36:68], h.MerkleRoot[:])
	binary.LittleEndian.PutUint32(buf[68:72], h.Timestamp)
	binary.LittleEndian.PutUint32(buf[72:76], h.Bits)
	binary.LittleEndian.PutUint32(buf[76:80], h.Nonce)
	return buf
}

// SerializeToBytes serializes the block
func (b *Block) SerializeToBytes() ([]byte, error) {
	// Simplified serialization for RPC submission
	buf := make([]byte, 80)
	copy(buf, b.Header.SerializeToBytes())
	return buf, nil
}