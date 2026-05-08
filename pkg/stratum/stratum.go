// Copyright (c) 2024-2026 The Fairchain Contributors
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

package stratum

import (
	"context"
	"crypto/sha256"
	"encoding"
	"encoding/binary"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/rnts08/fairchain-miner/pkg/algorithm"
	"github.com/rnts08/fairchain-miner/pkg/memory"
	"github.com/rnts08/fairchain-miner/pkg/metrics"
	"github.com/rnts08/fairchain-miner/pkg/tui"
	"github.com/rnts08/fairchain-miner/pkg/worker"
)

// RunMiner connects to a stratum pool and mines.
func RunMiner(ctx context.Context, addr, user string, hasher algorithm.Hasher, numWorkers, powerLimit int, app *tui.App) {
	if app == nil {
		fmt.Printf("stratum mining: %s (user=%s)\n\n", addr, user)
	} else {
		app.Logf(tui.LogStratum, "Connecting to stratum: %s...", addr)
	}

	client := NewClient(addr, user, "x", hasher, func(msg string) {
		if app != nil {
			app.LogStratum(msg)
		}
	})

	if err := client.Connect(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "stratum connect failed: %v\n", err)
		os.Exit(1)
	}

	go client.Run(ctx)

	// Trackers for individual workers to feed the detailed view.
	workerTrackers := make([]*metrics.HashrateTracker, numWorkers)
	for i := range workerTrackers {
		workerTrackers[i] = metrics.NewHashrateTracker()
	}

	tracker := metrics.NewHashrateTracker()
	startTime := time.Now()

	// Hashrate reporter goroutine.
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				r1, r15, r24 := tracker.Rates()
				if app != nil {
					wRates := make([]float64, numWorkers)
					for i, t := range workerTrackers {
						wRates[i] = t.Rate()
					}
					app.UpdateHashrate(r1, r15, r24)
					app.UpdateNetwork(addr, client.CurrentDifficulty(),
						client.Stats.SharesAccepted.Load(),
						client.Stats.SharesRejected.Load(),
						client.Stats.SharesStale.Load())
					app.UpdateWorkers(wRates, tui.GetCPUTemps(numWorkers), client.LastLatency())

					const blockReward, blockTime = 50.0, 600.0
					const networkDifficultyFactor = 4295032833.0
					simReward := (r1 * blockReward * blockTime) / (client.CurrentDifficulty() * networkDifficultyFactor)
					totalShares := client.Stats.SharesAccepted.Load() + client.Stats.SharesRejected.Load() + client.Stats.SharesStale.Load()
					var avgSolveTime time.Duration
					if totalShares > 0 {
						avgSolveTime = time.Since(startTime) / time.Duration(totalShares)
					}
					app.UpdateSummary(client.Stats.SharesAccepted.Load(), client.Stats.SharesRejected.Load(), client.Stats.SharesStale.Load(), avgSolveTime, simReward)
				} else if r1 > 0 {
					accepted := client.Stats.SharesAccepted.Load()
					rejected := client.Stats.SharesRejected.Load()
					stale := client.Stats.SharesStale.Load()
					blocks := client.Stats.BlocksFound.Load()
					fmt.Printf("  ⛏  %s  |  shares: %d A / %d R / %d S  |  blocks: %d  |  uptime: %s\n",
						metrics.FormatHashrate(r1), accepted, rejected, stale, blocks, time.Since(startTime).Round(time.Second))
				}
			}
		}
	}()

	var (
		jobMu      sync.RWMutex
		currentJob *Job
		jobCond    = sync.NewCond(&jobMu)
	)

	// Start workers.
	for w := 0; w < numWorkers; w++ {
		go func(workerID int) {
			defer workerTrackers[workerID].Stop()
			lastAffinity := worker.IsAffinityEnabled()
			if lastAffinity {
				_ = worker.SetAffinity(workerID)
			}
			node := memory.GetNodeForCPU(workerID)
			ws := algorithm.NewWorkspaceOnNode(node)
			defer ws.Free()

			lastNuma := memory.IsNumaEnabled()
			lastHuge := memory.IsHugepagesEnabled()
			var lastJobID string

			for {
				select {
				case <-ctx.Done():
					return
				default:
				}

				jobMu.RLock()
				for currentJob == nil {
					jobMu.RUnlock()
					jobMu.Lock()
					jobCond.Wait()
					jobMu.Unlock()
					jobMu.RLock()
				}
				job := currentJob
				jobMu.RUnlock()

				// If it's a new job, we need to prepare.
				if job.ID != lastJobID {
					lastJobID = job.ID
				}

				// Generate unique extranonce2 for this worker/attempt.
				en2 := client.NextExtranonce2()
				hdr := client.BuildHeader(job, en2, 0)

				// Mine for a short burst (or until job change).
				// We use pool logic but for a single worker.
				// Actually, I'll just do a batch of nonces.
				sn := uint64(workerID) * (0x100000000 / uint64(numWorkers))
				en := sn + (0x100000000 / uint64(numWorkers))

				// Stamp extranonce once, then iterate nonces.
				headerBuf := hdr

				// Precompute SHA-256 midstate.
				midHasher := sha256.New()
				midHasher.Write(headerBuf[:64])
				midState, _ := midHasher.(encoding.BinaryMarshaler).MarshalBinary()

				for pos := sn; pos < en; pos++ {
					// Check for job change periodically.
					if pos%1024 == 0 {
						select {
						case <-ctx.Done():
							return
						default:
						}
						jobMu.RLock()
						if currentJob != job {
							jobMu.RUnlock()
							break
						}
						jobMu.RUnlock()

						// Check for hardware setting changes (NUMA/Hugepages/Affinity)
						if worker.IsAffinityEnabled() != lastAffinity {
							if worker.IsAffinityEnabled() {
								_ = worker.SetAffinity(workerID)
							} else {
								_ = worker.UnsetAffinity()
							}
							lastAffinity = worker.IsAffinityEnabled()
						}

						if memory.IsNumaEnabled() != lastNuma || memory.IsHugepagesEnabled() != lastHuge {
							ws.Free()
							lastNuma = memory.IsNumaEnabled()
							lastHuge = memory.IsHugepagesEnabled()
							ws = algorithm.NewWorkspaceOnNode(node)
							if app != nil {
								app.Logf(tui.LogMining, "Worker %d: Memory re-allocated (NUMA: %v, Hugepages: %v)", workerID, lastNuma, lastHuge)
							}
						}
					}

					nonce := uint32(pos)
					binary.LittleEndian.PutUint32(headerBuf[76:80], nonce)

					h := hasher.PoWHashMidstate(headerBuf[:], ws, midState)
					tracker.Add(1)
					workerTrackers[workerID].Add(1)

					if h.LessOrEqual(job.Target) {
						// Share found!
						_ = client.SubmitShare(job.ID, en2, job.NTime, nonce)

						// Network block?
						if h.LessOrEqual(job.NetTarget) {
							client.Stats.BlocksFound.Add(1)
							if app != nil {
								app.Logf(tui.LogMining, "!!! BLOCK FOUND !!! (nonce=%d)", nonce)
							} else {
								fmt.Printf("  [!] BLOCK FOUND! (nonce=%d)\n", nonce)
							}
						}
					}

					// Throttling.
					currPower := worker.GetGlobalPowerLimit()
					if currPower < 100 && pos%100 == 0 {
						sleepRatio := float64(100-currPower) / float64(currPower)
						time.Sleep(time.Duration(float64(time.Millisecond) * sleepRatio))
					}
				}
			}
		}(w)
	}

	for {
		select {
		case <-ctx.Done():
			client.Close()
			return
		case err := <-client.ErrChannel():
			fmt.Fprintf(os.Stderr, "stratum error: %v\n", err)
			// Reconnect happens inside client.Run, so we just log and continue?
			// Wait, if it's a fatal error, we exit.
			// But our Run has reconnect logic.
		case job := <-client.JobChannel():
			jobMu.Lock()
			currentJob = job
			jobCond.Broadcast()
			jobMu.Unlock()
			if app == nil {
				fmt.Printf("  [stratum] new job %s (diff %.4f)\n", job.ID, client.CurrentDifficulty())
			}
		}
	}
}

func sleep(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}
