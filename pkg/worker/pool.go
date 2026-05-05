// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

// Package worker manages mining worker goroutines with nonce partitioning,
// CPU affinity, and power throttling.
package worker

import (
	"context"
	"crypto/sha256"
	"encoding"
	"encoding/binary"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rnts08/fairchain-miner/pkg/algorithm"
	"github.com/rnts08/fairchain-miner/pkg/memory"
	"github.com/rnts08/fairchain-miner/pkg/metrics"
	"github.com/rnts08/fairchain-miner/pkg/template"
	"github.com/rnts08/fairchain-miner/pkg/types"
)

// MineResult holds the outcome of a mining attempt.
type MineResult struct {
	Found  bool
	Nonce  uint32
	Hashes uint64
}

// Pool manages a set of mining worker goroutines.
type Pool struct {
	numWorkers int
	powerLimit int // 1-100
	trackers   []*metrics.HashrateTracker
}

// NewPool creates a new worker pool.
func NewPool(numWorkers, powerLimit int) *Pool {
	if numWorkers < 1 {
		numWorkers = 1
	}
	if powerLimit < 1 {
		powerLimit = 1
	}
	if powerLimit > 100 {
		powerLimit = 100
	}
	trackers := make([]*metrics.HashrateTracker, numWorkers)
	for i := range trackers {
		trackers[i] = metrics.NewHashrateTracker()
	}
	return &Pool{
		numWorkers: numWorkers,
		powerLimit: powerLimit,
		trackers:   trackers,
	}
}

func (p *Pool) WorkerRates() []float64 {
	rates := make([]float64, len(p.trackers))
	for i, t := range p.trackers {
		rates[i] = t.Rate()
	}
	return rates
}

// Mine searches for a valid PoW nonce across all workers.
// staleCheck is called periodically to detect when the chain tip has changed.
func (p *Pool) Mine(ctx context.Context, hasher *algorithm.Hasher, tmpl *template.BlockTemplate, tracker *metrics.HashrateTracker, staleCheck func() bool) MineResult {
	numWorkers := p.numWorkers
	rangeSize := uint64(0x100000000) / uint64(numWorkers)

	type result struct {
		nonce uint32
	}

	mineCtx, mineCancel := context.WithCancel(ctx)
	defer mineCancel()

	resultCh := make(chan result, 1)
	var hashCount atomic.Uint64
	var wg sync.WaitGroup

	// Stale-tip detector.
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-mineCtx.Done():
				return
			case <-ticker.C:
				if staleCheck != nil && staleCheck() {
					mineCancel()
					return
				}
			}
		}
	}()

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		startNonce := uint64(w) * rangeSize
		endNonce := startNonce + rangeSize
		if w == numWorkers-1 {
			endNonce = 0x100000000
		}

		go func(workerID int, sn, en uint64) {
			defer wg.Done()

			// Each worker gets its own NUMA-aware workspace.
			node := memory.GetNodeForCPU(workerID)
			ws := algorithm.NewWorkspaceOnNode(node)

			lastAffinity := IsAffinityEnabled() // Capture initial state
			if lastAffinity {
				_ = SetAffinity(workerID)
			}

			// P5.3: Set NUMA-aware allocation.
			lastNuma := memory.IsNumaEnabled()
			lastHuge := memory.IsHugepagesEnabled()

			// Each worker gets its own copy of the header bytes to stamp nonces into.
			var headerBuf [types.BlockHeaderSize]byte
			copy(headerBuf[:], tmpl.HeaderBytes[:])

			// Precompute SHA-256 midstate for the first 64 bytes of the header (P2.5)
			// The first 64 bytes are constant across all nonces in this template.
			midHasher := sha256.New()
			midHasher.Write(headerBuf[:64])
			midState, _ := midHasher.(encoding.BinaryMarshaler).MarshalBinary()

			for pos := sn; pos < en; pos++ {
				select {
				case <-mineCtx.Done():
					return
				default:
				}

				// Check for hardware setting changes (NUMA/Hugepages/Affinity)
				if pos%1024 == 0 {
					if IsAffinityEnabled() != lastAffinity {
						// Affinity changed, re-apply
						if IsAffinityEnabled() {
							_ = SetAffinity(workerID)
						} else {
							_ = UnsetAffinity()
						}
						lastAffinity = IsAffinityEnabled()
					}

					// Memory settings changed, re-allocate workspace
					if memory.IsNumaEnabled() != lastNuma || memory.IsHugepagesEnabled() != lastHuge {
						ws.Free()
						lastNuma = memory.IsNumaEnabled()
						lastHuge = memory.IsHugepagesEnabled()
						ws = algorithm.NewWorkspaceOnNode(node)
					}
				}

				batchStart := time.Now()

				// Stamp nonce into header buffer (bytes 76-79, little-endian).
				nonce := uint32(pos)
				binary.LittleEndian.PutUint32(headerBuf[76:80], nonce)

				// Compute PoW hash using midstate optimization.
				h := hasher.PoWHashMidstate(headerBuf[:], ws, midState)
				hashCount.Add(1)
				tracker.Add(1)

				// Check against target.
				if h.LessOrEqual(tmpl.Target) {
					select {
					case resultCh <- result{nonce: nonce}:
					default:
					}
					mineCancel()
					return
				}

				// Power limit throttling.
				currPower := GetGlobalPowerLimit()
				if currPower < 100 {
					elapsed := time.Since(batchStart)
					sleepRatio := float64(100-currPower) / float64(currPower)
					time.Sleep(time.Duration(float64(elapsed) * sleepRatio))
				}
			}
		}(w, startNonce, endNonce)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	res, ok := <-resultCh
	mineCancel()
	wg.Wait()

	if !ok {
		return MineResult{Found: false, Hashes: hashCount.Load()}
	}
	return MineResult{Found: true, Nonce: res.nonce, Hashes: hashCount.Load()}
}

// RunBenchmark runs the hasher in a tight loop for benchmarking (no target check).
func (p *Pool) RunBenchmark(ctx context.Context, hasher *algorithm.Hasher, input []byte, tracker *metrics.HashrateTracker) {
	var wg sync.WaitGroup

	for w := 0; w < p.numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// Each worker gets its own NUMA-aware workspace.
			_ = SetAffinity(workerID)
			node := memory.GetNodeForCPU(workerID)
			ws := algorithm.NewWorkspaceOnNode(node)

			// Each worker hashes with incrementing nonce-like data.
			var buf [80]byte
			copy(buf[:], input)

			var nonce uint32
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				binary.LittleEndian.PutUint32(buf[76:80], nonce)
				hasher.PoWHash(buf[:], ws)
				tracker.Add(1)
				nonce++

				// Power limit throttling.
				if p.powerLimit < 100 {
					// For benchmark, approximate: sleep every 10 hashes.
					if nonce%10 == 0 {
						sleepRatio := float64(100-p.powerLimit) / float64(p.powerLimit)
						time.Sleep(time.Duration(float64(50*time.Millisecond) * sleepRatio))
					}
				}
			}
		}(w)
	}

	wg.Wait()
}
