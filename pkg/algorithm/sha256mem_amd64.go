//go:build amd64

package algorithm

import (
	"encoding/binary"
	"unsafe"

	"github.com/rnts08/fairchain-miner/pkg/types"
)

// sha256CompressSHA_NI is implemented in sha256mem_amd64.s
//
//go:noescape
func sha256CompressSHA_NI(state *[8]uint32, block *[64]byte)

// prefetcht0 is implemented in sha256mem_amd64.s
//
//go:noescape
func prefetcht0(addr uintptr)

type amd64Hasher struct {
	optHasher
	useSHA bool
}

func newOptimizedHasher() Hasher {
	return &amd64Hasher{
		optHasher: *NewOptHasher(),
		useSHA:    hasSHANI,
	}
}

func (h *amd64Hasher) Name() string {
	if h.useSHA {
		return "sha256mem-shani"
	}
	return "sha256mem-amd64"
}

func (h *amd64Hasher) PoWHash(data []byte, ws *Workspace) types.Hash {
	return h.PoWHashMidstate(data, ws, nil)
}

func (h *amd64Hasher) PoWHashMidstate(data []byte, ws *Workspace, midstate []byte) types.Hash {
	// Phase 1: Seed
	var seed [32]byte
	if midstate != nil && len(data) == 80 {
		seed = ws.sum256Midstate(data, midstate)
	} else {
		seed = ws.sum256(data)
	}

	mem := *ws.Mem
	mem[0] = seed

	// Phase 2: Memory fill
	for i := uint32(1); i < Slots; i++ {
		if i%HardenInterval == 0 {
			if h.useSHA {
				state := (*[8]uint32)(unsafe.Pointer(&mem[i][0]))
				copy(mem[i][:], mem[i-1][:])
				// Hardware-accelerated compression
				// Create a padded 64-byte block for SHA-256
				var block [64]byte
				copy(block[:32], mem[i-1][:])
				block[32] = 0x80
				binary.BigEndian.PutUint64(block[56:], 256)
				sha256CompressSHA_NI(state, &block)
			} else {
				mem[i] = ws.sum256(mem[i-1][:])
			}
		} else {
			arxFillGeneric(&mem[i], &mem[i-1], i)
		}
	}

	// Phase 3 & 4: Mixing with prefetching
	acc := mem[Slots-1]
	copy(ws.mixBuf[:32], acc[:])

	for i := 0; i < MixRounds*2; i++ {
		var off uint32
		if i >= MixRounds {
			off = uint32((i % 7) * 4)
		}

		idx := binary.LittleEndian.Uint32(ws.mixBuf[off:off+4]) % uint32(Slots)
		prefetcht0(uintptr(unsafe.Pointer(&mem[idx])))
		copy(ws.mixBuf[32:], mem[idx][:])

		if h.useSHA {
			state := (*[8]uint32)(unsafe.Pointer(&ws.mixBuf[0]))
			var block [64]byte
			copy(block[:], ws.mixBuf[:])
			sha256CompressSHA_NI(state, &block)
		} else {
			ws.Hasher.Reset()
			ws.Hasher.Write(ws.mixBuf[:])
			ws.Hasher.Sum(ws.mixBuf[:0])
		}
	}

	final := ws.sum256(ws.mixBuf[:32])
	return types.Hash(final).Reversed()
}
