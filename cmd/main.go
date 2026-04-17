package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/bams-repo/fairchain/internal/algorithms"
	"github.com/bams-repo/fairchain/internal/coinparams"
	"github.com/bams-repo/fairchain/internal/crypto"
	"github.com/bams-repo/fairchain/internal/types"
	"github.com/bams-repo/fairchain/pkg/miner"
)

type chainInfo struct {
	Height   uint32 `json:"blocks"`
	BestHash string `json:"bestblockhash"`
	Bits     string `json:"bits"`
	Chain    string `json:"chain"`
}

type blockInfo struct {
	Hash      string `json:"hash"`
	Height    uint32 `json:"height"`
	Timestamp uint32 `json:"time"`
	Bits      string `json:"bits"`
	Nonce     uint32 `json:"nonce"`
}

type Config struct {
	RPCAddr string `toml:"rpc_address"`
	Workers int    `toml:"workers"`
	Power   int    `toml:"power_limit"`
}

func loadConfig(path string) (*Config, error) {
	cfg := &Config{
		RPCAddr: "http://127.0.0.1:19335",
		Workers: runtime.NumCPU(),
		Power:   100,
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}
	_, err := toml.DecodeFile(path, cfg)
	return cfg, err
}


func main() {
	cfg, err := loadConfig("config.toml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	rpcAddr := flag.String("rpc", cfg.RPCAddr, "Node RPC address")
	workers := flag.Int("workers", cfg.Workers, "Number of mining threads")
	power := flag.Int("power", cfg.Power, "CPU power limit percentage (1-100)")
	simulate := flag.Bool("t", false, "Run in offline simulation mode")
	flag.Parse()

	h, err := algorithms.GetHasher(coinparams.Algorithm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unsupported PoW algorithm %q: %v\n", coinparams.Algorithm, err)
		os.Exit(1)
	}

	m := miner.New(h, *workers)
	m.SetPowerLimit(*power)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sig
		fmt.Fprintf(os.Stderr, "\nshutting down...\n")
		cancel()
	}()

	m.StartHashrateMonitor(ctx)

	if *simulate {
		// Run TUI simulation mode
		runTUI(true, *rpcAddr, *workers, *power)
		return
	}

	fmt.Printf("fairchain-miner: algo=%s workers=%d power=%d%% rpc=%s\n\n",
		coinparams.Algorithm, m.Workers(), m.PowerLimit(), *rpcAddr)

	var totalBlocks uint64
	startTime := time.Now()

	for ctx.Err() == nil {
		ci, err := fetchChainInfo(*rpcAddr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "rpc error: %v (retrying in 2s)\n", err)
			sleep(ctx, 2*time.Second)
			continue
		}

		tip, err := fetchBlockByHeight(*rpcAddr, ci.Height)
		if err != nil {
			fmt.Fprintf(os.Stderr, "fetch tip error: %v\n", err)
			sleep(ctx, 1*time.Second)
			continue
		}

		prevHash, err := types.HashFromReverseHex(ci.BestHash)
		if err != nil {
			fmt.Fprintf(os.Stderr, "parse hash error: %v\n", err)
			sleep(ctx, 1*time.Second)
			continue
		}

		var bits uint32
		fmt.Sscanf(ci.Bits, "%x", &bits)
		newHeight := ci.Height + 1

		blockTimestamp := uint32(time.Now().Unix())
		if blockTimestamp <= tip.Timestamp {
			blockTimestamp = tip.Timestamp + 1
		}

		subsidy := fetchSubsidy(ci.Chain, newHeight)
		cb := makeCoinbaseTx(newHeight, subsidy)

		block := &types.Block{
			Header: types.BlockHeader{
				Version:    1,
				PrevBlock:  prevHash,
				Timestamp:  blockTimestamp,
				Bits:       bits,
				Nonce:      0,
			},
			Transactions: []types.Transaction{cb},
		}

		merkle, err := crypto.ComputeMerkleRoot(block.Transactions)
		if err != nil {
			fmt.Fprintf(os.Stderr, "merkle error: %v\n", err)
			continue
		}
		block.Header.MerkleRoot = merkle

		target := crypto.CompactToHash(bits)
		work := crypto.CalcWork(bits)

		hashrate := m.Hashrate()
		hashrateStr := ""
		if m.HashrateReady() {
			hashrateStr = fmt.Sprintf("  hashrate=%.1f H/s", float64(hashrate))
		}

		fmt.Printf("mining height %d  bits=0x%08x  expected_hashes=%d%s\n",
			newHeight, bits, work, hashrateStr)

		start := time.Now()
		found, nonce, hashes := m.MineHeader(ctx, block.Header, target)
		elapsed := time.Since(start)

		if ctx.Err() != nil {
			break
		}

		if !found {
			fmt.Printf("  nonce space exhausted after %d hashes (%.1fs)\n", hashes, elapsed.Seconds())
			continue
		}

		block.Header.Nonce = nonce
		blockHash := crypto.HashBlockHeader(&block.Header)

		fmt.Printf("  FOUND nonce=%d hashes=%d time=%.1fs rate=%.1f H/s\n",
			nonce, hashes, elapsed.Seconds(), float64(hashes)/elapsed.Seconds())

		rejected, detail := submitBlock(*rpcAddr, block)
		if rejected {
			fmt.Printf("  REJECTED: %s\n", detail)
			sleep(ctx, 500*time.Millisecond)
			continue
		}

		totalBlocks++
		elapsedTotal := time.Since(startTime)
		fmt.Printf("  ACCEPTED hash=%s  height=%d  total_mined=%d  uptime=%s\n\n",
			blockHash.ReverseString()[:16], newHeight, totalBlocks, elapsedTotal.Round(time.Second))
	}

	fmt.Printf("\nfairchain-miner stopped. mined %d blocks in %s\n",
		totalBlocks, time.Since(startTime).Round(time.Second))
}



