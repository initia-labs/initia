package quickstart

import (
	"fmt"
	"math"
	"path/filepath"

	"github.com/go-viper/mapstructure/v2"
	"github.com/spf13/viper"

	cmtcfg "github.com/cometbft/cometbft/config"

	serverconfig "github.com/cosmos/cosmos-sdk/server/config"

	"github.com/initia-labs/initia/cmd/initiad/appconfig"
)

// applyConfigToml modifies config.toml: rpc address, tx_index, retain-height.
func applyConfigToml(cfg QuickstartConfig, homeDir string) error {
	configPath := filepath.Join(homeDir, "config")
	configFile := filepath.Join(configPath, "config.toml")

	cmtCfg, err := loadCometConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config.toml: %w", err)
	}

	// RPC listen address
	cmtCfg.RPC.ListenAddress = cfg.RPCAddress

	// TX indexing
	switch cfg.TxIndexing {
	case TxIndexNull:
		cmtCfg.TxIndex.Indexer = "null"
	case TxIndexDefault, TxIndexCustom:
		cmtCfg.TxIndex.Indexer = "kv"
	}

	// TX index retain-height follows min-retain-blocks (only when indexing is enabled)
	if cfg.TxIndexing != TxIndexNull {
		if cfg.MinRetainBlocks > math.MaxInt64 {
			cmtCfg.TxIndex.RetainHeight = math.MaxInt64
		} else {
			cmtCfg.TxIndex.RetainHeight = int64(cfg.MinRetainBlocks)
		}
	} else {
		cmtCfg.TxIndex.RetainHeight = 0
	}

	cmtcfg.WriteConfigFile(configFile, cmtCfg)
	return nil
}

// applyAppToml modifies app.toml: pruning, min-retain-blocks, api, memiavl, index-events.
func applyAppToml(cfg QuickstartConfig, homeDir string) error {
	configPath := filepath.Join(homeDir, "config")
	appCfgFile := filepath.Join(configPath, "app.toml")

	appCfg, err := loadAppConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load app.toml: %w", err)
	}

	// Pruning
	appCfg.Pruning = cfg.Pruning
	if cfg.Pruning == PruningCustom {
		appCfg.PruningKeepRecent = cfg.PruningKeepRecent
		appCfg.PruningInterval = cfg.PruningInterval
	} else {
		appCfg.PruningKeepRecent = "0"
		appCfg.PruningInterval = "0"
	}

	// Min retain blocks (block pruning)
	appCfg.MinRetainBlocks = cfg.MinRetainBlocks

	// Index events (the event keys to index)
	if cfg.TxIndexing == TxIndexDefault || cfg.TxIndexing == TxIndexCustom {
		appCfg.IndexEvents = cfg.TxIndexingKeys
	} else {
		appCfg.IndexEvents = []string{}
	}

	// REST API
	if cfg.APIAddress != "" {
		appCfg.API.Enable = true
		appCfg.API.Address = cfg.APIAddress
	}

	// MemIAVL
	appCfg.MemIAVL.Enable = cfg.MemIAVL

	// Write back using template to preserve comments
	serverconfig.SetConfigTemplate(appconfig.AppConfigTemplate())
	serverconfig.WriteConfigFile(appCfgFile, appCfg)
	return nil
}

// loadAppConfig reads app.toml via viper and unmarshals into the app config struct.
func loadAppConfig(configPath string) (*appconfig.InitiaAppConfig, error) {
	v := viper.New()
	v.SetConfigType("toml")
	v.SetConfigName("app")
	v.AddConfigPath(configPath)

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	cfg := &appconfig.InitiaAppConfig{}
	if err := v.Unmarshal(cfg, func(dc *mapstructure.DecoderConfig) {
		dc.Squash = true
	}); err != nil {
		return nil, err
	}

	return cfg, nil
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
