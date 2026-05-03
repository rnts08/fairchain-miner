// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

package metrics

import (
	"fmt"
	"time"
)

// FormatHashrate returns a human-readable hashrate string with appropriate unit.
func FormatHashrate(hps float64) string {
	switch {
	case hps >= 1e12:
		return fmt.Sprintf("%.2f TH/s", hps/1e12)
	case hps >= 1e9:
		return fmt.Sprintf("%.2f GH/s", hps/1e9)
	case hps >= 1e6:
		return fmt.Sprintf("%.2f MH/s", hps/1e6)
	case hps >= 1e3:
		return fmt.Sprintf("%.2f KH/s", hps/1e3)
	default:
		return fmt.Sprintf("%.2f H/s", hps)
	}
}

// FormatDuration returns a compact human-readable duration.
func FormatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60

	if h > 0 {
		return fmt.Sprintf("%dh%02dm%02ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%dm%02ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}

// Reporter formats and prints mining statistics to the console.
type Reporter struct {
	tracker   *HashrateTracker
	startTime time.Time
}

// NewReporter creates a new console reporter.
func NewReporter(tracker *HashrateTracker) *Reporter {
	return &Reporter{
		tracker:   tracker,
		startTime: time.Now(),
	}
}

// PrintStatus prints a one-line status update.
func (r *Reporter) PrintStatus(blocksFound uint64) {
	rate := r.tracker.Rate()
	total := r.tracker.TotalHashes()
	uptime := time.Since(r.startTime)

	fmt.Printf("  ⛏  %s  |  hashes: %d  |  blocks: %d  |  uptime: %s\n",
		FormatHashrate(rate), total, blocksFound, FormatDuration(uptime))
}

// PrintBenchmarkResult prints final benchmark results.
func (r *Reporter) PrintBenchmarkResult(numWorkers int) {
	rate := r.tracker.Rate()
	total := r.tracker.TotalHashes()
	uptime := time.Since(r.startTime)

	fmt.Printf("\n══════════════════════════════════════════════\n")
	fmt.Printf("  Benchmark Complete\n")
	fmt.Printf("══════════════════════════════════════════════\n")
	fmt.Printf("  Total hashes:  %d\n", total)
	fmt.Printf("  Duration:      %s\n", FormatDuration(uptime))
	fmt.Printf("  Hashrate:      %s\n", FormatHashrate(rate))
	fmt.Printf("  Per-worker:    %s\n", FormatHashrate(rate/float64(numWorkers)))
	fmt.Printf("  Workers:       %d\n", numWorkers)
	fmt.Printf("══════════════════════════════════════════════\n")
}
