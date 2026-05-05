// Copyright (c) 2024-2026 The Fairchain Contributors
// Copyright (c) 2024-2026 The Fairchain Contributors
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

package stratum

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/rnts08/fairchain-miner/pkg/algorithm"
	"github.com/rnts08/fairchain-miner/pkg/config"
	"github.com/rnts08/fairchain-miner/pkg/memory"
	"github.com/rnts08/fairchain-miner/pkg/metrics"
	"github.com/rnts08/fairchain-miner/pkg/template"
	"github.com/rnts08/fairchain-miner/pkg/tui"
	"github.com/rnts08/fairchain-miner/pkg/worker"
)
	// --- Flags ---
	rpcAddr := flag.String("rpc", "http://127.0.0.1:19445", "Node RPC address")
	workers := flag.Int("workers", 0, "Number of mining threads (0 = all CPUs)")
	powerLimit := flag.Int("power-limit", 100, "CPU power limit percentage (1-100)")
	benchmark := flag.Bool("benchmark", false, "Benchmark mode (no RPC, measure hashrate only)")
	duration := flag.Duration("duration", 0, "Benchmark duration (0 = run forever)")
	stratumAddr := flag.String("stratum", "", "Stratum server address (e.g. stratum+tcp://pool:3333)")
	stratumUser := flag.String("user", "", "Stratum worker username")
	gpuMode := flag.Bool("gpu", false, "Enable GPU mining (requires CUDA/OpenCL build)")
	gpuDevice := flag.Int("device", 0, "GPU device index")
	showVersion := flag.Bool("version", false, "Show version and exit")
	noTui := flag.Bool("no-tui", false, "Disable interactive TUI")
	flag.Parse()

	if *showVersion {
		fmt.Printf("fairchain-miner %s\n", version)
		os.Exit(0)
	}

	if *workers <= 0 {
		*workers = runtime.NumCPU()
	}
	if *workers < 1 {
		*workers = 1
	}

	// --- Signal handling ---
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		fmt.Fprintf(os.Stderr, "\nshutting down...\n")
		cancel()
	}()

	// --- Banner ---
	hasher := algorithm.New()

	// --- Load Persistent Config ---
	store, _ := config.NewStore("config.sqlite")
	dbCfg, _ := store.Load()

	// Apply flags, overriding loaded config if explicitly set
	if *stratumAddr != "" {
		dbCfg.StratumAddr = *stratumAddr
	} else {
		*stratumAddr = dbCfg.StratumAddr // Use loaded config if flag not set
	}
	if *stratumUser != "" {
		dbCfg.StratumUser = *stratumUser
	} else {
		*stratumUser = dbCfg.StratumUser // Use loaded config if flag not set
	}
	if *powerLimit != 100 { // If power-limit flag was explicitly set (not default 100)
		dbCfg.PowerLimit = *powerLimit
	} else {
		*powerLimit = dbCfg.PowerLimit // Use loaded config if flag not set
	}
	// ThermalLimit, PowerSavingsEnabled, PowerSavingsThreshold are only configurable via TUI for now,
	// so no flag override logic is needed here.


	// --- Initialize Hardware Control State ---
	worker.SetGlobalPowerLimit(*powerLimit)
	memory.SetNumaEnabled(dbCfg.NumaEnabled)
	memory.SetHugepagesEnabled(dbCfg.HugepagesEnabled)
	worker.SetAffinityEnabled(dbCfg.AffinityEnabled)
	// ThermalLimit is handled within TUI, no global setter needed here.

	// --- TUI Setup ---
	var app *tui.App
	if !*noTui && !*benchmark {
		app = tui.NewApp(*workers, hasher.Name(), dbCfg, store) // Pass dbCfg and store
		// Redirect logs to TUI
		// os.Stderr = os.NewFile(uintptr(syscall.Stderr), "/dev/null")
	}

	if app == nil {
		// --- Banner (Console Mode) ---
		fmt.Printf("╔══════════════════════════════════════════════╗\n")
		fmt.Printf("║        fairchain-miner v%s           ║\n", version)
		fmt.Printf("╠══════════════════════════════════════════════╣\n")
		fmt.Printf("║  Algorithm:   %-30s║\n", hasher.Name())
		fmt.Printf("║  Workers:     %-30d║\n", *workers)
		fmt.Printf("║  Power Limit: %-29d%%║\n", *powerLimit)
		if *gpuMode {
			fmt.Printf("║  GPU:         device %-23d║\n", *gpuDevice)
		}
		fmt.Printf("╚══════════════════════════════════════════════╝\n\n")
	}

	// --- Benchmark mode ---
	if *benchmark {
		runBenchmark(ctx, hasher, *workers, *powerLimit, *duration)
		return
	}

	// --- Stratum mode ---
	if *stratumAddr != "" {
		if app != nil {
			go runStratumMiner(ctx, dbCfg.StratumAddr, dbCfg.StratumUser, hasher, *workers, *powerLimit, app)
			if err := app.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
			}
			cancel()
		} else {
			runStratumMiner(ctx, *stratumAddr, *stratumUser, hasher, *workers, *powerLimit, nil)
		}
		return
	}

	// --- GPU mode placeholder ---
	if *gpuMode {
		fmt.Fprintf(os.Stderr, "GPU mining not yet implemented (requires -tags cuda or -tags opencl)\n")
		os.Exit(1)
	}

	// --- Solo mining via RPC ---
	if app != nil {
		go runSoloMiner(ctx, dbCfg.StratumAddr, hasher, *workers, *powerLimit, app) // Use dbCfg.StratumAddr for RPC
		if err := app.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		}
		cancel()
	} else {
		runSoloMiner(ctx, *rpcAddr, hasher, *workers, *powerLimit, nil)
	}
}

