package quickstart

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	cmtcfg "github.com/cometbft/cometbft/config"
	"github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/p2p/pex"

	"github.com/initia-labs/initia/cmd/initiad/quickstart/providers"
)

// rpcEndpoints maps network names to public RPC URLs for fallback operations.
var rpcEndpoints = map[string]string{
	NetworkMainnet: "https://rpc.initia.xyz",
	NetworkTestnet: "https://rpc.testnet.initia.xyz",
}

func downloadGenesis(network, homeDir string) error {
	pc := providers.Polkachu[network]
	destPath := filepath.Join(homeDir, "config", "genesis.json")
	if err := providers.DownloadFile(pc.GenesisURL, destPath); err != nil {
		fmt.Printf("Pre-built genesis download failed (%v), fetching from RPC...\n", err)
		return downloadGenesisFromRPC(rpcEndpoints[network], destPath)
	}
	return nil
}

func downloadAddrbook(network, homeDir string) error {
	pc := providers.Polkachu[network]
	destPath := filepath.Join(homeDir, "config", "addrbook.json")
	if err := providers.DownloadFile(pc.AddrbookURL, destPath); err != nil {
		fmt.Printf("Pre-built addrbook download failed (%v), building from RPC peers...\n", err)
		return buildAddrbookFromRPC(rpcEndpoints[network], destPath)
	}
	return nil
}

func setupStateSync(network, homeDir string) error {
	pc := providers.Polkachu[network]

	latestHeight, err := providers.FetchLatestHeight(pc.StateSyncRPC)
	if err != nil {
		return fmt.Errorf("failed to fetch latest height: %w", err)
	}

	trustHeight := max(latestHeight-2000, 1)

	trustHash, err := providers.FetchBlockHash(pc.StateSyncRPC, trustHeight)
	if err != nil {
		return fmt.Errorf("failed to fetch trust hash: %w", err)
	}
	if trustHash == "" {
		return fmt.Errorf("RPC returned empty block hash for height %d; the block may have been pruned", trustHeight)
	}

	stateSyncPeer, err := providers.FetchStateSyncPeer(pc.LivePeersAPI)
	if err != nil {
		return fmt.Errorf("failed to fetch state sync peer: %w", err)
	}
	if stateSyncPeer == "" {
		return fmt.Errorf("API returned empty state sync peer")
	}

	return applyStateSync(homeDir, pc.StateSyncRPC, trustHeight, trustHash, stateSyncPeer)
}

// applyStateSync modifies statesync and p2p settings in config.toml.
func applyStateSync(homeDir, rpc string, trustHeight int64, trustHash, stateSyncPeer string) error {
	configPath := filepath.Join(homeDir, "config")
	configFile := filepath.Join(configPath, "config.toml")

	cfg, err := loadCometConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config.toml: %w", err)
	}

	cfg.StateSync.Enable = true
	cfg.StateSync.RPCServers = []string{rpc, rpc}
	cfg.StateSync.TrustHeight = trustHeight
	cfg.StateSync.TrustHash = trustHash

	// Append state sync peer to persistent_peers if not already present
	if !strings.Contains(cfg.P2P.PersistentPeers, stateSyncPeer) {
		if cfg.P2P.PersistentPeers != "" {
			cfg.P2P.PersistentPeers += ","
		}
		cfg.P2P.PersistentPeers += stateSyncPeer
	}

	cmtcfg.WriteConfigFile(configFile, cfg)
	return nil
}

// downloadGenesisFromRPC fetches genesis.json from the RPC /genesis endpoint.
// The response wraps genesis in {"result":{"genesis":{...}}}, so we extract the inner object.
func downloadGenesisFromRPC(rpcURL, destPath string) error {
	resp, err := providers.HTTPClient.Get(rpcURL + "/genesis")
	if err != nil {
		return fmt.Errorf("failed to fetch genesis from RPC: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("RPC %s/genesis returned status %d", rpcURL, resp.StatusCode)
	}

	var result struct {
		Result struct {
			Genesis json.RawMessage `json:"genesis"`
		} `json:"result"`
	}

	if err := json.NewDecoder(io.LimitReader(resp.Body, 500<<20)).Decode(&result); err != nil {
		return fmt.Errorf("failed to decode genesis response: %w", err)
	}

	if len(result.Result.Genesis) == 0 {
		return fmt.Errorf("RPC returned empty genesis")
	}

	return providers.WriteAtomically(destPath, strings.NewReader(string(result.Result.Genesis)))
}

// buildAddrbookFromRPC fetches peers from the RPC node's /net_info and builds
// addrbook.json using CometBFT's pex.AddrBook API.
func buildAddrbookFromRPC(rpcURL, destPath string) error {
	peerAddrs, err := fetchNetInfoPeerAddrs(rpcURL)
	if err != nil {
		return fmt.Errorf("failed to fetch net_info: %w", err)
	}
	if len(peerAddrs) == 0 {
		return fmt.Errorf("RPC %s/net_info returned no peers", rpcURL)
	}

	srcAddr, err := fetchRPCNodeAddr(rpcURL)
	if err != nil {
		return fmt.Errorf("failed to fetch RPC node status: %w", err)
	}

	book := pex.NewAddrBook(destPath, false)
	book.SetLogger(log.NewNopLogger())

	for _, addr := range peerAddrs {
		if err := book.AddAddress(addr, srcAddr); err != nil {
			continue
		}
	}

	book.Save()
	return nil
}

// fetchNetInfoPeerAddrs calls /net_info and returns parsed p2p.NetAddress entries.
func fetchNetInfoPeerAddrs(rpcURL string) ([]*p2p.NetAddress, error) {
	resp, err := providers.HTTPClient.Get(rpcURL + "/net_info")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
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
		ip := net.ParseIP(peer.RemoteIP)
		if ip == nil {
			continue
		}
		addr := p2p.NewNetAddressIPPort(ip, port)
		addr.ID = p2p.ID(peer.NodeInfo.ID)
		addrs = append(addrs, addr)
	}

	return addrs, nil
}

// fetchRPCNodeAddr calls /status and resolves the RPC hostname to build
// the source p2p.NetAddress for addrbook entries.
func fetchRPCNodeAddr(rpcURL string) (*p2p.NetAddress, error) {
	resp, err := providers.HTTPClient.Get(rpcURL + "/status")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
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

	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&result); err != nil {
		return nil, err
	}

	port, err := parsePortFromListenAddr(result.Result.NodeInfo.ListenAddr)
	if err != nil {
		port = 26656
	}

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
