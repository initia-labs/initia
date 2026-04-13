package quickstart

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	cmtcfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/p2p/pex"
)

var httpClient = &http.Client{
	Timeout: 60 * time.Second,
}

type networkConfig struct {
	GenesisURL   string
	AddrbookURL  string
	StateSyncRPC string
	LivePeersAPI string
	RPCURL       string // public RPC endpoint for fallback addrbook generation
}

var networks = map[string]networkConfig{
	NetworkMainnet: {
		GenesisURL:   "https://snapshots.polkachu.com/genesis/initia/genesis.json",
		AddrbookURL:  "https://snapshots.polkachu.com/addrbook/initia/addrbook.json",
		StateSyncRPC: "https://initia-rpc.polkachu.com:443",
		LivePeersAPI: "https://polkachu.com/api/v2/chains/initia/live_peers",
		RPCURL:       "https://rpc.initia.xyz",
	},
	NetworkTestnet: {
		GenesisURL:   "https://snapshots.polkachu.com/testnet-genesis/initia/genesis.json",
		AddrbookURL:  "https://snapshots.polkachu.com/testnet-addrbook/initia/addrbook.json",
		StateSyncRPC: "https://initia-testnet-rpc.polkachu.com:443",
		LivePeersAPI: "https://polkachu.com/api/v2/chains/initia-testnet/live_peers",
		RPCURL:       "https://rpc.testnet.initia.xyz",
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
	if err := downloadFile(nc.AddrbookURL, destPath); err != nil {
		fmt.Printf("Pre-built addrbook download failed (%v), building from RPC peers...\n", err)
		return buildAddrbookFromRPC(nc.RPCURL, destPath)
	}
	return nil
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

	// Limit download to 500MB to prevent disk exhaustion
	if _, err = io.Copy(out, io.LimitReader(resp.Body, 500<<20)); err != nil {
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



// buildAddrbookFromRPC fetches peers from the RPC node's /net_info and builds
// addrbook.json using CometBFT's pex.AddrBook API.
func buildAddrbookFromRPC(rpcURL, destPath string) error {
	// Fetch peer addresses from /net_info
	peerAddrs, err := fetchNetInfoPeerAddrs(rpcURL)
	if err != nil {
		return fmt.Errorf("failed to fetch net_info: %w", err)
	}
	if len(peerAddrs) == 0 {
		return fmt.Errorf("RPC %s/net_info returned no peers", rpcURL)
	}

	// Fetch RPC node address for the "src" field
	srcAddr, err := fetchRPCNodeAddr(rpcURL)
	if err != nil {
		return fmt.Errorf("failed to fetch RPC node status: %w", err)
	}

	// Create addrbook using CometBFT's pex API
	book := pex.NewAddrBook(destPath, false)
	book.SetLogger(log.NewNopLogger())

	for _, addr := range peerAddrs {
		if err := book.AddAddress(addr, srcAddr); err != nil {
			// Skip peers that fail validation (e.g., non-routable IPs)
			continue
		}
	}

	book.Save()
	return nil
}

// fetchNetInfoPeerAddrs calls /net_info and returns parsed p2p.NetAddress entries.
func fetchNetInfoPeerAddrs(rpcURL string) ([]*p2p.NetAddress, error) {
	resp, err := httpClient.Get(rpcURL + "/net_info")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s/net_info returned status %d", rpcURL, resp.StatusCode)
	}

	var result struct {
		Result struct {
			Peers []struct {
				NodeInfo struct {
					ID         string `json:"id"`
					ListenAddr string `json:"listen_addr"`
				} `json:"node_info"`
				RemoteIP string `json:"remote_ip"`
			} `json:"peers"`
		} `json:"result"`
	}

	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&result); err != nil {
		return nil, err
	}

	addrs := make([]*p2p.NetAddress, 0, len(result.Result.Peers))
	for _, peer := range result.Result.Peers {
		if peer.RemoteIP == "" || peer.NodeInfo.ID == "" {
			continue
		}
		port, err := parsePortFromListenAddr(peer.NodeInfo.ListenAddr)
		if err != nil {
			continue
		}
		addr := p2p.NewNetAddressIPPort(net.ParseIP(peer.RemoteIP), port)
		addr.ID = p2p.ID(peer.NodeInfo.ID)
		addrs = append(addrs, addr)
	}

	return addrs, nil
}

// fetchRPCNodeAddr calls /status and resolves the RPC hostname to build
// the source p2p.NetAddress for addrbook entries.
func fetchRPCNodeAddr(rpcURL string) (*p2p.NetAddress, error) {
	resp, err := httpClient.Get(rpcURL + "/status")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s/status returned status %d", rpcURL, resp.StatusCode)
	}

	var result struct {
		Result struct {
			NodeInfo struct {
				ID         string `json:"id"`
				ListenAddr string `json:"listen_addr"`
			} `json:"node_info"`
		} `json:"result"`
	}

	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&result); err != nil {
		return nil, err
	}

	port, err := parsePortFromListenAddr(result.Result.NodeInfo.ListenAddr)
	if err != nil {
		port = 26656
	}

	// Resolve the RPC URL hostname to get the IP
	parsed, err := url.Parse(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RPC URL %q: %w", rpcURL, err)
	}

	host := parsed.Hostname()
	ip := net.ParseIP(host)
	if ip == nil {
		ips, err := net.LookupHost(host)
		if err != nil || len(ips) == 0 {
			return nil, fmt.Errorf("failed to resolve RPC hostname %q: %w", host, err)
		}
		ip = net.ParseIP(ips[0])
	}

	addr := p2p.NewNetAddressIPPort(ip, port)
	addr.ID = p2p.ID(result.Result.NodeInfo.ID)
	return addr, nil
}

// parsePortFromListenAddr extracts the port number from a CometBFT listen address
// like "tcp://0.0.0.0:26656" or "0.0.0.0:26656".
func parsePortFromListenAddr(listenAddr string) (uint16, error) {
	addr := listenAddr
	if idx := strings.Index(addr, "://"); idx >= 0 {
		addr = addr[idx+3:]
	}

	_, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse listen_addr %q: %w", listenAddr, err)
	}

	port, err := strconv.ParseUint(portStr, 10, 16)
	if err != nil {
		return 0, fmt.Errorf("invalid port in listen_addr %q: %w", listenAddr, err)
	}

	return uint16(port), nil
}
