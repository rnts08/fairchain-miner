//go:build !tui

package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bams-repo/fairchain/internal/coinparams"
)

func runTUI(simulate bool, rpcAddr string, workers int, power int) {
	fmt.Printf("fairchain-miner: Running in SIMULATION MODE\n")
	fmt.Printf("algo=%s workers=%d power=%d%%\n\n",
		coinparams.Algorithm, workers, power)

	fmt.Println("Running CLI simulation mode...")

	// Simple CLI simulation
	hashes := uint64(0)
	start := time.Now()
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		fmt.Fprintf(os.Stderr, "\nshutting down...\n")
		cancel()
	}()

	for range ticker.C {
		select {
		case <-ctx.Done():
			return
		default:
		}

		hashrate := 18500 + (rand.Int63() % 7000)
		hashes += uint64(hashrate)
		fmt.Printf("\rHashrate: %d H/s | Total hashes: %d | Time: %s",
			hashrate, hashes, time.Since(start).Round(time.Second))
	}
}