// runBenchmark runs the hasher in a tight loop to measure raw hashrate.
func runBenchmark(ctx context.Context, hasher *algorithm.Hasher, numWorkers, powerLimit int, dur time.Duration) {
	fmt.Printf("benchmark mode: %d workers, duration=%s\n\n", numWorkers, dur)

	if dur > 0 {
		var benchCancel context.CancelFunc
		ctx, benchCancel = context.WithTimeout(ctx, dur)
		defer benchCancel()
	}

	// Use a dummy 80-byte header for benchmarking.
	var header [80]byte
	for i := range header {
		header[i] = byte(i)
	}

	tracker := metrics.NewHashrateTracker()
	pool := worker.NewPool(numWorkers, powerLimit)

	pool.RunBenchmark(ctx, hasher, header[:], tracker)

	fmt.Printf("\n── Benchmark Complete ──\n")
	fmt.Printf("  Total hashes: %d\n", tracker.TotalHashes())
	fmt.Printf("  Hashrate:     %.2f H/s\n", tracker.Rate())
	fmt.Printf("  Per-worker:   %.2f H/s\n", tracker.Rate()/float64(numWorkers))
}

// runSoloMiner connects to a node via RPC and mines blocks.
func runSoloMiner(ctx context.Context, rpcAddr string, hasher *algorithm.Hasher, numWorkers, powerLimit int, app *tui.App) {
	client := rpc.NewClient(rpcAddr)
	tracker := metrics.NewHashrateTracker()
	pool := worker.NewPool(numWorkers, powerLimit)
	builder := template.NewBuilder()

	if app == nil {
		fmt.Printf("solo mining: rpc=%s\n\n", rpcAddr)
	} else {
		app.Logf("Solo mining started (RPC: %s)", rpcAddr)
	}

	var totalBlocks atomic.Uint64
	var currentBits atomic.Uint32
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
				rate := tracker.Rate()
				if app != nil {
					bits := currentBits.Load()
					var diff float64
					if bits != 0 {
						diff = stratum.HashToDifficulty(stratum.CompactToHash(bits))
					}
					app.UpdateHashrate(rate)
					app.UpdateNetwork(rpcAddr, diff, int64(totalBlocks.Load()), 0, 0)
					app.UpdateWorkers(pool.WorkerRates(), tui.GetCPUTemps(numWorkers))

					// Check for power savings mode
					tui.CheckPowerSavings(app, numWorkers)

				} else if rate > 0 {
					fmt.Printf("  ⛏  %.2f H/s  |  blocks: %d  |  uptime: %s\n",
						rate, totalBlocks.Load(), time.Since(startTime).Round(time.Second))
				}
			}
		}
	}()

	for ctx.Err() == nil {
		// Fetch chain info.
		info, err := client.GetBlockchainInfo()
		if err != nil {
			fmt.Fprintf(os.Stderr, "rpc error: %v (retrying in 2s)\n", err)
			sleep(ctx, 2*time.Second)
			continue
		}

		// Fetch tip block for timestamp.
		tip, err := client.GetBlockByHeight(info.Height)
		if err != nil {
			fmt.Fprintf(os.Stderr, "fetch tip error: %v\n", err)
			sleep(ctx, 1*time.Second)
			continue
		}

		// Build block template.
		tmpl, err := builder.Build(info, tip)
		if err != nil {
			fmt.Fprintf(os.Stderr, "template error: %v\n", err)
			sleep(ctx, 1*time.Second)
			continue
		}

		currentBits.Store(tmpl.Bits)
		if app != nil {
			app.Logf("Mining height %d  bits=0x%08x  ts=%d",
				tmpl.Height, tmpl.Bits, tmpl.Timestamp)
		} else {
			fmt.Printf("mining height %d  bits=0x%08x  ts=%d\n",
				tmpl.Height, tmpl.Bits, tmpl.Timestamp)
		}

		// Mine.
		result := pool.Mine(ctx, hasher, tmpl, tracker, func() bool {
			// Stale check: re-fetch chain info.
			ci, err := client.GetBlockchainInfo()
			if err != nil {
				return false
			}
			return ci.BestHash != info.BestHash
		})

		if ctx.Err() != nil {
			break
		}

		if !result.Found {
			fmt.Printf("  stale or exhausted after %d hashes\n", result.Hashes)
			continue
		}

		// Submit block.
		block := builder.Assemble(tmpl, result.Nonce)
		rejected, detail := client.SubmitBlock(block)
		if rejected {
			fmt.Printf("  REJECTED: %s\n", detail)
			sleep(ctx, 500*time.Millisecond)
			continue
		}

		totalBlocks.Add(1)
		if app != nil {
			app.Logf("✓ ACCEPTED  height=%d  nonce=%d  hashes=%d  blocks_mined=%d",
				tmpl.Height, result.Nonce, result.Hashes, totalBlocks.Load())
		} else {
			fmt.Printf("  ✓ ACCEPTED  height=%d  nonce=%d  hashes=%d  blocks_mined=%d\n\n",
				tmpl.Height, result.Nonce, result.Hashes, totalBlocks.Load())
		}
	}

	if app == nil {
		fmt.Printf("\nfairchain-miner stopped. mined %d blocks in %s\n",
			totalBlocks.Load(), time.Since(startTime).Round(time.Second))
	}
}

