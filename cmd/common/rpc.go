package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/bams-repo/fairchain/internal/types"
)

// ChainInfo contains node status information
type ChainInfo struct {
	Height   uint32 `json:"blocks"`
	BestHash string `json:"bestblockhash"`
	Bits     string `json:"bits"`
	Chain    string `json:"chain"`
}

// BlockInfo contains block header information
type BlockInfo struct {
	Hash      string `json:"hash"`
	Height    uint32 `json:"height"`
	Timestamp uint32 `json:"time"`
	Bits      string `json:"bits"`
	Nonce     uint32 `json:"nonce"`
}

// FetchChainInfo gets blockchain status from RPC
func FetchChainInfo(rpc string) (*ChainInfo, error) {
	resp, err := http.Get(rpc + "/getblockchaininfo")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var info ChainInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}

// FetchBlockByHeight gets block header at given height
func FetchBlockByHeight(rpc string, height uint32) (*BlockInfo, error) {
	resp, err := http.Get(fmt.Sprintf("%s/getblockbyheight?height=%d", rpc, height))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var info BlockInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, err
	}
	return &info, nil
}

// SubmitBlock submits solved block to network
func SubmitBlock(rpc string, block *types.Block) (rejected bool, detail string) {
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