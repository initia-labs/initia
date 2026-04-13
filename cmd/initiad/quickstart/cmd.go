package quickstart

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const (
	NetworkMainnet = "mainnet"
	NetworkTestnet = "testnet"

	SyncStateSync = "statesync"
	SyncSnapshot  = "snapshot"

	TxIndexNull    = "null"
	TxIndexDefault = "default"
	TxIndexCustom  = "custom"

	PruningDefault    = "default"
	PruningNothing    = "nothing"
	PruningEverything = "everything"
	PruningCustom     = "custom"

	DefaultMinRetainBlocks = uint64(1_000_000)
	DefaultAPIAddress      = "tcp://0.0.0.0:1317"
	DefaultRPCAddress      = "tcp://127.0.0.1:26657"
)

var DefaultTxIndexingKeys = []string{
	"tx.height",
	"tx.hash",
	"send_packet.packet_sequence",
	"recv_packet.packet_sequence",
	"write_acknowledgement.packet_sequence",
	"acknowledge_packet.packet_sequence",
	"timeout_packet.packet_sequence",
	"finalize_token_deposit.l1_sequence",
}

const (
	flagInteractive       = "interactive"
	flagNetwork           = "network"
	flagSyncMethod        = "sync-method"
	flagSnapshotURL       = "snapshot-url"
	flagTxIndexing        = "tx-indexing"
	flagTxIndexingKeys    = "tx-indexing-keys"
	flagPruning           = "pruning"
	flagPruningKeepRecent = "pruning-keep-recent"
	flagPruningInterval   = "pruning-interval"
	flagMinRetainBlocks   = "min-retain-blocks"
	flagMemIAVL           = "memiavl"
	flagAPIAddress        = "api-address"
	flagRPCAddress        = "rpc-address"
)

type QuickstartConfig struct {
	Network           string
	SyncMethod        string
	SnapshotURL       string
	TxIndexing        string
	TxIndexingKeys    []string
	Pruning           string
	PruningKeepRecent string
	PruningInterval   string
	MinRetainBlocks   uint64
	MemIAVL           bool
	APIAddress        string
	RPCAddress        string
}

