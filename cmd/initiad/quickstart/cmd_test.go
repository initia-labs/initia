package quickstart

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadFromFlags_ValidMainnetStateSync(t *testing.T) {
	cmd := QuickstartCmd("")
	cmd.SetArgs([]string{})
	cmd.Flags().Set("network", "mainnet")
	cmd.Flags().Set("sync-method", "statesync")
	cmd.Flags().Set("tx-indexing", "default")
	cmd.Flags().Set("pruning", "default")

	cfg, err := readFromFlags(cmd)
	require.NoError(t, err)
	require.Equal(t, "mainnet", cfg.Network)
	require.Equal(t, "statesync", cfg.SyncMethod)
	require.Equal(t, "", cfg.SnapshotURL)
	require.Equal(t, "default", cfg.TxIndexing)
	require.Equal(t, DefaultTxIndexingKeys, cfg.TxIndexingKeys)
	require.Equal(t, "default", cfg.Pruning)
}

func TestReadFromFlags_ValidSnapshot(t *testing.T) {
	cmd := QuickstartCmd("")
	cmd.SetArgs([]string{})
	cmd.Flags().Set("network", "testnet")
	cmd.Flags().Set("sync-method", "snapshot")
	cmd.Flags().Set("snapshot-url", "https://example.com/snap.tar.lz4")
	cmd.Flags().Set("tx-indexing", "null")
	cmd.Flags().Set("pruning", "nothing")

	cfg, err := readFromFlags(cmd)
	require.NoError(t, err)
	require.Equal(t, "testnet", cfg.Network)
	require.Equal(t, "snapshot", cfg.SyncMethod)
	require.Equal(t, "https://example.com/snap.tar.lz4", cfg.SnapshotURL)
	require.Equal(t, "null", cfg.TxIndexing)
	require.Nil(t, cfg.TxIndexingKeys)
	require.Equal(t, "nothing", cfg.Pruning)
}

func TestReadFromFlags_CustomPruning(t *testing.T) {
	cmd := QuickstartCmd("")
	cmd.SetArgs([]string{})
	cmd.Flags().Set("network", "mainnet")
	cmd.Flags().Set("sync-method", "statesync")
	cmd.Flags().Set("tx-indexing", "default")
	cmd.Flags().Set("pruning", "custom")
	cmd.Flags().Set("pruning-keep-recent", "1000")
	cmd.Flags().Set("pruning-interval", "10")

	cfg, err := readFromFlags(cmd)
	require.NoError(t, err)
	require.Equal(t, "custom", cfg.Pruning)
	require.Equal(t, "1000", cfg.PruningKeepRecent)
	require.Equal(t, "10", cfg.PruningInterval)
}

func TestReadFromFlags_CustomTxIndexing(t *testing.T) {
	cmd := QuickstartCmd("")
	cmd.SetArgs([]string{})
	cmd.Flags().Set("network", "mainnet")
	cmd.Flags().Set("sync-method", "statesync")
	cmd.Flags().Set("tx-indexing", "custom")
	cmd.Flags().Set("tx-indexing-keys", "key1,key2,key3")
	cmd.Flags().Set("pruning", "default")

	cfg, err := readFromFlags(cmd)
	require.NoError(t, err)
	require.Equal(t, "custom", cfg.TxIndexing)
	require.Equal(t, []string{"key1", "key2", "key3"}, cfg.TxIndexingKeys)
}

func TestReadFromFlags_InvalidNetwork(t *testing.T) {
	cmd := QuickstartCmd("")
	cmd.SetArgs([]string{})
	cmd.Flags().Set("network", "invalid")
	cmd.Flags().Set("sync-method", "statesync")
	cmd.Flags().Set("tx-indexing", "default")
	cmd.Flags().Set("pruning", "default")

	_, err := readFromFlags(cmd)
	require.Error(t, err)
	require.Contains(t, err.Error(), "mainnet")
	require.Contains(t, err.Error(), "testnet")
}

func TestReadFromFlags_InvalidSyncMethod(t *testing.T) {
	cmd := QuickstartCmd("")
	cmd.SetArgs([]string{})
	cmd.Flags().Set("network", "mainnet")
	cmd.Flags().Set("sync-method", "invalid")
	cmd.Flags().Set("tx-indexing", "default")
	cmd.Flags().Set("pruning", "default")

	_, err := readFromFlags(cmd)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid sync-method")
}

func TestReadFromFlags_SnapshotMissingURL(t *testing.T) {
	cmd := QuickstartCmd("")
	cmd.SetArgs([]string{})
	cmd.Flags().Set("network", "mainnet")
	cmd.Flags().Set("sync-method", "snapshot")
	cmd.Flags().Set("tx-indexing", "default")
	cmd.Flags().Set("pruning", "default")

	_, err := readFromFlags(cmd)
	require.Error(t, err)
	require.Contains(t, err.Error(), "snapshot-url")
}

func TestReadFromFlags_InvalidTxIndexing(t *testing.T) {
	cmd := QuickstartCmd("")
	cmd.SetArgs([]string{})
	cmd.Flags().Set("network", "mainnet")
	cmd.Flags().Set("sync-method", "statesync")
	cmd.Flags().Set("tx-indexing", "invalid")
	cmd.Flags().Set("pruning", "default")

	_, err := readFromFlags(cmd)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid tx-indexing")
}

func TestReadFromFlags_CustomTxIndexingMissingKeys(t *testing.T) {
	cmd := QuickstartCmd("")
	cmd.SetArgs([]string{})
	cmd.Flags().Set("network", "mainnet")
	cmd.Flags().Set("sync-method", "statesync")
	cmd.Flags().Set("tx-indexing", "custom")
	cmd.Flags().Set("pruning", "default")

	_, err := readFromFlags(cmd)
	require.Error(t, err)
	require.Contains(t, err.Error(), "tx-indexing-keys")
}

func TestReadFromFlags_DefaultTxIndexingKeys(t *testing.T) {
	cmd := QuickstartCmd("")
	cmd.SetArgs([]string{})
	cmd.Flags().Set("network", "mainnet")
	cmd.Flags().Set("sync-method", "statesync")
	cmd.Flags().Set("tx-indexing", "default")
	cmd.Flags().Set("pruning", "default")

	cfg, err := readFromFlags(cmd)
	require.NoError(t, err)
	require.Equal(t, DefaultTxIndexingKeys, cfg.TxIndexingKeys)
}
