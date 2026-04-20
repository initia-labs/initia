package appconfig

import (
	"fmt"
	"time"

	oracleconfig "github.com/skip-mev/connect/v2/oracle/config"

	tmcfg "github.com/cometbft/cometbft/config"

	serverconfig "github.com/cosmos/cosmos-sdk/server/config"

	"github.com/initia-labs/initia/abcipp"
	initiaapp "github.com/initia-labs/initia/app"
	moveconfig "github.com/initia-labs/initia/x/move/config"

	initiastorecfg "github.com/initia-labs/store/config"
)

// InitiaAppConfig is the initia-specific app config that extends the SDK server config.
type InitiaAppConfig struct {
	serverconfig.Config
	ABCIPP     abcipp.AppConfig               `mapstructure:"abcipp"`
	MoveConfig moveconfig.MoveConfig          `mapstructure:"move"`
	Oracle     oracleconfig.AppConfig         `mapstructure:"oracle"`
	MemIAVL    initiastorecfg.MemIAVLConfig   `mapstructure:"memiavl"`
	VersionDB  initiastorecfg.VersionDBConfig `mapstructure:"versiondb"`
}

// AppConfigTemplate returns the full app.toml template string.
func AppConfigTemplate() string {
	return serverconfig.DefaultConfigTemplate +
		abcipp.DefaultConfigTemplate +
		moveconfig.DefaultConfigTemplate +
		oracleconfig.DefaultConfigTemplate +
		initiastorecfg.DefaultMemIAVLConfigTemplate +
		initiastorecfg.DefaultVersionDBConfigTemplate
}

// InitAppConfig returns the default app config template and struct
// with initia-specific defaults.
func InitAppConfig() (string, *InitiaAppConfig) {
	srvCfg := serverconfig.DefaultConfig()
	srvCfg.MinGasPrices = fmt.Sprintf("0%s", initiaapp.BondDenom)
	srvCfg.Mempool.MaxTxs = 2000
	srvCfg.QueryGasLimit = 3000000
	srvCfg.InterBlockCache = false

	appConfig := &InitiaAppConfig{
		Config:     *srvCfg,
		ABCIPP:     abcipp.DefaultAppConfig(),
		MoveConfig: moveconfig.DefaultMoveConfig(),
		Oracle:     oracleconfig.NewDefaultAppConfig(),
		MemIAVL:    initiastorecfg.DefaultMemIAVLConfig(),
		VersionDB:  initiastorecfg.DefaultVersionDBConfig(),
	}
	appConfig.Oracle.ClientTimeout = 500 * time.Millisecond

	return AppConfigTemplate(), appConfig
}

// InitTendermintConfig returns the default CometBFT config with initia-specific tuning.
func InitTendermintConfig() *tmcfg.Config {
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
