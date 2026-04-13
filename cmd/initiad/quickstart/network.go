package quickstart

import (
	"crypto/rand"
	"encoding/hex"
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
	"github.com/mitchellh/mapstructure"
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
	if err := v.Unmarshal(cfg, func(dc *mapstructure.DecoderConfig) {
		dc.Squash = true
	}); err != nil {
		return nil, err
	}

	cfg.SetRoot(filepath.Dir(configPath))
	return cfg, nil
}

// ── Fallback addrbook generation from RPC ──────────────────────────────────────
//
// When the pre-built addrbook download fails, we reconstruct addrbook.json by
// querying the public RPC node's /net_info (peer list) and /status (node identity).
//
// CometBFT addrbook format notes:
//   - "key": a random 20-byte hex string used internally for bucket hashing.
//     We generate a fresh random one; CometBFT accepts any valid hex string.
//   - "addr": the peer's reachable address. We use remote_ip from net_info
//     combined with the port parsed from node_info.listen_addr.
//   - "src": the node that told us about this peer (the RPC node itself).
//     We use result.node_info.id from /status and resolve the RPC hostname to get the IP.
//   - "bucket_type": 1 = "new" bucket (peers we haven't connected to yet).
//   - "buckets": [0] is a valid assignment; CometBFT rehashes on load.
//   - "last_attempt": set to current time to indicate freshly discovered peers.
//   - "last_success" / "last_ban_time": zero time.

// addrBook is the top-level CometBFT address book structure.
type addrBook struct {
	Key   string          `json:"key"`
	Addrs []addrBookEntry `json:"addrs"`
}

// addrBookEntry represents a single peer entry in the address book.
type addrBookEntry struct {
	Addr        netAddress `json:"addr"`
	Src         netAddress `json:"src"`
	Buckets     []int      `json:"buckets"`
	Attempts    int        `json:"attempts"`
	BucketType  int        `json:"bucket_type"`
	LastAttempt time.Time  `json:"last_attempt"`
	LastSuccess time.Time  `json:"last_success"`
	LastBanTime time.Time  `json:"last_ban_time"`
}

// netAddress is a CometBFT network address (node ID + IP + port).
type netAddress struct {
	ID   string `json:"id"`
	IP   string `json:"ip"`
	Port uint16 `json:"port"`
}

// buildAddrbookFromRPC fetches peers from the RPC node and writes addrbook.json.
func buildAddrbookFromRPC(rpcURL, destPath string) error {
	// Fetch peers from /net_info
	peers, err := fetchNetInfoPeers(rpcURL)
	if err != nil {
		return fmt.Errorf("failed to fetch net_info: %w", err)
	}
	if len(peers) == 0 {
		return fmt.Errorf("RPC %s/net_info returned no peers", rpcURL)
	}

	// Fetch RPC node identity for the "src" field
	src, err := fetchRPCNodeAddress(rpcURL)
	if err != nil {
		return fmt.Errorf("failed to fetch RPC node status: %w", err)
	}

	// Generate random 20-byte key
	keyBytes := make([]byte, 20)
	if _, err := rand.Read(keyBytes); err != nil {
		return fmt.Errorf("failed to generate addrbook key: %w", err)
	}

	now := time.Now()
	zeroTime := time.Time{}

	book := addrBook{
		Key:   hex.EncodeToString(keyBytes),
		Addrs: make([]addrBookEntry, 0, len(peers)),
	}

	for _, p := range peers {
		book.Addrs = append(book.Addrs, addrBookEntry{
			Addr:        p,
			Src:         src,
			Buckets:     []int{0},
			Attempts:    0,
			BucketType:  1,
			LastAttempt: now,
			LastSuccess: zeroTime,
			LastBanTime: zeroTime,
		})
	}

	data, err := json.MarshalIndent(book, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal addrbook: %w", err)
	}

	// Write atomically via temp file
	tmpPath := destPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write %s: %w", tmpPath, err)
	}
	defer os.Remove(tmpPath)

	return os.Rename(tmpPath, destPath)
}

// fetchNetInfoPeers calls /net_info and returns parsed peer addresses.
func fetchNetInfoPeers(rpcURL string) ([]netAddress, error) {
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

	addrs := make([]netAddress, 0, len(result.Result.Peers))
	for _, p := range result.Result.Peers {
		port, err := parsePortFromListenAddr(p.NodeInfo.ListenAddr)
		if err != nil {
			// Skip peers with unparseable listen addresses
			continue
		}
		if p.RemoteIP == "" || p.NodeInfo.ID == "" {
			continue
		}
		addrs = append(addrs, netAddress{
			ID:   p.NodeInfo.ID,
			IP:   p.RemoteIP,
			Port: port,
		})
	}

	return addrs, nil
}

// fetchRPCNodeAddress calls /status and resolves the RPC hostname to build
// the "src" address for addrbook entries.
func fetchRPCNodeAddress(rpcURL string) (netAddress, error) {
	// Fetch node ID and listen port from /status
	resp, err := httpClient.Get(rpcURL + "/status")
	if err != nil {
		return netAddress{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return netAddress{}, fmt.Errorf("%s/status returned status %d", rpcURL, resp.StatusCode)
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
		return netAddress{}, err
	}

	port, err := parsePortFromListenAddr(result.Result.NodeInfo.ListenAddr)
	if err != nil {
		port = 26656 // default CometBFT P2P port
	}

	// Resolve the RPC URL hostname to get the IP for the src field
	parsed, err := url.Parse(rpcURL)
	if err != nil {
		return netAddress{}, fmt.Errorf("failed to parse RPC URL %q: %w", rpcURL, err)
	}

	host := parsed.Hostname()
	ip := host // fallback: use the hostname directly if it's already an IP
	if net.ParseIP(host) == nil {
		// It's a hostname, resolve it
		ips, err := net.LookupHost(host)
		if err != nil || len(ips) == 0 {
			// If DNS resolution fails, use the hostname as-is
			ip = host
		} else {
			ip = ips[0]
		}
	}

	return netAddress{
		ID:   result.Result.NodeInfo.ID,
		IP:   ip,
		Port: port,
	}, nil
}

// parsePortFromListenAddr extracts the port number from a CometBFT listen address
// like "tcp://0.0.0.0:26656" or "0.0.0.0:26656".
func parsePortFromListenAddr(listenAddr string) (uint16, error) {
	// Strip protocol prefix if present
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
