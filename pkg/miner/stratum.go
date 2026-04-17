// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

package miner

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"github.com/bams-repo/fairchain/internal/algorithms"
	"github.com/bams-repo/fairchain/internal/crypto"
	"github.com/bams-repo/fairchain/internal/types"
)

// StratumClient implements Stratum V1 pool mining client
type StratumClient struct {
	conn   net.Conn
	host   string
	user   string
	pass   string
	hasher algorithms.Hasher
	miner  *Miner

	extranonce1     string
	extranonce2size int
	difficulty      float64
	nextID          uint64

	currentJob atomic.Value // *StratumJob

	connected atomic.Bool
	authOK    atomic.Bool

	onNewJob func(*StratumJob)
	onShare  func(bool, uint64)
}

// StratumJob represents an active mining job from pool
type StratumJob struct {
	ID           string
	PrevBlock    types.Hash
	Coinbase1    []byte
	Coinbase2    []byte
	MerkleBranch []types.Hash
	Version      uint32
	Bits         uint32
	Timestamp    uint32
	CleanJobs    bool

	target types.Hash
}

// NewStratumClient creates a stratum client
func NewStratumClient(host, user, pass string, m *Miner, h algorithms.Hasher) *StratumClient {
	return &StratumClient{
		host:   host,
		user:   user,
		pass:   pass,
		miner:  m,
		hasher: h,
	}
}

// SetOnShare sets the callback for share results
func (sc *StratumClient) SetOnShare(f func(bool, uint64)) {
	sc.onShare = f
}

// SetOnNewJob sets the callback for new job notifications
func (sc *StratumClient) SetOnNewJob(f func(*StratumJob)) {
	sc.onNewJob = f
}

// Connect establishes connection to stratum pool
func (sc *StratumClient) Connect(ctx context.Context) error {
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", sc.host)
	if err != nil {
		return err
	}
	sc.conn = conn
	sc.connected.Store(true)

	// NOTE: We do NOT start readLoop here
	// The server sends notifications AFTER the subscribe response
	// We need to receive them in the request() loop, not in a separate goroutine
	// Start readLoop AFTER handshake completes
	return sc.handshake()
}

func (sc *StratumClient) handshake() error {
	if err := sc.subscribe(); err != nil {
		return err
	}
	if err := sc.authorize(); err != nil {
		return err
	}
	// Start readLoop after handshake - now notifications will come
	go sc.readLoop(context.Background())
	return nil
}

func (sc *StratumClient) subscribe() error {
	req := map[string]interface{}{
		"id":     sc.nextID,
		"method": "mining.subscribe",
		"params": []string{"standalone-miner/1.0"},
	}
	sc.nextID++

	resp, err := sc.request(req)
	if err != nil {
		return err
	}

	result, ok := resp.Result.([]interface{})
	if !ok || len(result) < 3 {
		return fmt.Errorf("invalid subscribe response: result=%v, len=%d", resp.Result, len(result))
	}

	// Response format: [[subscriptions], extranonce1, extranonce2_size]
	// subscriptions is array of [method, subscription_id] pairs

	extranonce1, ok := result[1].(string)
	if !ok {
		return fmt.Errorf("invalid extranonce1")
	}
	sc.extranonce1 = extranonce1

	en2Size, ok := result[2].(float64)
	if !ok {
		return fmt.Errorf("invalid extranonce2 size")
	}
	sc.extranonce2size = int(en2Size)

	return nil
}

func (sc *StratumClient) authorize() error {
	req := map[string]interface{}{
		"id":     sc.nextID,
		"method": "mining.authorize",
		"params": []string{sc.user, sc.pass},
	}
	sc.nextID++

	resp, err := sc.request(req)
	if err != nil {
		return err
	}

	ok, isBool := resp.Result.(bool)
	if !isBool || !ok {
		return fmt.Errorf("authorization failed")
	}

	sc.authOK.Store(true)
	return nil
}

func (sc *StratumClient) request(req map[string]interface{}) (*stratumResponse, error) {
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	data = append(data, '\n')

	if _, err := sc.conn.Write(data); err != nil {
		return nil, err
	}

	// Get numeric request ID for comparison
	var reqID float64
	reqID = float64(req["id"].(uint64))
	fmt.Printf("DEBUG: reqID set to %v (original type: %T)\n", reqID, req["id"])

	// Read ALL available data - keep reading until timeout
	// The server may send multiple messages
	sc.conn.SetReadDeadline(time.Now().Add(15 * time.Second))

	buf := make([]byte, 0, 8192)
	tmp := make([]byte, 4096)
	for {
		n, err := sc.conn.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				// Timeout is expected when server finishes sending
				break
			}
			// Other error - might be EOF or close
			break
		}
	}

	fmt.Printf("DEBUG request: received %d bytes\n", len(buf))

	// Print first part of raw data
	if len(buf) > 0 {
		showLen := len(buf)
		if showLen > 200 {
			showLen = 200
		}
		fmt.Printf("DEBUG request: raw start: %s\n", string(buf[:showLen]))
	}

	// Process each line separately
	lines := splitLines(string(buf))
	fmt.Printf("DEBUG request: got %d lines\n", len(lines))

	var response *stratumResponse

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var msg stratumResponse
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}

		// Handle notifications - server sends them before the response
		// But don't skip looking for response after handling them
		hasMethod := msg.Method != ""

		if hasMethod {
			sc.handleNotification(&msg)
			// Don't continue - still need to find the response
		}

		// Check ID match for response
		idMatched := false

		// Only consider as response if:
		// 1. Has no method (not a notification), AND
		// 2. Has matching ID (or null ID which some servers use)
		if !hasMethod {
			fmt.Printf("DEBUG: checking response ID=%v (want %v), Method=%q\n", msg.ID, reqID, msg.Method)
			if msg.ID == nil {
				// null ID can be a response
				idMatched = true
			} else if id, ok := msg.ID.(float64); ok {
				idMatched = (id == reqID)
			}

			if idMatched {
				response = &msg
				break
			}
		}
	}

	if response != nil {
		return response, nil
	}

	return nil, fmt.Errorf("no matching response found")
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func (sc *StratumClient) readLoop(ctx context.Context) {
	scanner := bufio.NewScanner(sc.conn)
	scanner.Buffer(make([]byte, 16384), 16384)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return
		default:
		}

		var msg stratumResponse
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			continue
		}

		if msg.Method != "" {
			sc.handleNotification(&msg)
		}
	}
}

