package oracle

import (
	"time"

	servertypes "github.com/cosmos/cosmos-sdk/server/types"

	oracleconfig "github.com/skip-mev/slinky/oracle/config"

	"github.com/spf13/cast"
)

func DefaultConfig() oracleconfig.AppConfig {
	return oracleconfig.AppConfig{
		Enabled:                 false,
		OracleAddress:           "",
		ClientTimeout:           time.Second * 2,
		MetricsEnabled:          false,
		PrometheusServerAddress: "localhost:8000",
	}
}

func ReadOracleConfig(appOpts servertypes.AppOptions) oracleconfig.AppConfig {
	config := oracleconfig.AppConfig{
		Enabled:                 cast.ToBool(appOpts.Get("oracle.enabled")),
		OracleAddress:           cast.ToString(appOpts.Get("oracle.oracle_address")),
		ClientTimeout:           cast.ToDuration(appOpts.Get("oracle.client_timeout")),
		MetricsEnabled:          cast.ToBool(appOpts.Get("oracle.metrics_enabled")),
		PrometheusServerAddress: cast.ToString(appOpts.Get("oracle.prometheus_server_address")),
	}

	return config
}
