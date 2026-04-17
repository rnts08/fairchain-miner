// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

package miner

import (
	"context"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bams-repo/fairchain/internal/algorithms"
	"github.com/bams-repo/fairchain/internal/types"
)

// Miner is a standalone proof-of-work miner that searches for valid nonces
// It has no node dependencies, works only with BlockHeader and Target
type Miner struct {
	hasher    algorithms.Hasher
	workers   int
	powerLimit atomic.Int32

	hashCount     atomic.Uint64
	hashrate      atomic.Uint64
	hashrateReady atomic.Bool

	ewmaMu        sync.Mutex
	ewmaRate      float64
	lastSnapCount uint64
	lastSnapTime  time.Time
	snapCount     int
}

// New creates a standalone miner
// numWorkers: 0 = use all available CPUs
func New(h algorithms.Hasher, numWorkers int) *Miner {
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}
	if numWorkers < 1 {
		numWorkers = 1
	}
	m := &Miner{
		hasher:  h,
		workers: numWorkers,
	}
	m.powerLimit.Store(100)
	return m
}

// Hashrate returns EWMA hashrate in hashes per second
func (m *Miner) Hashrate() uint64 {
	return m.hashrate.Load()
}

// HashrateReady returns true once enough samples exist
func (m *Miner) HashrateReady() bool {
	return m.hashrateReady.Load()
}

// Workers returns current worker count
func (m *Miner) Workers() int {
	return m.workers
}

// PowerLimit returns current power limit percentage (1-100)
func (m *Miner) PowerLimit() int {
	return int(m.powerLimit.Load())
}

// SetPowerLimit adjusts CPU throttling
func (m *Miner) SetPowerLimit(pct int) {
	if pct < 1 {
		pct = 1
	}
	if pct > 100 {
		pct = 100
	}
	m.powerLimit.Store(int32(pct))
}

// MaxWorkers returns number of logical CPUs
func MaxWorkers() int {
	n := runtime.NumCPU()
	if n < 1 {
		return 1
	}
	return n
}

// StartHashrateMonitor begins tracking hashrate in background
func (m *Miner) StartHashrateMonitor(ctx context.Context) {
	m.ewmaMu.Lock()
	m.ewmaRate = 0
	m.lastSnapCount = m.hashCount.Load()
	m.lastSnapTime = time.Now()
	m.snapCount = 0
	m.ewmaMu.Unlock()
	m.hashrateReady.Store(false)
	m.hashrate.Store(0)

	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				m.snapshotHashrate()
			}
		}
	}()
}

const ewmaAlpha = 0.049 // ~60s time constant with 3s samples

func (m *Miner) snapshotHashrate() {
	now := time.Now()
	current := m.hashCount.Load()

	m.ewmaMu.Lock()
	dt := now.Sub(m.lastSnapTime).Seconds()
	if dt <= 0 {
		m.ewmaMu.Unlock()
		return
	}

	instantRate := float64(current-m.lastSnapCount) / dt
	m.lastSnapCount = current
	m.lastSnapTime = now
	m.snapCount++

	if m.snapCount == 1 {
		m.ewmaRate = instantRate
	} else {
		m.ewmaRate = ewmaAlpha*instantRate + (1-ewmaAlpha)*m.ewmaRate
	}

	rate := m.ewmaRate
	ready := m.snapCount >= 4
	m.ewmaMu.Unlock()

	m.hashrate.Store(uint64(rate))
	if ready {
		m.hashrateReady.Store(true)
	}
}

// MineHeader searches for a valid nonce for the given header
// Returns found bool, nonce, total hashes performed
// Cancels early if ctx is cancelled
func (m *Miner) MineHeader(ctx context.Context, header types.BlockHeader, target types.Hash) (found bool, nonce uint32, hashes uint64) {
	rangeSize := uint64(0x100000000) / uint64(m.workers)
	batchSize := uint64(4)
	if m.hasher.Name() == "sha256mem" {
		batchSize = 32
	}

	type result struct {
		nonce uint32
	}

	workerCtx, workerCancel := context.WithCancel(ctx)
	defer workerCancel()

	resultCh := make(chan result, 1)
	var wg sync.WaitGroup
	var hashCount atomic.Uint64

	for w := 0; w < m.workers; w++ {
		wg.Add(1)
		startNonce := uint64(w) * rangeSize
		endNonce := startNonce + rangeSize
		if w == m.workers-1 {
			endNonce = 0x100000000
		}

		go func(wHeader types.BlockHeader, start, end uint64) {
			defer wg.Done()
			wHeader.Nonce = uint32(start)
			pos := start

			for pos < end {
				select {
				case <-workerCtx.Done():
					return
				default:
				}

				remaining := end - pos
				batch := batchSize
				if remaining < batch {
					batch = remaining
				}

				batchStart := time.Now()

				for i := uint64(0); i < batch; i++ {
					hdrBytes := wHeader.SerializeToBytes()
					hash := m.hasher.PoWHash(hdrBytes)
					hashCount.Add(1)

					targetHash := types.Hash(target)
	if hash.LessOrEqual(targetHash) {
						select {
						case resultCh <- result{nonce: wHeader.Nonce}:
						default:
						}
						workerCancel()
						return
					}

					wHeader.Nonce++
					if wHeader.Nonce == 0 {
						return
					}
				}

				pos += batch

				// Power limit throttling
				if pct := int(m.powerLimit.Load()); pct < 100 {
					elapsed := time.Since(batchStart)
					sleepRatio := float64(100-pct) / float64(pct)
					time.Sleep(time.Duration(float64(elapsed) * sleepRatio))
				}
			}
		}(header, startNonce, endNonce)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	res, ok := <-resultCh
	workerCancel()
	wg.Wait()

	totalHashes := hashCount.Load()
	m.hashCount.Add(totalHashes)

	if ok {
		return true, res.nonce, totalHashes
	}

	return false, 0, totalHashes
}