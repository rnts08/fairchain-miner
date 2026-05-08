package algorithm

import (
	"crypto/sha256"
	"hash"
	"unsafe"
)

// Constants for the sha256mem algorithm.
// Note: These should match pkg/algorithm/sha256mem.go
const (
	Slots          = 2097152
	HardenInterval = 128
	MixRounds      = 32768
	ScratchpadSize = Slots * 32
)

// Hasher maintains the state for a single mining worker to avoid allocations in the hot path.
// It implements Tier 1 Pure Go micro-optimizations.
type Hasher struct {
	scratchpad []byte // 64 MiB pre-allocated scratchpad
	h          hash.Hash
	header     [80]byte // Reusable header buffer for nonce updates
}

// NewHasher allocates a new Hasher with a dedicated scratchpad.
func NewHasher() *Hasher {
	return &Hasher{
		scratchpad: make([]byte, ScratchpadSize),
		h:          sha256.New(),
	}
}

// PoWHash calculates the sha256mem hash for the given header.
// This implementation uses Tier 1 optimizations: pointer arithmetic,
// bounds check elimination, and state reuse.
func (ctx *Hasher) PoWHash(header []byte) [32]byte {
	// Phase 1: Seed
	ctx.h.Reset()
	ctx.h.Write(header)
	// Write seed directly into the first slot of scratchpad
	ctx.h.Sum(ctx.scratchpad[:0])

	// Use unsafe pointers to treat the scratchpad as an array of [8]uint32
	// This avoids binary.LittleEndian overhead and enables manual bounds check elimination.
	memBase := (*[Slots][8]uint32)(unsafe.Pointer(&ctx.scratchpad[0]))

	// Phase 2: Memory Fill
	for i := uint32(1); i < Slots; i++ {
		if i%HardenInterval == 0 {
			// Serial hardening
			ctx.h.Reset()
			// Use the previous 32 bytes as input
			ctx.h.Write(ctx.scratchpad[(i-1)*32 : i*32])
			// Write result directly into current slot
			ctx.h.Sum(ctx.scratchpad[i*32 : i*32 : (i+1)*32])
		} else {
			// Fast ARX fill
			// Manual pointer access to bypass Go's bounds checks
			src := &memBase[i-1]
			dst := &memBase[i]
			
			// ARX logic inlined for words 0..7
			// v = src ^ (index + word_idx); v = ROTL(v, 13); v += src
			for w := uint32(0); w < 8; w++ {
				v := src[w]
				v ^= (i + w)
				v = (v << 13) | (v >> 19) // ROTL 13
				v += src[w]
				dst[w] = v
			}
		}
	}

	// Phase 3 & 4: Mix Passes
	var acc [32]byte
	copy(acc[:], ctx.scratchpad[(Slots-1)*32:])
	
	// Reusable 64-byte buffer for acc || mem[idx]
	var mixBuf [64]byte

	// Pass A: Data-dependent indexing
	for i := 0; i < MixRounds; i++ {
		// LE32(acc[0:4])
		idx := *(*uint32)(unsafe.Pointer(&acc[0])) % Slots
		
		copy(mixBuf[0:32], acc[:])
		copy(mixBuf[32:64], ctx.scratchpad[idx*32:(idx+1)*32])
		
		ctx.h.Reset()
		ctx.h.Write(mixBuf[:])
		// Use acc[:] to store the sum directly, avoiding allocations
		ctx.h.Sum(acc[:0]) 
	}

	// Pass B: Rotating offset indexing
	for i := 0; i < MixRounds; i++ {
		off := (i % 7) * 4 // Cycles through offsets 0, 4, 8, 12, 16, 20, 24
		idx := *(*uint32)(unsafe.Pointer(&acc[off])) % Slots
		
		copy(mixBuf[0:32], acc[:])
		copy(mixBuf[32:64], ctx.scratchpad[idx*32:(idx+1)*32])
		
		ctx.h.Reset()
		ctx.h.Write(mixBuf[:])
		ctx.h.Sum(acc[:0])
	}

	// Phase 5: Finalize
	ctx.h.Reset()
	ctx.h.Write(acc[:])
	final := ctx.h.Sum(nil)
	
	// Reverse byte order (LE internal -> PoW result order)
	var result [32]byte
	for i := 0; i < 32; i++ {
		result[i] = final[31-i]
	}
	return result
}

// HashWithNonce implements batch nonce serialization.
func (ctx *Hasher) HashWithNonce(baseHeader []byte, nonce uint32) [32]byte {
	if len(ctx.header) == 0 {
		copy(ctx.header[:], baseHeader)
	}
	// Only update the last 4 bytes (standard PoW nonce position)
	*(*uint32)(unsafe.Pointer(&ctx.header[76])) = nonce
	return ctx.PoWHash(ctx.header[:])
}