// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

// Package stratum implements a Stratum V1 mining client compatible with
// the fairchain-src built-in stratum server.
package stratum

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/awnumar/memguard"
	"github.com/rnts08/fairchain-miner/pkg/algorithm"
	"github.com/rnts08/fairchain-miner/pkg/types"
)

// Stats holds stratum mining statistics.
type Stats struct {
	SharesSubmitted atomic.Int64
	SharesAccepted  atomic.Int64
	SharesRejected  atomic.Int64
	SharesStale     atomic.Int64
	BlocksFound     atomic.Int64
}

// Job represents a stratum mining job received from the server.
type Job struct {
	ID           string
	PrevHash     [32]byte // LE internal byte order (decoded from stratum's swapped format)
	Coinbase1    []byte
	Coinbase2    []byte
	MerkleBranch []types.Hash
	Version      uint32
	Bits         uint32
	NTime        uint32
	CleanJobs    bool
	Target       types.Hash  // share target (from set_difficulty)
	NetTarget    types.Hash  // network target (from bits)
}

// Client is a Stratum V1 mining client.
type Client struct {
	addr       *memguard.LockedBuffer
	workerName *memguard.LockedBuffer
	password   *memguard.LockedBuffer
	workerMu   sync.RWMutex

	conn     net.Conn
	scanner  *bufio.Scanner
	writeMu  sync.Mutex

	extranonce1    []byte
	extranonce2Len int

	difficulty   float64
	diffMu       sync.RWMutex

	requestTimes map[uint64]time.Time
	lastLatency  atomic.Int64 // nanoseconds

	currentJob   *Job
	jobMu        sync.RWMutex

	Stats Stats
	
	hasher  algorithm.Hasher // Changed from *algorithm.Hasher
	msgID   atomic.Uint64
	en2ID   atomic.Uint64 // unique id for extranonce2 generation

	// Channels for communication.
	jobCh  chan *Job
	errCh  chan error

	onLog func(string) // log callback
}

// NewClient creates a new stratum client.
func NewClient(addr, workerName, password string, hasher algorithm.Hasher, onLog func(string)) *Client {
	if password == "" {
		password = "x"
	}
	return &Client{
		addr:       memguard.NewBufferFromBytes([]byte(addr)),
		workerName: memguard.NewBufferFromBytes([]byte(workerName)),
		password:   memguard.NewBufferFromBytes([]byte(password)),
		hasher:     hasher,
		difficulty: 0.001,
		jobCh:      make(chan *Job, 4),
		errCh:      make(chan error, 1),
		onLog:      onLog,
		requestTimes: make(map[uint64]time.Time),
	}
}

func (c *Client) log(format string, args ...interface{}) {
	if c.onLog != nil {
		c.onLog(fmt.Sprintf(format, args...))
	}
}

// Connect establishes connection and performs subscribe + authorize handshake.
func (c *Client) Connect(ctx context.Context) error {
	return c.connectWithRetry(ctx)
}

func (c *Client) connectWithRetry(ctx context.Context) error {
	backoff := 1 * time.Second
	maxBackoff := 60 * time.Second

	for {
		err := c.dialAndAuth(ctx)
		if err == nil {
			return nil
		}

		c.log("connection failed: %v, retrying in %v", err, backoff)
		
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}

func (c *Client) dialAndAuth(ctx context.Context) error {
	var d net.Dialer
	addr := string(c.addr.Bytes())
	if strings.HasPrefix(addr, "stratum+tcp://") {
		addr = addr[14:]
	} else if strings.HasPrefix(addr, "stratum://") {
		addr = addr[10:]
	}

	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return err
	}
	c.conn = conn
	c.scanner = bufio.NewScanner(conn)
	c.scanner.Buffer(make([]byte, 0, 32768), 32768)

	// Subscribe
	if err := c.subscribe(); err != nil {
		conn.Close()
		return fmt.Errorf("subscribe: %w", err)
	}

	// Authorize
	if err := c.authorize(); err != nil {
		conn.Close()
		return fmt.Errorf("authorize: %w", err)
	}

	c.log("stratum connected: %s (extranonce1=%s, en2_size=%d)",
		string(c.addr.Bytes()), hex.EncodeToString(c.extranonce1), c.extranonce2Len)

	return nil
}

