package types

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
)

// Hash represents a 32-byte sha256 hash
type Hash [32]byte

// BlockHeaderSize is the size of the serialized block header in bytes
const BlockHeaderSize = 80

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

// ZeroHash is a hash with all bytes set to zero
var ZeroHash Hash

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

// Reversed returns a new Hash with reversed bytes
func (h Hash) Reversed() Hash {
	var rev Hash
	for i := 0; i < 32; i++ {
		rev[i] = h[31-i]
	}
	return rev
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

// SerializeToBytes serializes the block header into an existing buffer
func (h *BlockHeader) SerializeInto(buf []byte) {
	binary.LittleEndian.PutUint32(buf[0:4], h.Version)
	copy(buf[4:36], h.PrevBlock[:])
	copy(buf[36:68], h.MerkleRoot[:])
	binary.LittleEndian.PutUint32(buf[68:72], h.Timestamp)
	binary.LittleEndian.PutUint32(buf[72:76], h.Bits)
	binary.LittleEndian.PutUint32(buf[76:80], h.Nonce)
}

// SerializeToBytes serializes the transaction
func (tx *Transaction) SerializeToBytes() []byte {
	buf := make([]byte, 0, 1024)
	v := make([]byte, 4)
	binary.LittleEndian.PutUint32(v, tx.Version)
	buf = append(buf, v...)

	buf = append(buf, VarInt(uint64(len(tx.Inputs)))...)
	for _, in := range tx.Inputs {
		buf = append(buf, in.PreviousOutPoint.Hash[:]...)
		idx := make([]byte, 4)
		binary.LittleEndian.PutUint32(idx, in.PreviousOutPoint.Index)
		buf = append(buf, idx...)
		buf = append(buf, VarInt(uint64(len(in.SignatureScript)))...)
		buf = append(buf, in.SignatureScript...)
		seq := make([]byte, 4)
		binary.LittleEndian.PutUint32(seq, in.Sequence)
		buf = append(buf, seq...)
	}

	buf = append(buf, VarInt(uint64(len(tx.Outputs)))...)
	for _, out := range tx.Outputs {
		val := make([]byte, 8)
		binary.LittleEndian.PutUint64(val, out.Value)
		buf = append(buf, val...)
		buf = append(buf, VarInt(uint64(len(out.PkScript)))...)
		buf = append(buf, out.PkScript...)
	}

	lt := make([]byte, 4)
	binary.LittleEndian.PutUint32(lt, tx.LockTime)
	buf = append(buf, lt...)

	return buf
}

// SerializeToBytes serializes the block
func (b *Block) SerializeToBytes() ([]byte, error) {
	header := b.Header.SerializeToBytes()
	buf := append([]byte{}, header...)

	buf = append(buf, VarInt(uint64(len(b.Transactions)))...)
	for _, tx := range b.Transactions {
		buf = append(buf, tx.SerializeToBytes()...)
	}

	return buf, nil
}

// VarInt returns the variable length integer encoding
func VarInt(v uint64) []byte {
	if v < 0xfd {
		return []byte{byte(v)}
	}
	if v <= 0xffff {
		buf := make([]byte, 3)
		buf[0] = 0xfd
		binary.LittleEndian.PutUint16(buf[1:], uint16(v))
		return buf
	}
	if v <= 0xffffffff {
		buf := make([]byte, 5)
		buf[0] = 0xfe
		binary.LittleEndian.PutUint32(buf[1:], uint32(v))
		return buf
	}
	buf := make([]byte, 9)
	buf[0] = 0xff
	binary.LittleEndian.PutUint64(buf[1:], v)
	return buf
}