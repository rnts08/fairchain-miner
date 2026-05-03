// Copyright (c) 2024-2026 The Fairchain Contributors
// Distributed under the MIT software license, see the accompanying
// file COPYING or http://www.opensource.org/licenses/mit-license.php.

// Package rpc provides an HTTP client for communicating with the Fairchain node.
// The miner fetches chain state and submits solved blocks exclusively through
// this interface — no direct chain/mempool/p2p imports.
package rpc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is an HTTP client for the Fairchain node's REST API.
type Client struct {
	addr       string
	httpClient *http.Client
}

// NewClient creates a new RPC client targeting the given node address.
func NewClient(addr string) *Client {
	return &Client{
		addr: addr,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// ChainInfo holds the response from getblockchaininfo.
type ChainInfo struct {
	Height   uint32 `json:"blocks"`
	BestHash string `json:"bestblockhash"`
	Bits     string `json:"bits"`
	Chain    string `json:"chain"`
}

// BlockInfo holds the response from getblockbyheight.
type BlockInfo struct {
	Hash      string `json:"hash"`
	Height    uint32 `json:"height"`
	Timestamp uint32 `json:"time"`
	Bits      string `json:"bits"`
	Nonce     uint32 `json:"nonce"`
}

// GetBlockchainInfo fetches the current chain state.
func (c *Client) GetBlockchainInfo() (*ChainInfo, error) {
	resp, err := c.httpClient.Get(c.addr + "/getblockchaininfo")
	if err != nil {
		return nil, fmt.Errorf("getblockchaininfo: %w", err)
	}
	defer resp.Body.Close()

	var info ChainInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("getblockchaininfo decode: %w", err)
	}
	return &info, nil
}

// GetBlockByHeight fetches block info at the given height.
func (c *Client) GetBlockByHeight(height uint32) (*BlockInfo, error) {
	resp, err := c.httpClient.Get(fmt.Sprintf("%s/getblockbyheight?height=%d", c.addr, height))
	if err != nil {
		return nil, fmt.Errorf("getblockbyheight: %w", err)
	}
	defer resp.Body.Close()

	var info BlockInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("getblockbyheight decode: %w", err)
	}
	return &info, nil
}

// SubmitBlock submits a serialized block to the node.
// Returns (rejected, detail). If rejected is false, the block was accepted.
func (c *Client) SubmitBlock(blockData []byte) (rejected bool, detail string) {
	resp, err := c.httpClient.Post(
		c.addr+"/submitblock",
		"application/octet-stream",
		bytes.NewReader(blockData),
	)
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
