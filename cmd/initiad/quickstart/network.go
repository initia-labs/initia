package quickstart

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

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
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download %s: status %d", url, resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", destPath, err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func setupStateSync(network, homeDir string) error {
	nc := networks[network]

	latestHeight, err := fetchLatestHeight(nc.StateSyncRPC)
	if err != nil {
		return fmt.Errorf("failed to fetch latest height: %w", err)
	}

	trustHeight := latestHeight - 2000

	trustHash, err := fetchBlockHash(nc.StateSyncRPC, trustHeight)
	if err != nil {
		return fmt.Errorf("failed to fetch trust hash: %w", err)
	}

	polkachuPeer, err := fetchPolkachuPeer(nc.LivePeersAPI)
	if err != nil {
		return fmt.Errorf("failed to fetch polkachu peer: %w", err)
	}

	return applyStateSync(homeDir, nc.StateSyncRPC, trustHeight, trustHash, polkachuPeer)
}

func fetchLatestHeight(rpc string) (int64, error) {
	resp, err := http.Get(rpc + "/abci_info")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result struct {
		Result struct {
			Response struct {
				LastBlockHeight string `json:"last_block_height"`
			} `json:"response"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, err
	}

	var height int64
	_, err = fmt.Sscanf(result.Result.Response.LastBlockHeight, "%d", &height)
	return height, err
}

func fetchBlockHash(rpc string, height int64) (string, error) {
	url := fmt.Sprintf("%s/block?height=%d", rpc, height)
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Result struct {
			BlockID struct {
				Hash string `json:"hash"`
			} `json:"block_id"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Result.BlockID.Hash, nil
}

type livePeersResponse struct {
	PolkachuPeer string `json:"polkachu_peer"`
}

func fetchPolkachuPeer(apiURL string) (string, error) {
	resp, err := http.Get(apiURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result livePeersResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.PolkachuPeer, nil
}

func applyStateSync(homeDir, rpc string, trustHeight int64, trustHash, polkachuPeer string) error {
	configPath := filepath.Join(homeDir, "config", "config.toml")

	vpr := viper.New()
	vpr.SetConfigFile(configPath)
	if err := vpr.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config.toml: %w", err)
	}

	vpr.Set("statesync.enable", true)
	vpr.Set("statesync.rpc_servers", rpc+","+rpc)
	vpr.Set("statesync.trust_height", trustHeight)
	vpr.Set("statesync.trust_hash", trustHash)

	existingPeers := vpr.GetString("p2p.persistent_peers")
	if !strings.Contains(existingPeers, polkachuPeer) {
		if existingPeers != "" {
			existingPeers += ","
		}
		existingPeers += polkachuPeer
		vpr.Set("p2p.persistent_peers", existingPeers)
	}

	return vpr.WriteConfig()
}
