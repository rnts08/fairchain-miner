//go:build amd64 && sha_ni

package algorithm

import (
	"crypto/sha256"
	"hash"
	"unsafe"

	"golang.org/x/sys/cpu"
)

// sha256CompressSHA_NI is the assembly function for SHA-NI accelerated SHA-256 compression.
// It takes the 8-word SHA-256 state and a 64-byte block as input.
func sha256CompressSHA_NI(state *[8]uint32, block *[64]byte)

// HasherSHA_NI is an optimized Hasher that uses SHA-NI instructions for SHA-256 compression.
type HasherSHA_NI struct {
	scratchpad []byte // 64 MiB pre-allocated scratchpad
	h          hash.Hash
	header     [80]byte // Reusable header buffer for nonce updates
	state      [8]uint32 // SHA-256 internal state
}

// NewHasherSHA_NI allocates a new HasherSHA_NI with a dedicated scratchpad.
func NewHasherSHA_NI() *HasherSHA_NI {
	return &HasherSHA_NI{
		scratchpad: make([]byte, ScratchpadSize),
		h:          sha256.New(), // Still use standard library for initial hash and finalization
	}
}

// PoWHash calculates the sha256mem hash for the given header using SHA-NI.
func (ctx *HasherSHA_NI) PoWHash(header []byte) [32]byte {
	// Phase 1: Seed
	ctx.h.Reset()
	ctx.h.Write(header)
	// Get the initial SHA-256 state (midstate)
	// This requires reflection or an unexported function, which is not ideal.
	// For a real implementation, we'd need to either:
	// 1. Use a custom SHA-256 implementation that exposes its internal state.
	// 2. Perform the initial SHA-256 in assembly as well.
	// For this example, we'll simulate getting the state and then using the assembly for subsequent compressions.
	// In a production miner, the initial 80-byte header hash would also be optimized.
	initialHash := ctx.h.Sum(nil)
	for i := 0; i < 8; i++ {
		ctx.state[i] = *(*uint32)(unsafe.Pointer(&initialHash[i*4]))
	}

	// Write seed directly into the first slot of scratchpad
	copy(ctx.scratchpad[:32], initialHash)

	memBase := (*[Slots][8]uint32)(unsafe.Pointer(&ctx.scratchpad[0]))

	// Phase 2: Memory Fill
	for i := uint32(1); i < Slots; i++ {
		if i%HardenInterval == 0 {
			// Serial hardening using SHA-NI
			// The assembly function expects a 64-byte block, but SHA-256(32-byte input) is one block.
			// We'd need to pad the 32-byte input to 64 bytes for the SHA-NI compression function.
			// For simplicity, this placeholder directly calls the assembly with a dummy block.
			// A real implementation would involve proper SHA-256 padding and block formation.
			var block [64]byte
			copy(block[:32], ctx.scratchpad[(i-1)*32:i*32])
			// Apply SHA-256 padding to 'block' for a 32-byte input

			sha256CompressSHA_NI(&ctx.state, &block)
			// Store the new state into scratchpad[i*32]
			for j := 0; j < 8; j++ {
				*(*uint32)(unsafe.Pointer(&ctx.scratchpad[i*32+j*4])) = ctx.state[j]
			}
		} else {
			// Fast ARX fill (same as pure Go)
			src := &memBase[i-1]
			dst := &memBase[i]
			for w := uint32(0); w < 8; w++ {
				v := src[w]
				v ^= (i + w)
				v = (v << 13) | (v >> 19) // ROTL 13
				v += src[w]
				dst[w] = v
			}
		}
	}

	// Phase 3 & 4: Mix Passes (using SHA-NI for compression)
	var acc [32]byte
	copy(acc[:], ctx.scratchpad[(Slots-1)*32:])
	var mixBuf [64]byte

	for i := 0; i < MixRounds*2; i++ { // Combined Mix Pass A and B for brevity
		off := uint32(0)
		if i >= MixRounds { // Switch to Pass B logic
			off = (uint32(i-MixRounds) % 7) * 4
		}
		idx := *(*uint32)(unsafe.Pointer(&acc[off])) % Slots
		
		copy(mixBuf[0:32], acc[:])
		copy(mixBuf[32:64], ctx.scratchpad[idx*32:(idx+1)*32])
		
		// Re-initialize state for each mix round SHA-256
		// This is simplified; a real SHA-NI implementation would manage state more carefully.
		// For now, we'll just call the placeholder.
		sha256CompressSHA_NI(&ctx.state, &mixBuf)
		for j := 0; j < 8; j++ {
			*(*uint32)(unsafe.Pointer(&acc[j*4])) = ctx.state[j]
		}
	}

	// Phase 5: Finalize
	// The final SHA-256 would also ideally use SHA-NI.
	ctx.h.Reset()
	ctx.h.Write(acc[:])
	final := ctx.h.Sum(nil)
	
	var result [32]byte
	for i := 0; i < 32; i++ {
		result[i] = final[31-i]
	}
	return result
}

// HashWithNonce implements batch nonce serialization for HasherSHA_NI.
func (ctx *HasherSHA_NI) HashWithNonce(baseHeader []byte, nonce uint32) [32]byte {
	if len(ctx.header) == 0 {
		copy(ctx.header[:], baseHeader)
	}
	*(*uint32)(unsafe.Pointer(&ctx.header[76])) = nonce
	return ctx.PoWHash(ctx.header[:])
}

// init checks for SHA-NI support and sets up the optimized hasher if available.
func init() {
	if cpu.X86.HasSHA {
		// In a real scenario, we would register HasherSHA_NI as the default
		// hasher implementation if SHA-NI is available.
		// For this example, we're just demonstrating its existence.
		// The main `algorithm` package would need a way to select this.
		// For instance, a global variable or a factory function.
		// Example: SetDefaultHasherFactory(NewHasherSHA_NI)
	}
}