// runStratumMiner connects to a stratum pool and mines.
func runStratumMiner(ctx context.Context, addr, user string, hasher *algorithm.Hasher, numWorkers, powerLimit int, app *tui.App) {
	if app == nil {
		fmt.Printf("stratum mining: %s (user=%s)\n\n", addr, user)
	} else {
		app.Logf("Connecting to stratum: %s...", addr)
	}

	client := stratum.NewClient(addr, user, "x", hasher, func(msg string) {
		if app != nil {
			app.Log(msg)
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
				rate := tracker.Rate()
				if app != nil {
					wRates := make([]float64, numWorkers)
					for i, t := range workerTrackers {
						wRates[i] = t.Rate()
					}
					app.UpdateHashrate(rate)
					app.UpdateNetwork(addr, client.CurrentDifficulty(),
						client.Stats.SharesAccepted.Load(),
						client.Stats.SharesRejected.Load(),
						client.Stats.SharesStale.Load())
					app.UpdateWorkers(wRates, tui.GetCPUTemps(numWorkers), client.LastLatency())
				} else if rate > 0 {
					accepted := client.Stats.SharesAccepted.Load()
					rejected := client.Stats.SharesRejected.Load()
					stale := client.Stats.SharesStale.Load()
					blocks := client.Stats.BlocksFound.Load()
					fmt.Printf("  ⛏  %s  |  shares: %d A / %d R / %d S  |  blocks: %d  |  uptime: %s\n",
						metrics.FormatHashrate(rate), accepted, rejected, stale, blocks, time.Since(startTime).Round(time.Second))
				}
			}
		}
	}()

	var (
		jobMu      sync.RWMutex
		currentJob *stratum.Job
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
								app.Logf("Worker %d: Memory re-allocated (NUMA: %v, Hugepages: %v)", workerID, lastNuma, lastHuge)
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
								app.Logf("!!! BLOCK FOUND !!! (nonce=%d)", nonce)
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
			fmt.Printf("  [stratum] new job %s (diff %.4f)\n", job.ID, client.CurrentDifficulty())
		}
	}
}

func sleep(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}
