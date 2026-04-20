package quickstart

import (
	"os"
	"path/filepath"
	"testing"

	cmtcfg "github.com/cometbft/cometbft/config"
	"github.com/stretchr/testify/require"

	serverconfig "github.com/cosmos/cosmos-sdk/server/config"

	"github.com/initia-labs/initia/cmd/initiad/appconfig"
)

func setupTestConfigDir(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "config")
	require.NoError(t, os.MkdirAll(configDir, 0o755))

	// Create config.toml
	tmCfg := cmtcfg.DefaultConfig()
	tmCfg.SetRoot(tmpDir)
	cmtcfg.WriteConfigFile(filepath.Join(configDir, "config.toml"), tmCfg)

	// Create app.toml
	template, appCfg := appconfig.InitAppConfig()
	serverconfig.SetConfigTemplate(template)
	serverconfig.WriteConfigFile(filepath.Join(configDir, "app.toml"), appCfg)

	return tmpDir
}

func TestApplyConfigToml(t *testing.T) {
	tmpDir := setupTestConfigDir(t)

	// Save original values before applying quickstart
	origCfg := cmtcfg.DefaultConfig()

	cfg := QuickstartConfig{
		RPCAddress:      "tcp://0.0.0.0:26657",
		TxIndexing:      TxIndexDefault,
		MinRetainBlocks: 500000,
	}

	err := applyConfigToml(cfg, tmpDir)
	require.NoError(t, err)

	// Load back and verify changed fields
	configPath := filepath.Join(tmpDir, "config")
	loaded, err := loadCometConfig(configPath)
	require.NoError(t, err)

	require.Equal(t, "tcp://0.0.0.0:26657", loaded.RPC.ListenAddress)
	require.Equal(t, "kv", loaded.TxIndex.Indexer)
	require.Equal(t, int64(500000), loaded.TxIndex.RetainHeight)

	// Verify unchanged fields are preserved
	require.Equal(t, origCfg.P2P.ListenAddress, loaded.P2P.ListenAddress)
	require.Equal(t, origCfg.Consensus.TimeoutCommit, loaded.Consensus.TimeoutCommit)
	require.Equal(t, origCfg.Mempool.Size, loaded.Mempool.Size)
	require.Equal(t, origCfg.P2P.PersistentPeers, loaded.P2P.PersistentPeers)
}

func TestApplyConfigTomlNullIndex(t *testing.T) {
	tmpDir := setupTestConfigDir(t)

	cfg := QuickstartConfig{
		RPCAddress:      "tcp://127.0.0.1:26657",
		TxIndexing:      TxIndexNull,
		MinRetainBlocks: 100000,
	}

	err := applyConfigToml(cfg, tmpDir)
	require.NoError(t, err)

	configPath := filepath.Join(tmpDir, "config")
	loaded, err := loadCometConfig(configPath)
	require.NoError(t, err)

	require.Equal(t, "null", loaded.TxIndex.Indexer)
	require.Equal(t, int64(0), loaded.TxIndex.RetainHeight)
}

func TestApplyAppToml(t *testing.T) {
	tmpDir := setupTestConfigDir(t)

	// Save original values before applying quickstart
	_, origCfg := appconfig.InitAppConfig()

	cfg := QuickstartConfig{
		Pruning:           PruningCustom,
		PruningKeepRecent: "500",
		PruningInterval:   "100",
		MinRetainBlocks:   1000000,
		TxIndexing:        TxIndexDefault,
		TxIndexingKeys:    DefaultTxIndexingKeys,
		APIAddress:        "tcp://0.0.0.0:1317",
		MemIAVL:           true,
	}

	err := applyAppToml(cfg, tmpDir)
	require.NoError(t, err)

	// Load back and verify changed fields
	configPath := filepath.Join(tmpDir, "config")
	loaded, err := loadAppConfig(configPath)
	require.NoError(t, err)

	require.Equal(t, PruningCustom, loaded.Pruning)
	require.Equal(t, "500", loaded.PruningKeepRecent)
	require.Equal(t, "100", loaded.PruningInterval)
	require.Equal(t, uint64(1000000), loaded.MinRetainBlocks)
	require.True(t, loaded.API.Enable)
	require.Equal(t, "tcp://0.0.0.0:1317", loaded.API.Address)
	require.True(t, loaded.MemIAVL.Enable)
	require.Equal(t, DefaultTxIndexingKeys, loaded.IndexEvents)

	// Verify unchanged fields are preserved
	require.Equal(t, origCfg.MinGasPrices, loaded.MinGasPrices)
	require.Equal(t, origCfg.Mempool.MaxTxs, loaded.Mempool.MaxTxs)
	require.Equal(t, origCfg.QueryGasLimit, loaded.QueryGasLimit)
	require.Equal(t, origCfg.GRPC.Enable, loaded.GRPC.Enable)
	require.Equal(t, origCfg.GRPC.Address, loaded.GRPC.Address)
	require.Equal(t, origCfg.GRPCWeb.Enable, loaded.GRPCWeb.Enable)
	require.Equal(t, origCfg.API.RPCReadTimeout, loaded.API.RPCReadTimeout)
	require.Equal(t, origCfg.API.MaxOpenConnections, loaded.API.MaxOpenConnections)
}

func TestApplyAppTomlDefaults(t *testing.T) {
	tmpDir := setupTestConfigDir(t)

	cfg := QuickstartConfig{
		Pruning:         PruningDefault,
		MinRetainBlocks: 500000,
		TxIndexing:      TxIndexNull,
		MemIAVL:         false,
	}

	err := applyAppToml(cfg, tmpDir)
	require.NoError(t, err)

	configPath := filepath.Join(tmpDir, "config")
	loaded, err := loadAppConfig(configPath)
	require.NoError(t, err)

	require.Equal(t, PruningDefault, loaded.Pruning)
	require.Equal(t, "0", loaded.PruningKeepRecent)
	require.Equal(t, "0", loaded.PruningInterval)
	require.Equal(t, uint64(500000), loaded.MinRetainBlocks)
	require.False(t, loaded.MemIAVL.Enable)
	require.Empty(t, loaded.IndexEvents)
}