// QuickstartCmd returns the cobra command for quickstart node setup.
func QuickstartCmd(defaultNodeHome string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "quickstart",
		Aliases: []string{"qstart", "qs"},
		Short:   "Quickly configure and sync an Initia node",
		Long: `Quickstart sets up an Initia node by downloading the genesis file,
address book, and configuring sync method, pruning, indexing, and other settings.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			interactive, _ := cmd.Flags().GetBool(flagInteractive)

			var cfg QuickstartConfig
			var err error

			if interactive {
				cfg, err = runInteractive(cmd)
				if err != nil {
					return err
				}
			} else {
				cfg, err = readFromFlags(cmd)
				if err != nil {
					return err
				}
			}

			homeDir, _ := cmd.Flags().GetString("home")
			if homeDir == "" {
				homeDir = defaultNodeHome
			}

			return run(cfg, homeDir)
		},
	}

	cmd.Flags().Bool(flagInteractive, false, "Run in interactive mode")
	cmd.Flags().String(flagNetwork, NetworkMainnet, "Network to join (mainnet or testnet)")
	cmd.Flags().String(flagSyncMethod, SyncStateSync, "Sync method (statesync or snapshot)")
	cmd.Flags().String(flagSnapshotURL, "", "URL of the snapshot to download (required when sync-method=snapshot)")
	cmd.Flags().String(flagTxIndexing, TxIndexDefault, "Transaction indexing mode (null, default, or custom)")
	cmd.Flags().String(flagTxIndexingKeys, "", "Comma-separated list of tx indexing keys (required when tx-indexing=custom)")
	cmd.Flags().String(flagPruning, PruningDefault, "Pruning strategy (default, nothing, everything, or custom)")
	cmd.Flags().String(flagPruningKeepRecent, "", "Number of recent states to keep (required when pruning=custom)")
	cmd.Flags().String(flagPruningInterval, "", "Pruning interval (required when pruning=custom)")
	cmd.Flags().Uint64(flagMinRetainBlocks, DefaultMinRetainBlocks, "Minimum number of blocks to retain")
	cmd.Flags().Bool(flagMemIAVL, true, "Enable MemIAVL for faster sync")
	cmd.Flags().String(flagAPIAddress, DefaultAPIAddress, "API server listen address")
	cmd.Flags().String(flagRPCAddress, DefaultRPCAddress, "RPC server listen address")

	return cmd
}

// readFromFlags reads and validates QuickstartConfig from command flags.
func readFromFlags(cmd *cobra.Command) (QuickstartConfig, error) {
	network, _ := cmd.Flags().GetString(flagNetwork)
	syncMethod, _ := cmd.Flags().GetString(flagSyncMethod)
	snapshotURL, _ := cmd.Flags().GetString(flagSnapshotURL)
	txIndexing, _ := cmd.Flags().GetString(flagTxIndexing)
	txIndexingKeysRaw, _ := cmd.Flags().GetString(flagTxIndexingKeys)
	pruning, _ := cmd.Flags().GetString(flagPruning)
	pruningKeepRecent, _ := cmd.Flags().GetString(flagPruningKeepRecent)
	pruningInterval, _ := cmd.Flags().GetString(flagPruningInterval)
	minRetainBlocks, _ := cmd.Flags().GetUint64(flagMinRetainBlocks)
	memIAVL, _ := cmd.Flags().GetBool(flagMemIAVL)
	apiAddress, _ := cmd.Flags().GetString(flagAPIAddress)
	rpcAddress, _ := cmd.Flags().GetString(flagRPCAddress)

	// Validate network
	if network != NetworkMainnet && network != NetworkTestnet {
		return QuickstartConfig{}, fmt.Errorf("invalid network %q: must be %q or %q", network, NetworkMainnet, NetworkTestnet)
	}

	// Validate sync method
	if syncMethod != SyncStateSync && syncMethod != SyncSnapshot {
		return QuickstartConfig{}, fmt.Errorf("invalid sync-method %q: must be %q or %q", syncMethod, SyncStateSync, SyncSnapshot)
	}

	// Validate snapshot URL when using snapshot sync
	if syncMethod == SyncSnapshot && snapshotURL == "" {
		return QuickstartConfig{}, fmt.Errorf("--%s is required when --%s=%s", flagSnapshotURL, flagSyncMethod, SyncSnapshot)
	}

	// Validate tx indexing
	if txIndexing != TxIndexNull && txIndexing != TxIndexDefault && txIndexing != TxIndexCustom {
		return QuickstartConfig{}, fmt.Errorf("invalid tx-indexing %q: must be %q, %q, or %q", txIndexing, TxIndexNull, TxIndexDefault, TxIndexCustom)
	}

	// Resolve tx indexing keys
	var txIndexingKeys []string
	switch txIndexing {
	case TxIndexCustom:
		if txIndexingKeysRaw == "" {
			return QuickstartConfig{}, fmt.Errorf("--%s is required when --%s=%s", flagTxIndexingKeys, flagTxIndexing, TxIndexCustom)
		}
		txIndexingKeys = splitAndTrim(txIndexingKeysRaw)
	case TxIndexDefault:
		txIndexingKeys = DefaultTxIndexingKeys
	}

	// Validate pruning
	switch pruning {
	case PruningDefault, PruningNothing, PruningEverything:
		// valid
	case PruningCustom:
		if pruningKeepRecent == "" {
			return QuickstartConfig{}, fmt.Errorf("--%s is required when --%s=%s", flagPruningKeepRecent, flagPruning, PruningCustom)
		}
		if pruningInterval == "" {
			return QuickstartConfig{}, fmt.Errorf("--%s is required when --%s=%s", flagPruningInterval, flagPruning, PruningCustom)
		}
	default:
		return QuickstartConfig{}, fmt.Errorf("invalid pruning %q: must be %q, %q, %q, or %q", pruning, PruningDefault, PruningNothing, PruningEverything, PruningCustom)
	}

	return QuickstartConfig{
		Network:           network,
		SyncMethod:        syncMethod,
		SnapshotURL:       snapshotURL,
		TxIndexing:        txIndexing,
		TxIndexingKeys:    txIndexingKeys,
		Pruning:           pruning,
		PruningKeepRecent: pruningKeepRecent,
		PruningInterval:   pruningInterval,
		MinRetainBlocks:   minRetainBlocks,
		MemIAVL:           memIAVL,
		APIAddress:        apiAddress,
		RPCAddress:        rpcAddress,
	}, nil
}

// run executes the quickstart setup with the given config and home directory.
func run(cfg QuickstartConfig, homeDir string) error {
	configDir := filepath.Join(homeDir, "config")
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		return fmt.Errorf("config directory %q does not exist; please run 'initiad init' first", configDir)
	}

	fmt.Println("Downloading genesis file...")
	if err := downloadGenesis(cfg.Network, homeDir); err != nil {
		return fmt.Errorf("failed to download genesis: %w", err)
	}

	fmt.Println("Downloading address book...")
	if err := downloadAddrbook(cfg.Network, homeDir); err != nil {
		return fmt.Errorf("failed to download address book: %w", err)
	}

	fmt.Println("Applying config.toml settings...")
	if err := applyConfigToml(cfg, homeDir); err != nil {
		return fmt.Errorf("failed to apply config.toml: %w", err)
	}

	fmt.Println("Applying app.toml settings...")
	if err := applyAppToml(cfg, homeDir); err != nil {
		return fmt.Errorf("failed to apply app.toml: %w", err)
	}

	switch cfg.SyncMethod {
	case SyncStateSync:
		fmt.Println("Setting up state sync...")
		if err := setupStateSync(cfg.Network, homeDir); err != nil {
			return fmt.Errorf("failed to set up state sync: %w", err)
		}
	case SyncSnapshot:
		fmt.Println("Downloading and extracting snapshot...")
		if err := downloadAndExtractSnapshot(cfg.SnapshotURL, homeDir); err != nil {
			return fmt.Errorf("failed to download/extract snapshot: %w", err)
		}
	}

	fmt.Println("Quickstart setup complete!")
	return nil
}

// splitAndTrim splits a comma-separated string and trims whitespace from each element.
func splitAndTrim(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// Stub functions — to be implemented in later tasks.

func runInteractive(cmd *cobra.Command) (QuickstartConfig, error) {
	return QuickstartConfig{}, fmt.Errorf("not implemented")
}

func applyConfigToml(cfg QuickstartConfig, homeDir string) error {
	return nil
}

func applyAppToml(cfg QuickstartConfig, homeDir string) error {
	return nil
}

func downloadAndExtractSnapshot(url, homeDir string) error {
	return nil
}
