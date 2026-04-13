package quickstart

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	cmtcfg "github.com/cometbft/cometbft/config"
	"github.com/spf13/viper"
)

var httpClient = &http.Client{
	Timeout: 60 * time.Second,
}

type networkConfig struct {
	GenesisURL   string
	AddrbookURL  string
	StateSyncRPC string
	LivePeersAPI string
}

var networks = map[string]networkConfig{
	NetworkMainnet: {
		GenesisURL:   "https://snapshots.polkachu.com/genesis/initia/genesis.json",
		AddrbookURL:  "https://snapshots.polkachu.com/addrbook/initia/addrbook.json",
		StateSyncRPC: "https://initia-rpc.polkachu.com:443",
		LivePeersAPI: "https://polkachu.com/api/v2/chains/initia/live_peers",
	},
	NetworkTestnet: {
		GenesisURL:   "https://snapshots.polkachu.com/testnet-genesis/initia/genesis.json",
		AddrbookURL:  "https://snapshots.polkachu.com/testnet-addrbook/initia/addrbook.json",
		StateSyncRPC: "https://initia-testnet-rpc.polkachu.com:443",
		LivePeersAPI: "https://polkachu.com/api/v2/chains/initia-testnet/live_peers",
	},
}

func downloadGenesis(network, homeDir string) error {
	nc := networks[network]
	destPath := filepath.Join(homeDir, "config", "genesis.json")
	return downloadFile(nc.GenesisURL, destPath)
}

func downloadAddrbook(network, homeDir string) error {
	nc := networks[network]
	destPath := filepath.Join(homeDir, "config", "addrbook.json")
	return downloadFile(nc.AddrbookURL, destPath)
}

func downloadFile(url, destPath string) error {
	resp, err := httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download %s: status %d", url, resp.StatusCode)
	}

	// Write to temp file and rename atomically to avoid corrupt files on failure
	tmpPath := destPath + ".tmp"
	out, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", tmpPath, err)
	}
	defer os.Remove(tmpPath)

	if _, err = io.Copy(out, resp.Body); err != nil {
		out.Close()
		return fmt.Errorf("failed to write %s: %w", destPath, err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("failed to close %s: %w", destPath, err)
	}

	return os.Rename(tmpPath, destPath)
}

func setupStateSync(network, homeDir string) error {
	nc := networks[network]

	latestHeight, err := fetchLatestHeight(nc.StateSyncRPC)
	if err != nil {
		return fmt.Errorf("failed to fetch latest height: %w", err)
	}

	trustHeight := latestHeight - 2000
	if trustHeight < 1 {
		trustHeight = 1
	}

	trustHash, err := fetchBlockHash(nc.StateSyncRPC, trustHeight)
	if err != nil {
		return fmt.Errorf("failed to fetch trust hash: %w", err)
	}
	if trustHash == "" {
		return fmt.Errorf("RPC returned empty block hash for height %d; the block may have been pruned", trustHeight)
	}

	polkachuPeer, err := fetchPolkachuPeer(nc.LivePeersAPI)
	if err != nil {
		return fmt.Errorf("failed to fetch polkachu peer: %w", err)
	}
	if polkachuPeer == "" {
		return fmt.Errorf("polkachu API returned empty peer")
	}

	return applyStateSync(homeDir, nc.StateSyncRPC, trustHeight, trustHash, polkachuPeer)
}

func fetchLatestHeight(rpc string) (int64, error) {
	resp, err := httpClient.Get(rpc + "/abci_info")
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

func fetchBlockHash(rpc string, height int64) (string, error) {
	url := fmt.Sprintf("%s/block?height=%d", rpc, height)
	resp, err := httpClient.Get(url)
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

type livePeersResponse struct {
	PolkachuPeer string `json:"polkachu_peer"`
}

func fetchPolkachuPeer(apiURL string) (string, error) {
	resp, err := httpClient.Get(apiURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API %s returned status %d", apiURL, resp.StatusCode)
	}

	var result livePeersResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&result); err != nil {
		return "", err
	}

	return result.PolkachuPeer, nil
}

// applyStateSync loads config.toml via viper, modifies statesync/p2p fields,
// then writes back using CometBFT's WriteConfigFile to preserve comments and formatting.
func applyStateSync(homeDir, rpc string, trustHeight int64, trustHash, polkachuPeer string) error {
	configPath := filepath.Join(homeDir, "config")
	configFile := filepath.Join(configPath, "config.toml")

	// Load config via viper and unmarshal into CometBFT config struct
	cfg, err := loadCometConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config.toml: %w", err)
	}

	// Set statesync config
	cfg.StateSync.Enable = true
	cfg.StateSync.RPCServers = []string{rpc, rpc}
	cfg.StateSync.TrustHeight = trustHeight
	cfg.StateSync.TrustHash = trustHash

	// Append polkachu peer to persistent_peers if not already present
	if !strings.Contains(cfg.P2P.PersistentPeers, polkachuPeer) {
		if cfg.P2P.PersistentPeers != "" {
			cfg.P2P.PersistentPeers += ","
		}
		cfg.P2P.PersistentPeers += polkachuPeer
	}

	// Write back using CometBFT template to preserve comments
	cmtcfg.WriteConfigFile(configFile, cfg)
	return nil
}

// loadCometConfig reads config.toml via viper and unmarshals into a CometBFT Config struct.
func loadCometConfig(configPath string) (*cmtcfg.Config, error) {
	v := viper.New()
	v.SetConfigType("toml")
	v.SetConfigName("config")
	v.AddConfigPath(configPath)

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	cfg := cmtcfg.DefaultConfig()
	if err := v.Unmarshal(cfg); err != nil {
		return nil, err
	}

	cfg.SetRoot(filepath.Dir(configPath))
	return cfg, nil
}
