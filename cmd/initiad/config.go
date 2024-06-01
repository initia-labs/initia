package main

import (
	"fmt"
	"time"

	oracleconfig "github.com/skip-mev/slinky/oracle/config"

	tmcfg "github.com/cometbft/cometbft/config"

	serverconfig "github.com/cosmos/cosmos-sdk/server/config"

	initiaapp "github.com/initia-labs/initia/app"
	initiaapporacle "github.com/initia-labs/initia/app/oracle"
	moveconfig "github.com/initia-labs/initia/x/move/config"
)

// initiaappConfig initia specify app config
type initiaappConfig struct {
	serverconfig.Config
	MoveConfig moveconfig.MoveConfig  `mapstructure:"move"`
	Oracle     oracleconfig.AppConfig `mapstructure:"oracle"`
}

// initAppConfig helps to override default appConfig template and configs.
// return "", nil if no custom configuration is required for the application.
func initAppConfig() (string, interface{}) {
	// Optionally allow the chain developer to overwrite the SDK's default
	// server config.
	srvCfg := serverconfig.DefaultConfig()

	// The SDK's default minimum gas price is set to "" (empty value) inside
	// app.toml. If left empty by validators, the node will halt on startup.
	// However, the chain developer can set a default app.toml value for their
	// validators here.
	//
	// In summary:
	// - if you leave srvCfg.MinGasPrices = "", all validators MUST tweak their
	//   own app.toml config,
	// - if you set srvCfg.MinGasPrices non-empty, validators CAN tweak their
	//   own app.toml to override, or use this default value.
	//
	// In simapp, we set the min gas prices to 0.
	srvCfg.MinGasPrices = fmt.Sprintf("0%s", initiaapp.BondDenom)
	srvCfg.Mempool.MaxTxs = 2000
	srvCfg.QueryGasLimit = 3000000

	initiaappConfig := initiaappConfig{
		Config:     *srvCfg,
		MoveConfig: moveconfig.DefaultMoveConfig(),
		Oracle:     initiaapporacle.DefaultConfig(),
	}

	initiaappTemplate := serverconfig.DefaultConfigTemplate +
		moveconfig.DefaultConfigTemplate +
		oracleconfig.DefaultConfigTemplate

	return initiaappTemplate, initiaappConfig
}

// initTendermintConfig helps to override default Tendermint Config values.
// return tmcfg.DefaultConfig if no custom configuration is required for the application.
func initTendermintConfig() *tmcfg.Config {
	cfg := tmcfg.DefaultConfig()

	// set block time to 3s
	cfg.Consensus.TimeoutPropose = 1800 * time.Millisecond
	cfg.Consensus.TimeoutProposeDelta = 300 * time.Millisecond
	cfg.Consensus.TimeoutPrevote = 600 * time.Millisecond
	cfg.Consensus.TimeoutPrevoteDelta = 300 * time.Millisecond
	cfg.Consensus.TimeoutPrecommit = 600 * time.Millisecond
	cfg.Consensus.TimeoutPrecommitDelta = 300 * time.Millisecond
	cfg.Consensus.TimeoutCommit = 3000 * time.Millisecond

	return cfg
}
