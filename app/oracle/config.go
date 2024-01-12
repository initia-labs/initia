package oracle

import (
	"fmt"
	"time"

	servertypes "github.com/cosmos/cosmos-sdk/server/types"

	oracleconfig "github.com/skip-mev/slinky/oracle/config"

	"github.com/spf13/cast"
)

// WrappedOracleConfig is the base config for both out-of-process and in-process oracles.
// If the oracle is to be configured out-of-process in base-app, a grpc-client of
// the grpc-server running at RemoteAddress is instantiated, otherwise, an in-process
// local client oracle is instantiated. Note, that you can only have one oracle
// running at a time.
type WrappedOracleConfig struct {
	// Enabled specifies whether the side-car oracle needs to be run.
	Enabled bool `mapstructure:"enabled" toml:"enabled"`

	// Production specifies whether the oracle is running in production mode. This is used to
	// determine whether the oracle should be run in debug mode or not.
	Production bool `mapstructure:"production" toml:"production"`

	// RemoteAddress is the address of the remote oracle server (if it is running out-of-process)
	RemoteAddress string `mapstructure:"remote_address" toml:"remote_address"`

	// ClientTimeout is the time that the client is willing to wait for responses from the oracle before timing out.
	ClientTimeout time.Duration `mapstructure:"client_timeout" toml:"client_timeout"`

	// MetricsConfig is the metrics configurations for the oracle. This configuration object allows for
	// metrics tracking of the interaction between the oracle and the app.
	MetricsConfig WrappedMetricConfig `mapstructure:"metrics"`
}

type WrappedMetricConfig struct {
	// Enabled indicates whether app side metrics should be enabled.
	Enabled bool `mapstructure:"enabled" toml:"enabled"`

	// PrometheusServerAddress is the address of the prometheus server that the oracle will expose metrics to
	PrometheusServerAddress string `mapstructure:"prometheus_server_address" toml:"prometheus_server_address"`

	// ValidatorConsAddress is the validator's consensus address. Validator's must register their
	// consensus address in order to enable app side metrics.
	ValidatorConsAddress string `mapstructure:"validator_cons_address" toml:"validator_cons_address"`
}

func (wmc WrappedMetricConfig) ToAppMetricConfig() oracleconfig.AppMetricsConfig {
	return oracleconfig.AppMetricsConfig{
		Enabled:              wmc.Enabled,
		ValidatorConsAddress: wmc.ValidatorConsAddress,
	}
}

func DefaultConfig() WrappedOracleConfig {
	return WrappedOracleConfig{
		Enabled:       false,
		Production:    false,
		RemoteAddress: "",
		ClientTimeout: time.Second * 2,
		MetricsConfig: WrappedMetricConfig{
			Enabled:                 false,
			PrometheusServerAddress: "localhost:8000",
			ValidatorConsAddress:    "",
		},
	}
}

var (
	DefaultConfigTemplate = `

###############################################################################
###                                  Oracle                                 ###
###############################################################################

[oracle]

# Enabled specifies whether the side-car oracle needs to be run.
enabled = "{{ .OracleConfig.Enabled }}"

# Production specifies whether the oracle is running in production mode. This is used to
# determine whether the oracle should be run in debug mode or not.
production = "{{ .OracleConfig.Production }}"

# RemoteAddress is the address of the remote oracle server (if it is running out-of-process)
remote_address = "{{ .OracleConfig.RemoteAddress }}"

# ClientTimeout is the time that the client is willing to wait for responses from the oracle before timing out.
client_timeout = "{{ .OracleConfig.ClientTimeout }}"

[oracle.metrics]

# Enabled indicates whether app side metrics should be enabled.
enabled = "{{ .OracleConfig.MetricsConfig.Enabled }}"

# PrometheusServerAddress is the address of the prometheus server that the oracle will expose metrics to
prometheus_server_address = "{{ .OracleConfig.MetricsConfig.PrometheusServerAddress }}"

# ValidatorConsAddress is the validator's consensus address. Validator's must register their
# consensus address in order to enable app side metrics.
validator_cons_address = "{{ .OracleConfig.MetricsConfig.ValidatorConsAddress }}"
`
)

func ReadWrappedOracleConfig(appOpts servertypes.AppOptions) WrappedOracleConfig {
	config := WrappedOracleConfig{
		Enabled:       cast.ToBool(appOpts.Get("oracle.enabled")),
		Production:    cast.ToBool(appOpts.Get("oracle.production")),
		RemoteAddress: cast.ToString(appOpts.Get("oracle.remote_address")),
		ClientTimeout: cast.ToDuration(appOpts.Get("oracle.client_timeout")),
		MetricsConfig: WrappedMetricConfig{
			Enabled:                 cast.ToBool(appOpts.Get("oracle.metrics.enabled")),
			PrometheusServerAddress: cast.ToString(appOpts.Get("oracle.metrics.prometheus_server_address")),
			ValidatorConsAddress:    cast.ToString(appOpts.Get("oracle.metrics.validator_cons_address")),
		},
	}

	return config
}

func (c WrappedOracleConfig) ValidateBasic() error {
	if !c.Enabled {
		return nil
	}

	if len(c.RemoteAddress) == 0 {
		return fmt.Errorf("must supply a remote address if the oracle is running out of process")
	}

	if c.ClientTimeout <= 0 {
		return fmt.Errorf("oracle client timeout must be greater than 0")
	}

	if c.MetricsConfig.Enabled && len(c.MetricsConfig.PrometheusServerAddress) == 0 {
		return fmt.Errorf("must supply a prometheus server address if metrics are enabled")
	}

	return nil
}