func makeCoinbaseTx(height uint32, subsidy uint64) types.Transaction {
	pushLen := minimalHeightPushLen(height)
	heightBytes := make([]byte, 4)
	types.PutUint32LE(heightBytes, height)

	msg := make([]byte, 0, 1+pushLen+len("standalone"))
	msg = append(msg, byte(pushLen))
	msg = append(msg, heightBytes[:pushLen]...)
	msg = append(msg, []byte("standalone")...)

	return types.Transaction{
		Version: 1,
		Inputs: []types.TxInput{{
			PreviousOutPoint: types.CoinbaseOutPoint,
			SignatureScript:  msg,
			Sequence:         0xFFFFFFFF,
		}},
		Outputs: []types.TxOutput{{
			Value:    subsidy,
			PkScript: []byte{0x00},
		}},
		LockTime: 0,
	}
}

func minimalHeightPushLen(height uint32) int {
	switch {
	case height <= 0xFF:
		return 1
	case height <= 0xFFFF:
		return 2
	case height <= 0xFFFFFF:
		return 3
	default:
		return 4
	}
}

func fetchSubsidy(chain string, height uint32) uint64 {
	var initial uint64
	var halving uint32
	switch chain {
	case "testnet":
		initial = 50_0000_00
		halving = 21_000_000
	case "mainnet":
		initial = 50_0000_0000
		halving = 210_000
	default:
		initial = 50_0000_0000
		halving = 150
	}
	halvings := height / halving
	if halvings >= 64 {
		return 0
	}
	return initial >> halvings
}

func fetchChainInfo(rpc string) (*chainInfo, error) {
	resp, err := http.Get(rpc + "/getblockchaininfo")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var info chainInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}

func fetchBlockByHeight(rpc string, height uint32) (*blockInfo, error) {
	resp, err := http.Get(fmt.Sprintf("%s/getblockbyheight?height=%d", rpc, height))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var info blockInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}

func submitBlock(rpc string, block *types.Block) (rejected bool, detail string) {
	data, err := block.SerializeToBytes()
	if err != nil {
		return true, fmt.Sprintf("serialize error: %v", err)
	}

	resp, err := http.Post(rpc+"/submitblock", "application/octet-stream", bytes.NewReader(data))
	if err != nil {
		return true, fmt.Sprintf("http error: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return true, string(body)
	}
	return false, string(body)
}

func sleep(ctx context.Context, d time.Duration) {
	select {
	case <-ctx.Done():
	case <-time.After(d):
	}
}