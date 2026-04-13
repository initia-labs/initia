package quickstart

import (
	"fmt"
	"path/filepath"

	cmtcfg "github.com/cometbft/cometbft/config"
	"github.com/spf13/viper"

	serverconfig "github.com/cosmos/cosmos-sdk/server/config"

	"github.com/initia-labs/initia/abcipp"
	moveconfig "github.com/initia-labs/initia/x/move/config"

	oracleconfig "github.com/skip-mev/connect/v2/oracle/config"

	initiastorecfg "github.com/initia-labs/store/config"
)

// appConfig mirrors initiaappConfig from cmd/initiad/config.go
// to support template-based writing that preserves comments.
type appConfig struct {
	serverconfig.Config
	ABCIPP     abcipp.AppConfig               `mapstructure:"abcipp"`
	MoveConfig moveconfig.MoveConfig           `mapstructure:"move"`
	Oracle     oracleconfig.AppConfig          `mapstructure:"oracle"`
	MemIAVL    initiastorecfg.MemIAVLConfig    `mapstructure:"memiavl"`
	VersionDB  initiastorecfg.VersionDBConfig  `mapstructure:"versiondb"`
}

func appConfigTemplate() string {
	return serverconfig.DefaultConfigTemplate +
		abcipp.DefaultConfigTemplate +
		moveconfig.DefaultConfigTemplate +
		oracleconfig.DefaultConfigTemplate +
		initiastorecfg.DefaultMemIAVLConfigTemplate +
		initiastorecfg.DefaultVersionDBConfigTemplate
}

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

	// TX index retain-height follows min-retain-blocks
	cmtCfg.TxIndex.RetainHeight = int64(cfg.MinRetainBlocks)

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
	serverconfig.SetConfigTemplate(appConfigTemplate())
	serverconfig.WriteConfigFile(appCfgFile, appCfg)
	return nil
}

// loadAppConfig reads app.toml via viper and unmarshals into the app config struct.
func loadAppConfig(configPath string) (*appConfig, error) {
	v := viper.New()
	v.SetConfigType("toml")
	v.SetConfigName("app")
	v.AddConfigPath(configPath)

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	cfg := &appConfig{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