// JobChannel returns the channel that receives new mining jobs.
func (c *Client) JobChannel() <-chan *Job {
	return c.jobCh
}

// ErrChannel returns the channel that receives connection errors.
func (c *Client) ErrChannel() <-chan error {
	return c.errCh
}

// CurrentDifficulty returns the current share difficulty.
func (c *Client) CurrentDifficulty() float64 {
	c.diffMu.RLock()
	defer c.diffMu.RUnlock()
	return c.difficulty
}

// LastLatency returns the RTT of the last server response.
func (c *Client) LastLatency() time.Duration {
	return time.Duration(c.lastLatency.Load())
}

// Run starts the read loop that processes server messages. Blocks until ctx
// is cancelled or the connection is closed.
func (c *Client) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		for c.scanner.Scan() {
			line := c.scanner.Text()
			if line == "" {
				continue
			}
			c.handleMessage(line)
		}

		if ctx.Err() != nil {
			return
		}

		// Connection lost, try to reconnect.
		c.log("connection lost, reconnecting...")
		if err := c.connectWithRetry(ctx); err != nil {
			select {
			case c.errCh <- err:
			default:
			}
			return
		}
	}
}

// Close shuts down the connection.
func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

// SubmitShare submits a found share to the server.
func (c *Client) SubmitShare(jobID string, extranonce2 []byte, ntime uint32, nonce uint32) error {
	c.Stats.SharesSubmitted.Add(1)

	ntimeHex := fmt.Sprintf("%08x", ntime)
	nonceHex := encodeUint32LE(nonce)
	en2Hex := hex.EncodeToString(extranonce2)

	id := c.nextID()
	c.workerMu.RLock()
	name := string(c.workerName.Bytes())
	c.workerMu.RUnlock()

	req := map[string]interface{}{
		"id":     id,
		"method": "mining.submit",
		"params": []string{
			name,
			jobID,
			en2Hex,
			ntimeHex,
			nonceHex,
		},
	}

	return c.sendJSON(req)
}

// --- Protocol helpers ---