func (sc *StratumClient) handleNotification(msg *stratumResponse) {
	switch msg.Method {
	case "mining.notify":
		sc.handleJob(msg)
	case "mining.set_difficulty":
		sc.handleDifficulty(msg)
	}
}

func (sc *StratumClient) handleJob(msg *stratumResponse) {
	params, ok := msg.Params.([]interface{})
	if !ok || len(params) < 9 {
		return
	}

	jobID, _ := params[0].(string)
	prevhashHex, _ := params[1].(string)
	coinbase1Hex, _ := params[2].(string)
	coinbase2Hex, _ := params[3].(string)
	merkleBranchRaw, _ := params[4].([]interface{})
	versionHex, _ := params[5].(string)
	bitsHex, _ := params[6].(string)
	ntimeHex, _ := params[7].(string)
	cleanJobs, _ := params[8].(bool)

	prevhash, _ := decodeStratumPrevhash(prevhashHex)
	coinbase1, _ := hex.DecodeString(coinbase1Hex)
	coinbase2, _ := hex.DecodeString(coinbase2Hex)

	var merkleBranch []types.Hash
	for _, h := range merkleBranchRaw {
		if hs, ok := h.(string); ok {
			var hash types.Hash
			if b, err := hex.DecodeString(hs); err == nil && len(b) == 32 {
				copy(hash[:], b)
				merkleBranch = append(merkleBranch, hash)
			}
		}
	}

	var version uint32
	fmt.Sscanf(versionHex, "%x", &version)

	var bits uint32
	fmt.Sscanf(bitsHex, "%x", &bits)

	var ntime uint32
	fmt.Sscanf(ntimeHex, "%x", &ntime)

	// Use bits to compute target (not difficulty, which may be wrong)
	target := crypto.CompactToHash(bits)

	job := &StratumJob{
		ID:           jobID,
		PrevBlock:    prevhash,
		Coinbase1:    coinbase1,
		Coinbase2:    coinbase2,
		MerkleBranch: merkleBranch,
		Version:      version,
		Bits:         bits,
		Timestamp:    ntime,
		CleanJobs:    cleanJobs,
		target:       target,
	}

	sc.currentJob.Store(job)

	if sc.onNewJob != nil {
		sc.onNewJob(job)
	}
}

func (sc *StratumClient) handleDifficulty(msg *stratumResponse) {
	params, ok := msg.Params.([]interface{})
	if !ok || len(params) < 1 {
		return
	}

	diff, ok := params[0].(float64)
	if ok && diff > 0 {
		sc.difficulty = diff
	}
}

// SubmitShare submits a found share to pool
func (sc *StratumClient) SubmitShare(job *StratumJob, nonce uint32, ntime uint32, extraNonce2 uint32) error {
	en2 := make([]byte, 4)
	binary.BigEndian.PutUint32(en2, extraNonce2)

	req := map[string]interface{}{
		"id":     sc.nextID,
		"method": "mining.submit",
		"params": []string{
			sc.user,
			job.ID,
			hex.EncodeToString(en2),
			fmt.Sprintf("%08x", ntime),
			fmt.Sprintf("%08x", nonce),
		},
	}
	sc.nextID++

	resp, err := sc.request(req)
	if err != nil {
		return err
	}

	accepted, ok := resp.Result.(bool)
	if sc.onShare != nil {
		sc.onShare(ok && accepted, 1)
	}

	if !ok || !accepted {
		return fmt.Errorf("share rejected")
	}

	return nil
}

// CurrentJob returns the active mining job
func (sc *StratumClient) CurrentJob() *StratumJob {
	if v := sc.currentJob.Load(); v != nil {
		return v.(*StratumJob)
	}
	return nil
}

// Close disconnects from pool
func (sc *StratumClient) Close() error {
	sc.connected.Store(false)
	return sc.conn.Close()
}

type stratumResponse struct {
	ID     interface{} `json:"id"`
	Result interface{} `json:"result"`
	Error  interface{} `json:"error"`
	Method string      `json:"method"`
	Params interface{} `json:"params"`
}

func decodeStratumPrevhash(s string) (types.Hash, error) {
	b, err := hex.DecodeString(s)
	if err != nil {
		return types.Hash{}, err
	}
	var h types.Hash
	for i := 0; i < 32; i += 4 {
		h[i+0] = b[i+3]
		h[i+1] = b[i+2]
		h[i+2] = b[i+1]
		h[i+3] = b[i+0]
	}
	return h, nil
}

func difficultyToTarget(diff float64) types.Hash {
	if diff <= 0 {
		diff = 1
	}

	diff1, _ := new(big.Int).SetString("00000000FFFF0000000000000000000000000000000000000000000000000000", 16)

	target := new(big.Float).SetInt(diff1)
	target.Quo(target, new(big.Float).SetFloat64(diff))

	targetInt, _ := target.Int(nil)
	b := targetInt.Bytes()

	var h types.Hash
	for i := 0; i < len(b) && i < 32; i++ {
		h[31-i] = b[i]
	}
	return h
}
