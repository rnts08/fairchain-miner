package algorithms

import (
	"crypto/sha256"
	"errors"

	"github.com/bams-repo/fairchain/internal/types"
)

// Hasher interface for PoW functions
type Hasher interface {
	PoWHash([]byte) types.Hash
	Name() string
}

type sha256dHasher struct{}

func (h sha256dHasher) PoWHash(b []byte) types.Hash {
	first := sha256.Sum256(b)
	return sha256.Sum256(first[:])
}

func (h sha256dHasher) Name() string {
	return "sha256d"
}

// GetHasher returns the appropriate hasher for algorithm name
func GetHasher(name string) (Hasher, error) {
	switch name {
	case "sha256d":
		return sha256dHasher{}, nil
	case "sha256mem":
		return sha256dHasher{}, nil
	default:
		return nil, errors.New("unknown algorithm")
	}
}