type stratumMsg struct {
	ID     interface{}     `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
	Result json.RawMessage `json:"result"`
	Error  json.RawMessage `json:"error"`
}

func (c *Client) handleMessage(line string) {
	var msg stratumMsg
	if err := json.Unmarshal([]byte(line), &msg); err != nil {
		return
	}

	// Measure latency if this is a response to a tracked ID
	if msg.ID != nil {
		id := parseStratumID(msg.ID)
		c.writeMu.Lock()
		if start, ok := c.requestTimes[id]; ok {
			c.lastLatency.Store(int64(time.Since(start)))
			delete(c.requestTimes, id)
		}
		c.writeMu.Unlock()
	}

	// Server notification (no id or null id).
	if msg.Method != "" {
		switch msg.Method {
		case "mining.notify":
			c.handleNotify(msg.Params)
		case "mining.set_difficulty":
			c.handleSetDifficulty(msg.Params)
		}
		return
	}

	// Response to our submit.
	if msg.Result != nil {
		var result bool
		if err := json.Unmarshal(msg.Result, &result); err == nil {
			if result {
				c.Stats.SharesAccepted.Add(1)
			} else {
				c.Stats.SharesRejected.Add(1)
			}
		}
	}
	if msg.Error != nil && string(msg.Error) != "null" {
		var errArr []interface{}
		if err := json.Unmarshal(msg.Error, &errArr); err == nil && len(errArr) >= 2 {
			errMsg, _ := errArr[1].(string)
			if errMsg != "" {
				// Check if it's a stale share.
				if errMsg == "job not found" {
					c.Stats.SharesStale.Add(1)
				}
				c.Stats.SharesRejected.Add(1)
			}
		}
	}
}

func (c *Client) handleNotify(params json.RawMessage) {
	var p []json.RawMessage
	if err := json.Unmarshal(params, &p); err != nil || len(p) < 9 {
		return
	}

	var jobID, prevHashHex, coinbase1Hex, coinbase2Hex string
	var merkleBranchHex []string
	var versionHex, bitsHex, ntimeHex string
	var cleanJobs bool

	json.Unmarshal(p[0], &jobID)
	json.Unmarshal(p[1], &prevHashHex)
	json.Unmarshal(p[2], &coinbase1Hex)
	json.Unmarshal(p[3], &coinbase2Hex)
	json.Unmarshal(p[4], &merkleBranchHex)
	json.Unmarshal(p[5], &versionHex)
	json.Unmarshal(p[6], &bitsHex)
	json.Unmarshal(p[7], &ntimeHex)
	json.Unmarshal(p[8], &cleanJobs)

	// Decode prevhash: stratum sends 4-byte-swapped groups.
	prevHash := decodeStratumPrevhash(prevHashHex)

	cb1, _ := hex.DecodeString(coinbase1Hex)
	cb2, _ := hex.DecodeString(coinbase2Hex)

	var branch []types.Hash
	for _, h := range merkleBranchHex {
		b, _ := hex.DecodeString(h)
		if len(b) == 32 {
			var hash types.Hash
			copy(hash[:], b)
			branch = append(branch, hash)
		}
	}

	version := decodeHexUint32BE(versionHex)
	bits := decodeHexUint32BE(bitsHex)
	ntime := decodeHexUint32BE(ntimeHex)

	netTarget := CompactToHash(bits)

	c.diffMu.RLock()
	diff := c.difficulty
	c.diffMu.RUnlock()
	shareTarget := DifficultyToTarget(diff)

	job := &Job{
		ID:           jobID,
		PrevHash:     prevHash,
		Coinbase1:    cb1,
		Coinbase2:    cb2,
		MerkleBranch: branch,
		Version:      version,
		Bits:         bits,
		NTime:        ntime,
		CleanJobs:    cleanJobs,
		Target:       shareTarget,
		NetTarget:    netTarget,
	}

	c.jobMu.Lock()
	c.currentJob = job
	c.jobMu.Unlock()

	// Non-blocking send to job channel.
	select {
	case c.jobCh <- job:
	default:
		// Drain old and send new.
		select {
		case <-c.jobCh:
		default:
		}
		c.jobCh <- job
	}
}

func (c *Client) handleSetDifficulty(params json.RawMessage) {
	var p []interface{}
	if err := json.Unmarshal(params, &p); err != nil || len(p) < 1 {
		return
	}

	var newDiff float64
	switch v := p[0].(type) {
	case float64:
		newDiff = v
	case json.Number:
		newDiff, _ = v.Float64()
	}

	if newDiff > 0 {
		c.diffMu.Lock()
		c.difficulty = newDiff
		c.diffMu.Unlock()
		c.log("difficulty set: %.6f", newDiff)
	}
}

func (c *Client) subscribe() error {
	id := c.nextID()
	req := map[string]interface{}{
		"id":     id,
		"method": "mining.subscribe",
		"params": []string{fmt.Sprintf("fairchain-miner/0.1.0")},
	}
	if err := c.sendJSON(req); err != nil {
		return err
	}

	// Read response.
	if !c.scanner.Scan() {
		return fmt.Errorf("no subscribe response")
	}

	var resp stratumMsg
	if err := json.Unmarshal(c.scanner.Bytes(), &resp); err != nil {
		return fmt.Errorf("parse subscribe response: %w", err)
	}

	// Parse result: [[["mining.set_difficulty","1"],["mining.notify","1"]], "extranonce1", extranonce2_size]
	var result []json.RawMessage
	if err := json.Unmarshal(resp.Result, &result); err != nil || len(result) < 3 {
		return fmt.Errorf("invalid subscribe result")
	}

	var en1Hex string
	json.Unmarshal(result[1], &en1Hex)
	c.extranonce1, _ = hex.DecodeString(en1Hex)

	var en2Size int
	json.Unmarshal(result[2], &en2Size)
	c.extranonce2Len = en2Size

	// Handle set_difficulty that may come before subscribe response.
	// Process any pending messages.
	return nil
}

// Reauthorize updates the worker credentials and triggers a reconnection.
// This is the most robust way to swap identities as it avoids race conditions
// with the background message handler and ensures a fresh session with the pool.
func (c *Client) Reauthorize(workerName, password string) error {
	c.workerMu.Lock()
	if c.workerName != nil {
		c.workerName.Destroy()
	}
	if c.password != nil {
		c.password.Destroy()
	}
	c.workerName = memguard.NewBufferFromBytes([]byte(workerName))
	c.password = memguard.NewBufferFromBytes([]byte(password))
	c.workerMu.Unlock()

	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) authorize() error {
	id := c.nextID()
	c.workerMu.RLock()
	params := []string{string(c.workerName.Bytes()), string(c.password.Bytes())}
	c.workerMu.RUnlock()

	req := map[string]interface{}{
		"id":     id,
		"method": "mining.authorize",
		"params": params,
	}
	if err := c.sendJSON(req); err != nil {
		return err
	}

	// Read messages until we get the authorize response.
	// (set_difficulty and notify may come before auth response)
	for i := 0; i < 10; i++ {
		if !c.scanner.Scan() {
			return fmt.Errorf("connection closed during authorize")
		}
		line := c.scanner.Text()

		var msg stratumMsg
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			continue
		}

		// Process notifications that arrive before the auth response.
		if msg.Method != "" {
			c.handleMessage(line)
			continue
		}

		// This should be our auth response.
		if msg.Result != nil {
			var result bool
			json.Unmarshal(msg.Result, &result)
			if !result {
				return fmt.Errorf("authorization rejected")
			}
			return nil
		}
	}

	return fmt.Errorf("authorize timeout")
}

func (c *Client) nextID() uint64 {
	return c.msgID.Add(1)
}

func parseStratumID(id interface{}) uint64 {
	switch v := id.(type) {
	case float64:
		return uint64(v)
	case uint64:
		return v
	}
	return 0
}

func (c *Client) sendJSON(v interface{}) error {
	msgMap, ok := v.(map[string]interface{})
	if ok {
		if id, ok := msgMap["id"].(uint64); ok {
			c.writeMu.Lock()
			c.requestTimes[id] = time.Now()
			c.writeMu.Unlock()
		}
	}

	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	_, err = c.conn.Write(data)
	return err
}

// --- Mining helpers ---

// BuildHeader constructs a block header from a job, extranonce2 and nonce.
func (c *Client) BuildHeader(job *Job, extranonce2 []byte, nonce uint32) [80]byte {
	// Build coinbase: coinbase1 + extranonce1 + extranonce2 + coinbase2
	coinbase := make([]byte, 0, len(job.Coinbase1)+len(c.extranonce1)+len(extranonce2)+len(job.Coinbase2))
	coinbase = append(coinbase, job.Coinbase1...)
	coinbase = append(coinbase, c.extranonce1...)
	coinbase = append(coinbase, extranonce2...)
	coinbase = append(coinbase, job.Coinbase2...)

	// Hash coinbase.
	cbHash := doubleSHA256(coinbase)

	// Compute merkle root.
	merkleRoot := cbHash
	for _, branch := range job.MerkleBranch {
		var combined [64]byte
		copy(combined[:32], merkleRoot[:])
		copy(combined[32:], branch[:])
		merkleRoot = doubleSHA256(combined[:])
	}

	// Merkle root needs to be reversed for the header (LE→BE for stratum).
	merkleRootBE := reversed(merkleRoot)

	// Build header.
	var hdr [80]byte
	binary.LittleEndian.PutUint32(hdr[0:4], job.Version)
	copy(hdr[4:36], job.PrevHash[:])
	copy(hdr[36:68], merkleRootBE[:])
	binary.LittleEndian.PutUint32(hdr[68:72], job.NTime)
	binary.LittleEndian.PutUint32(hdr[72:76], job.Bits)
	binary.LittleEndian.PutUint32(hdr[76:80], nonce)

	return hdr
}

// Extranonce1 returns the assigned extranonce1.
func (c *Client) Extranonce1() []byte {
	return c.extranonce1
}

// Extranonce2Len returns the extranonce2 byte size.
func (c *Client) Extranonce2Len() int {
	return c.extranonce2Len
}

// NextExtranonce2 generates a unique extranonce2 for a worker.
func (c *Client) NextExtranonce2() []byte {
	size := c.extranonce2Len
	en2 := make([]byte, size)
	val := c.en2ID.Add(1)
	
	// Write exactly as many bytes as available in the buffer
	for i := 0; i < size; i++ {
		en2[size-1-i] = byte(val >> (i * 8))
	}
	
	return en2
}

// --- Utility functions ---

func doubleSHA256(data []byte) types.Hash {
	first := sha256.Sum256(data)
	second := sha256.Sum256(first[:])
	var h types.Hash
	copy(h[:], second[:])
	// Return in LE internal byte order (reversed from big-endian SHA output).
	for i, j := 0, 31; i < j; i, j = i+1, j-1 {
		h[i], h[j] = h[j], h[i]
	}
	return h
}

func reversed(h types.Hash) types.Hash {
	var r types.Hash
	for i := 0; i < 32; i++ {
		r[i] = h[31-i]
	}
	return r
}

func decodeStratumPrevhash(hexStr string) [32]byte {
	raw, _ := hex.DecodeString(hexStr)
	var out [32]byte
	if len(raw) == 32 {
		// Reverse each 4-byte group (undo stratum's byte-swap encoding).
		for i := 0; i < 32; i += 4 {
			out[i+0] = raw[i+3]
			out[i+1] = raw[i+2]
			out[i+2] = raw[i+1]
			out[i+3] = raw[i+0]
		}
	}
	return out
}

func decodeHexUint32BE(s string) uint32 {
	b, _ := hex.DecodeString(s)
	if len(b) != 4 {
		return 0
	}
	return binary.BigEndian.Uint32(b)
}

func encodeUint32LE(v uint32) string {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], v)
	return hex.EncodeToString(buf[:])
}

// CompactToHash converts Bitcoin compact bits to a 256-bit target hash.
func CompactToHash(compact uint32) types.Hash {
	mantissa := compact & 0x007fffff
	exponent := compact >> 24

	var target big.Int
	if exponent <= 3 {
		mantissa >>= 8 * (3 - exponent)
		target.SetInt64(int64(mantissa))
	} else {
		target.SetInt64(int64(mantissa))
		target.Lsh(&target, 8*(uint(exponent)-3))
	}

	b := target.Bytes()
	var h types.Hash
	for i, j := 0, len(b)-1; j >= 0; i, j = i+1, j-1 {
		if i >= 32 {
			break
		}
		h[i] = b[j]
	}
	return h
}

// DifficultyToTarget converts a relative difficulty value to a target hash.
func DifficultyToTarget(diff float64) types.Hash {
	if diff <= 0 {
		diff = 1
	}
	invDiff := new(big.Float).SetPrec(256).Quo(
		new(big.Float).SetPrec(256).SetFloat64(1.0),
		new(big.Float).SetPrec(256).SetFloat64(diff),
	)
	exp96 := new(big.Float).SetPrec(256).SetInt(new(big.Int).Lsh(big.NewInt(1), 96))
	high128f := new(big.Float).SetPrec(256).Mul(invDiff, exp96)
	high128, _ := high128f.Int(nil)

	max128 := new(big.Int).Lsh(big.NewInt(1), 128)
	max128.Sub(max128, big.NewInt(1))
	if high128.Cmp(max128) > 0 {
		high128.Set(max128)
	}

	var h types.Hash
	for i := 0; i < 16; i++ {
		h[i] = 0xFF
	}
	high128Bytes := high128.Bytes()
	for i := 0; i < len(high128Bytes) && i < 16; i++ {
		h[16+(15-i)] = high128Bytes[i]
	}
	return h
}

// HashToDifficulty calculates the relative difficulty from a target hash.
func HashToDifficulty(h types.Hash) float64 {
	var high128 big.Int
	b := make([]byte, 16)
	for i := 0; i < 16; i++ {
		b[i] = h[16+(15-i)]
	}
	high128.SetBytes(b)

	if high128.Sign() == 0 {
		return 0
	}

	exp96 := new(big.Float).SetPrec(256).SetInt(new(big.Int).Lsh(big.NewInt(1), 96))
	valf := new(big.Float).SetPrec(256).SetInt(&high128)

	diff, _ := new(big.Float).SetPrec(256).Quo(exp96, valf).Float64()
	return diff
}
// ValidHashRaw replicates cpuminer's raw hash comparison.
func ValidHashRaw(hash, target types.Hash) bool {
	for i := 7; i >= 0; i-- {
		h := binary.LittleEndian.Uint32(hash[i*4 : i*4+4])
		t := binary.LittleEndian.Uint32(target[i*4 : i*4+4])
		if h > t {
			return false
		}
		if h < t {
			return true
		}
	}
	return true
}
