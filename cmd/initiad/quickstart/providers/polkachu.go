package providers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// HTTPClient is the shared HTTP client with timeout for all provider requests.
var HTTPClient = &http.Client{
	Timeout: 60 * time.Second,
}

// PolkachuConfig holds Polkachu endpoint URLs for a specific network.
type PolkachuConfig struct {
	GenesisURL   string
	AddrbookURL  string
	StateSyncRPC string
	LivePeersAPI string
}

// Polkachu endpoint configurations per network.
var Polkachu = map[string]PolkachuConfig{
	"mainnet": {
		GenesisURL:   "https://snapshots.polkachu.com/genesis/initia/genesis.json",
		AddrbookURL:  "https://snapshots.polkachu.com/addrbook/initia/addrbook.json",
		StateSyncRPC: "https://initia-rpc.polkachu.com:443",
		LivePeersAPI: "https://polkachu.com/api/v2/chains/initia/live_peers",
	},
	"testnet": {
		GenesisURL:   "https://snapshots.polkachu.com/testnet-genesis/initia/genesis.json",
		AddrbookURL:  "https://snapshots.polkachu.com/testnet-addrbook/initia/addrbook.json",
		StateSyncRPC: "https://initia-testnet-rpc.polkachu.com:443",
		LivePeersAPI: "https://polkachu.com/api/v2/chains/initia-testnet/live_peers",
	},
}

// FetchStateSyncPeer fetches the state sync peer address from Polkachu's live_peers API.
func FetchStateSyncPeer(apiURL string) (string, error) {
	resp, err := HTTPClient.Get(apiURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API %s returned status %d", apiURL, resp.StatusCode)
	}

	var result struct {
		PolkachuPeer string `json:"polkachu_peer"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&result); err != nil {
		return "", err
	}

	return result.PolkachuPeer, nil
}

// FetchLatestHeight fetches the latest block height from the given RPC endpoint.
func FetchLatestHeight(rpc string) (int64, error) {
	resp, err := HTTPClient.Get(rpc + "/abci_info")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("RPC %s/abci_info returned status %d", rpc, resp.StatusCode)
	}

	var result struct {
		Result struct {
			Response struct {
				LastBlockHeight string `json:"last_block_height"`
			} `json:"response"`
		} `json:"result"`
	}

	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&result); err != nil {
		return 0, err
	}

	var height int64
	_, err = fmt.Sscanf(result.Result.Response.LastBlockHeight, "%d", &height)
	return height, err
}

// FetchBlockHash fetches the block hash at the given height from the given RPC endpoint.
func FetchBlockHash(rpc string, height int64) (string, error) {
	url := fmt.Sprintf("%s/block?height=%d", rpc, height)
	resp, err := HTTPClient.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("RPC %s returned status %d", url, resp.StatusCode)
	}

	var result struct {
		Result struct {
			BlockID struct {
				Hash string `json:"hash"`
			} `json:"block_id"`
		} `json:"result"`
	}

	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&result); err != nil {
		return "", err
	}

	return result.Result.BlockID.Hash, nil
}

// DownloadFile downloads a URL to destPath atomically (via temp file + rename).
func DownloadFile(url, destPath string) error {
	resp, err := HTTPClient.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download %s: status %d", url, resp.StatusCode)
	}

	return WriteAtomically(destPath, io.LimitReader(resp.Body, 500<<20))
}

// WriteAtomically writes data from a reader to destPath via a temp file + rename.
func WriteAtomically(destPath string, r io.Reader) error {
	tmpPath := destPath + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", tmpPath, err)
	}
	defer os.Remove(tmpPath)

	if _, err = io.Copy(out, r); err != nil {
		out.Close()
		return fmt.Errorf("failed to write %s: %w", destPath, err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("failed to close %s: %w", destPath, err)
	}

	return os.Rename(tmpPath, destPath)
}
