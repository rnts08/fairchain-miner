// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

// Package metrics provides hashrate measurement and reporting.
package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

// HashrateTracker provides EWMA-smoothed hashrate measurement.
type HashrateTracker struct {
	totalHashes atomic.Uint64

	mu            sync.Mutex
	ewmaRate      float64
	lastSnapCount uint64
	lastSnapTime  time.Time
	snapCount     int
	startTime     time.Time

	// Background ticker for periodic snapshots.
	done chan struct{}
}

// EWMA smoothing factor. With a 3-second sample interval, this gives
// an effective time constant of ~60 seconds: alpha = 1 - exp(-3/60) ≈ 0.049.
const ewmaAlpha = 0.049

// NewHashrateTracker creates and starts a new hashrate tracker.
func NewHashrateTracker() *HashrateTracker {
	now := time.Now()
	t := &HashrateTracker{
		lastSnapTime: now,
		startTime:    now,
		done:         make(chan struct{}),
	}

	// Background snapshot goroutine.
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-t.done:
				return
			case <-ticker.C:
				t.snapshot()
			}
		}
	}()

	return t
}

// Add records n hashes completed.
func (t *HashrateTracker) Add(n uint64) {
	t.totalHashes.Add(n)
}

// TotalHashes returns the total number of hashes computed.
func (t *HashrateTracker) TotalHashes() uint64 {
	return t.totalHashes.Load()
}

// Rate returns the current EWMA-smoothed hashrate in H/s.
func (t *HashrateTracker) Rate() float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.ewmaRate
}

// Stop stops the background snapshot goroutine.
func (t *HashrateTracker) Stop() {
	select {
	case <-t.done:
	default:
		close(t.done)
	}
}

func (t *HashrateTracker) snapshot() {
	now := time.Now()
	current := t.totalHashes.Load()

	t.mu.Lock()
	defer t.mu.Unlock()

	dt := now.Sub(t.lastSnapTime).Seconds()
	if dt <= 0 {
		return
	}

	instantRate := float64(current-t.lastSnapCount) / dt
	t.lastSnapCount = current
	t.lastSnapTime = now
	t.snapCount++

	if t.snapCount == 1 {
		t.ewmaRate = instantRate
	} else {
		t.ewmaRate = ewmaAlpha*instantRate + (1-ewmaAlpha)*t.ewmaRate
	}
}
