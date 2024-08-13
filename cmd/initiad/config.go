package main

import (
	"fmt"
	"time"

	oracleconfig "github.com/skip-mev/slinky/oracle/config"

	tmcfg "github.com/cometbft/cometbft/config"

	serverconfig "github.com/cosmos/cosmos-sdk/server/config"

	initiaapp "github.com/initia-labs/initia/app"
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

	appConfig := initiaappConfig{
		Config:     *srvCfg,
		MoveConfig: moveconfig.DefaultMoveConfig(),
		Oracle:     oracleconfig.NewDefaultAppConfig(),
	}
	appConfig.Oracle.ClientTimeout = 500 * time.Millisecond

	appConfigTemplate := serverconfig.DefaultConfigTemplate +
		moveconfig.DefaultConfigTemplate +
		oracleconfig.DefaultConfigTemplate

	return appConfigTemplate, appConfig
}

// initTendermintConfig helps to override default Tendermint Config values.
// return tmcfg.DefaultConfig if no custom configuration is required for the application.
func initTendermintConfig() *tmcfg.Config {
	cfg := tmcfg.DefaultConfig()

	// performance turning configs
	cfg.P2P.SendRate = 20480000
	cfg.P2P.RecvRate = 20480000
	cfg.P2P.MaxPacketMsgPayloadSize = 1000000 // 1MB
	cfg.P2P.FlushThrottleTimeout = 10 * time.Millisecond
	cfg.Consensus.PeerGossipSleepDuration = 30 * time.Millisecond

	// mempool configs
	cfg.Mempool.Size = 1000
	cfg.Mempool.MaxTxsBytes = 10737418240
	cfg.Mempool.MaxTxBytes = 2048576

	// set propose timeout to 3s and increase timeout by 500ms each round
	cfg.Consensus.TimeoutPropose = 3 * time.Second
	cfg.Consensus.TimeoutProposeDelta = 500 * time.Millisecond

	// no need to increase wait timeout(delta) for prevote and precommit
	cfg.Consensus.TimeoutPrevote = 500 * time.Millisecond
	cfg.Consensus.TimeoutPrevoteDelta = 0 * time.Millisecond
	cfg.Consensus.TimeoutPrecommit = 500 * time.Millisecond
	cfg.Consensus.TimeoutPrecommitDelta = 0 * time.Millisecond

	// set commit timeout to 2s
	cfg.Consensus.TimeoutCommit = 2 * time.Second

	return cfg
}
