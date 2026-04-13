package quickstart

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func runInteractive(cmd *cobra.Command) (QuickstartConfig, error) {
	reader := bufio.NewReader(cmd.InOrStdin())
	var cfg QuickstartConfig

	// 1. Network
	network, err := promptChoice(reader, cmd, "Select network", []string{NetworkMainnet, NetworkTestnet})
	if err != nil {
		return cfg, err
	}
	cfg.Network = network

	// 2. Sync method
	syncMethod, err := promptChoice(reader, cmd, "Select sync method", []string{SyncStateSync, SyncSnapshot})
	if err != nil {
		return cfg, err
	}
	cfg.SyncMethod = syncMethod

	// 3. Snapshot URL
	if cfg.SyncMethod == SyncSnapshot {
		snapshotHint := "https://polkachu.com/tendermint_snapshots/initia"
		if cfg.Network == NetworkTestnet {
			snapshotHint = "https://www.polkachu.com/testnets/initia/snapshots"
		}
		cmd.Printf("  Find snapshots at: %s\n", snapshotHint)
		url, err := promptString(reader, cmd, "Enter snapshot download URL", "")
		if err != nil {
			return cfg, err
		}
		if url == "" {
			return cfg, fmt.Errorf("snapshot URL is required")
		}
		cfg.SnapshotURL = url
	}

	// 4. App state pruning
	pruning, err := promptChoice(reader, cmd, "Select app state pruning", []string{PruningDefault, PruningNothing, PruningEverything, PruningCustom})
	if err != nil {
		return cfg, err
	}
	cfg.Pruning = pruning

	// 5. Custom pruning params
	if cfg.Pruning == PruningCustom {
		keepRecent, err := promptString(reader, cmd, "Keep recent states", "362880")
		if err != nil {
			return cfg, err
		}
		cfg.PruningKeepRecent = keepRecent

		interval, err := promptString(reader, cmd, "Pruning interval", "100")
		if err != nil {
			return cfg, err
		}
		cfg.PruningInterval = interval
	}

	// 6. Min retain blocks
	minRetainStr, err := promptString(reader, cmd, "Min retain blocks", fmt.Sprintf("%d", DefaultMinRetainBlocks))
	if err != nil {
		return cfg, err
	}
	minRetain, err := strconv.ParseUint(minRetainStr, 10, 64)
	if err != nil {
		return cfg, fmt.Errorf("invalid min-retain-blocks: %w", err)
	}
	cfg.MinRetainBlocks = minRetain

	// 7. TX indexing
	txIndexing, err := promptChoiceWithLabels(reader, cmd, "Select tx indexing", []choiceOption{
		{Value: TxIndexNull, Label: "disable - no tx indexing"},
		{Value: TxIndexDefault, Label: "default - IBC related keys (tx.height, tx.hash, packet sequences, etc.)"},
		{Value: TxIndexCustom, Label: "custom - specify your own indexing keys"},
	})
	if err != nil {
		return cfg, err
	}
	cfg.TxIndexing = txIndexing

	// 8. Custom indexing keys
	if cfg.TxIndexing == TxIndexDefault {
		cfg.TxIndexingKeys = DefaultTxIndexingKeys
	} else if cfg.TxIndexing == TxIndexCustom {
		keysStr, err := promptString(reader, cmd, "Enter indexing keys (comma-separated)", "")
		if err != nil {
			return cfg, err
		}
		cfg.TxIndexingKeys = splitAndTrim(keysStr)
	}

	// 9. MemIAVL (not compatible with snapshot sync)
	if cfg.SyncMethod == SyncSnapshot {
		cfg.MemIAVL = false
	} else {
		memiavl, err := promptYesNo(reader, cmd, "Enable MemIAVL?")
		if err != nil {
			return cfg, err
		}
		cfg.MemIAVL = memiavl
	}

	// 10. REST API
	apiEnable, err := promptYesNo(reader, cmd, "Enable REST API?")
	if err != nil {
		return cfg, err
	}
	if apiEnable {
		apiAddr, err := promptString(reader, cmd, "API listen address", DefaultAPIAddress)
		if err != nil {
			return cfg, err
		}
		cfg.APIAddress = apiAddr
	}

	// 11. RPC address
	rpcAddr, err := promptString(reader, cmd, "RPC listen address", DefaultRPCAddress)
	if err != nil {
		return cfg, err
	}
	cfg.RPCAddress = rpcAddr

	return cfg, nil
}

func promptChoice(reader *bufio.Reader, cmd *cobra.Command, prompt string, choices []string) (string, error) {
	cmd.Printf("\n%s:\n", prompt)
	for i, c := range choices {
		cmd.Printf("  %d) %s\n", i+1, c)
	}
	cmd.Printf("Enter choice [1-%d]: ", len(choices))

	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	input = strings.TrimSpace(input)

	idx, err := strconv.Atoi(input)
	if err != nil || idx < 1 || idx > len(choices) {
		return "", fmt.Errorf("invalid choice: %s", input)
	}

	return choices[idx-1], nil
}

type choiceOption struct {
	Value string
	Label string
}

func promptChoiceWithLabels(reader *bufio.Reader, cmd *cobra.Command, prompt string, choices []choiceOption) (string, error) {
	cmd.Printf("\n%s:\n", prompt)
	for i, c := range choices {
		cmd.Printf("  %d) %s\n", i+1, c.Label)
	}
	cmd.Printf("Enter choice [1-%d]: ", len(choices))

	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	input = strings.TrimSpace(input)

	idx, err := strconv.Atoi(input)
	if err != nil || idx < 1 || idx > len(choices) {
		return "", fmt.Errorf("invalid choice: %s", input)
	}

	return choices[idx-1].Value, nil
}

func promptString(reader *bufio.Reader, cmd *cobra.Command, prompt, defaultVal string) (string, error) {
	if defaultVal != "" {
		cmd.Printf("%s [%s]: ", prompt, defaultVal)
	} else {
		cmd.Printf("%s: ", prompt)
	}

	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	input = strings.TrimSpace(input)

	if input == "" {
		return defaultVal, nil
	}
	return input, nil
}

func promptYesNo(reader *bufio.Reader, cmd *cobra.Command, prompt string) (bool, error) {
	cmd.Printf("%s [y/n]: ", prompt)

	input, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	input = strings.TrimSpace(strings.ToLower(input))

	return input == "y" || input == "yes", nil
}
