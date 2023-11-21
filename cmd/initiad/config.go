package main

import (
	"fmt"

	tmcfg "github.com/cometbft/cometbft/config"

	serverconfig "github.com/cosmos/cosmos-sdk/server/config"

	initiaapp "github.com/initia-labs/initia/app"
	moveconfig "github.com/initia-labs/initia/x/move/config"
)

// initiaappConfig initia specify app config
type initiaappConfig struct {
	serverconfig.Config
	MoveConfig moveconfig.MoveConfig `mapstructure:"move"`
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

	initiaappConfig := initiaappConfig{
		Config:     *srvCfg,
		MoveConfig: moveconfig.DefaultMoveConfig(),
	}

	initiaappTemplate := serverconfig.DefaultConfigTemplate + moveconfig.DefaultConfigTemplate

	return initiaappTemplate, initiaappConfig
}

// initTendermintConfig helps to override default Tendermint Config values.
// return tmcfg.DefaultConfig if no custom configuration is required for the application.
func initTendermintConfig() *tmcfg.Config {
	cfg := tmcfg.DefaultConfig()

	// block time from 5s to 3s
	cfg.Consensus.TimeoutPropose = cfg.Consensus.TimeoutPropose * 3 / 5
	cfg.Consensus.TimeoutProposeDelta = cfg.Consensus.TimeoutProposeDelta * 3 / 5
	cfg.Consensus.TimeoutPrevote = cfg.Consensus.TimeoutPrevote * 3 / 5
	cfg.Consensus.TimeoutPrevoteDelta = cfg.Consensus.TimeoutPrevoteDelta * 3 / 5
	cfg.Consensus.TimeoutPrecommit = cfg.Consensus.TimeoutPrecommit * 3 / 5
	cfg.Consensus.TimeoutPrecommitDelta = cfg.Consensus.TimeoutPrecommitDelta * 3 / 5
	cfg.Consensus.TimeoutCommit = cfg.Consensus.TimeoutCommit * 3 / 5

	return cfg
}
