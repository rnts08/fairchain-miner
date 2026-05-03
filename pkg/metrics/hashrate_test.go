// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

package metrics

import (
	"testing"
	"time"
)

func TestHashrateTracker(t *testing.T) {
	tracker := NewHashrateTracker()
	defer tracker.Stop()

	// Add some hashes.
	tracker.Add(1000)

	if tracker.TotalHashes() != 1000 {
		t.Errorf("expected 1000 total hashes, got %d", tracker.TotalHashes())
	}

	// Rate should be 0 initially (before first snapshot).
	rate := tracker.Rate()
	if rate != 0 {
		t.Errorf("expected 0 initial rate, got %f", rate)
	}

	// Force a snapshot by waiting slightly and calling snapshot directly.
	time.Sleep(10 * time.Millisecond)
	tracker.snapshot()

	// After snapshot, rate should be non-zero.
	rate = tracker.Rate()
	if rate <= 0 {
		t.Errorf("expected positive rate after snapshot, got %f", rate)
	}
}

func TestHashrateTrackerConcurrent(t *testing.T) {
	tracker := NewHashrateTracker()
	defer tracker.Stop()

	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			for j := 0; j < 100; j++ {
				tracker.Add(1)
			}
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	if tracker.TotalHashes() != 1000 {
		t.Errorf("expected 1000 total hashes from concurrent adds, got %d", tracker.TotalHashes())
	}
}
