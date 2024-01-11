package oracle

import (
	"fmt"
	"strings"
	"time"

	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"gopkg.in/yaml.v3"

	oracleconfig "github.com/skip-mev/slinky/oracle/config"
	"github.com/skip-mev/slinky/providers/coinbase"
	"github.com/skip-mev/slinky/providers/coingecko"
	oracletypes "github.com/skip-mev/slinky/x/oracle/types"

	"github.com/spf13/cast"
	"github.com/spf13/viper"
)

// WrappedOracleConfig is the base config for both out-of-process and in-process oracles.
// If the oracle is to be configured out-of-process in base-app, a grpc-client of
// the grpc-server running at RemoteAddress is instantiated, otherwise, an in-process
// local client oracle is instantiated. Note, that you can only have one oracle
// running at a time.
type WrappedOracleConfig struct {
	// Enabled specifies whether the side-car oracle needs to be run.
	Enabled bool `mapstructure:"enabled" toml:"enabled"`

	// InProcess specifies whether the oracle configured, is currently running as a remote grpc-server, or will be run in process
	InProcess bool `mapstructure:"in_process" toml:"in_process"`

	// RemoteAddress is the address of the remote oracle server (if it is running out-of-process)
	RemoteAddress string `mapstructure:"remote_address" toml:"remote_address"`

	// ClientTimeout is the time that the client is willing to wait for responses from the oracle before timing out.
	ClientTimeout time.Duration `mapstructure:"client_timeout" toml:"client_timeout"`

	// UpdateInterval is the interval at which the oracle will fetch prices from providers
	UpdateInterval time.Duration `mapstructure:"update_interval" toml:"update_interval"`

	// Providers is the list of providers that the oracle will fetch prices from.
	Providers []oracleconfig.ProviderConfig `mapstructure:"providers" toml:"providers"`

	// CurrencyPairs is the list of currency pairs that the oracle will fetch prices for.
	CurrencyPairs []oracletypes.CurrencyPair `mapstructure:"currency_pairs" toml:"currency_pairs"`

	// Production specifies whether the oracle is running in production mode. This is used to
	// determine whether the oracle should be run in debug mode or not.
	Production bool `mapstructure:"production" toml:"production"`

	// MetricsConfig is the metrics configurations for the oracle. This configuration object allows for
	// metrics tracking of the oracle and the interaction between the oracle and the app.
	MetricsConfig oracleconfig.MetricsConfig `mapstructure:"metrics"`

	// Config for the CoinGecko provider
	CoinGeckoConfig wrappedCoinGeckoConfig `mapstructure:"coingecko"`

	// Config for the CoinBase provider
	CoinBaseConfig wrappedCoinBaseConfig `mapstructure:"coinbase"`
}

func DefaultConfig() WrappedOracleConfig {
	return WrappedOracleConfig{
		Enabled:        false,
		Production:     false,
		InProcess:      true,
		RemoteAddress:  "",
		ClientTimeout:  time.Second * 2,
		UpdateInterval: time.Second * 2,
		Providers:      nil,
		CurrencyPairs:  nil,
		MetricsConfig: oracleconfig.MetricsConfig{
			PrometheusServerAddress: "localhost:8000",
			OracleMetrics: oracleconfig.OracleMetricsConfig{
				Enabled: true,
			},
			AppMetrics: oracleconfig.AppMetricsConfig{
				Enabled:              false,
				ValidatorConsAddress: "",
			},
		},
	}
}

var (
	DefaultConfigTemplate = fmt.Sprintf(`

###############################################################################
###                                  Oracle                                 ###
###############################################################################

[oracle]

# Enabled specifies whether the side-car oracle needs to be run.
enabled = "{{ .OracleConfig.Enabled }}"

# InProcess specifies whether the oracle configured, is currently running as a remote grpc-server, or will be run in process
in_process = "{{ .OracleConfig.InProcess }}"

# Production specifies whether the oracle is running in production mode. This is used to
# determine whether the oracle should be run in debug mode or not.
production = "{{ .OracleConfig.Production }}"

# RemoteAddress is the address of the remote oracle server (if it is running out-of-process)
remote_address = "{{ .OracleConfig.RemoteAddress }}"

# ClientTimeout is the time that the client is willing to wait for responses from the oracle before timing out.
client_timeout = "{{ .OracleConfig.ClientTimeout }}"

# UpdateInterval is the interval at which the oracle will fetch prices from providers
update_interval = "{{ .OracleConfig.UpdateInterval }}"

# Uncomment this section to enable %s price fetching.
# [[oracle.providers]]
# name = "%s"
# timeout = "500ms"  # Replace "500ms" with your desired timeout duration.
# interval = "1s"  # Replace "1s" with your desired update interval duration.
# max_queries = 5  # Replace "5" with your desired maximum number of queries per update interval.

# Uncomment this section to enable %s price fetching.
# [[providers]]
# name = "%s"
# timeout = "500ms"
# interval = "1s"
# max_queries = 1 # CoinGecko is atomic so it can fetch all prices in a single request.

# [[oracle.currency_pairs]]
# base = "BITCOIN"
# quote = "USD"

# [[currency_pairs]]
# base = "ETHEREUM"
# quote = "USD"

[oracle.metrics]
# PrometheusServerAddress is the address of the prometheus server that the oracle will expose metrics to
prometheus_server_address = "{{ .OracleConfig.MetricsConfig.PrometheusServerAddress }}"

[oracle.metrics.oracle_metrics]
# Enabled indicates whether metrics should be enabled.
enabled = "{{ .OracleConfig.MetricsConfig.OracleMetrics.Enabled }}"

[oracle.metrics.app_metrics]
# Enabled indicates whether app side metrics should be enabled.
enabled = "{{ .OracleConfig.MetricsConfig.AppMetrics.Enabled }}"

# ValidatorConsAddress is the validator's consensus address. Validator's must register their
# consensus address in order to enable app side metrics.
validator_cons_address = "{{ .OracleConfig.MetricsConfig.AppMetrics.ValidatorConsAddress }}"

[oracle.coingecko]
# APIKey is the API key used to make requests to the CoinGecko API.
api_key = "{{ .OracleConfig.CoinGeckoConfig.APIKey }}"

# SupportedBases maps an oracle base currency to a CoinGecko base currency.
[oracle.coingecko.supported_bases]
"BITCOIN"  = "bitcoin"
"ETHEREUM" = "ethereum"
"ATOM"     = "cosmos"
"SOLANA"   = "solana"
"POLKADOT" = "polkadot"
"DYDX"     = "dydx-chain"

# SupportedQuotes maps an oracle quote currency to a CoinGecko quote currency.
[oracle.coingecko.supported_quotes]
"USD"      = "usd"
"ETHEREUM" = "eth"

[oracle.coinbase.symbol_map]
"BITCOIN"  = "BTC"
"USD"      = "USD"
"ETHEREUM" = "ETH"
"ATOM"     = "ATOM"
"SOLANA"   = "SOL"
"POLKADOT" = "DOT"
"DYDX"     = "DYDX"

`,
		coinbase.Name, coinbase.Name,
		coingecko.Name, coingecko.Name,
	)
)

func ReadWrappedOracleConfig(appOpts servertypes.AppOptions) WrappedOracleConfig {
	v := interface{}(appOpts)
	viper := v.(*viper.Viper)

	var providers []oracleconfig.ProviderConfig
	viper.UnmarshalKey("oracle.providers", &providers)
	for i, p := range providers {
		p.Path = "dummy"
		providers[i] = p
	}

	var currencyPairs []oracletypes.CurrencyPair
	viper.UnmarshalKey("oracle.currency_pairs", &currencyPairs)

	var coingeckoSupportedBases map[string]string
	viper.UnmarshalKey("oracle.coingecko.supported_bases", &coingeckoSupportedBases)
	var coingeckoSupportedQuotes map[string]string
	viper.UnmarshalKey("oracle.coingecko.supported_quotes", &coingeckoSupportedQuotes)
	var coinbaseSymbolMap map[string]string
	viper.UnmarshalKey("oracle.coinbase.symbol_map", &coinbaseSymbolMap)

	config := WrappedOracleConfig{
		Enabled:        cast.ToBool(appOpts.Get("oracle.enabled")),
		InProcess:      cast.ToBool(appOpts.Get("oracle.in_process")),
		Production:     cast.ToBool(appOpts.Get("oracle.production")),
		RemoteAddress:  cast.ToString(appOpts.Get("oracle.remote_address")),
		ClientTimeout:  cast.ToDuration(appOpts.Get("oracle.client_timeout")),
		UpdateInterval: cast.ToDuration(appOpts.Get("oracle.update_interval")),
		Providers:      providers,
		CurrencyPairs:  currencyPairs,
		MetricsConfig: oracleconfig.MetricsConfig{
			PrometheusServerAddress: cast.ToString(appOpts.Get("oracle.metrics.prometheus_server_address")),
			OracleMetrics: oracleconfig.OracleMetricsConfig{
				Enabled: cast.ToBool(appOpts.Get("oracle.metrics.oracle_metrics.enabled")),
			},
			AppMetrics: oracleconfig.AppMetricsConfig{
				Enabled:              cast.ToBool(appOpts.Get("oracle.metrics.app_metrics.enabled")),
				ValidatorConsAddress: cast.ToString(appOpts.Get("oracle.metrics.app_metrics.validator_cons_address")),
			},
		},
		CoinGeckoConfig: wrappedCoinGeckoConfig{
			APIKey:          cast.ToString(appOpts.Get("oracle.coingecko.api_key")),
			SupportedBases:  coingeckoSupportedBases,
			SupportedQuotes: coingeckoSupportedQuotes,
		},
		CoinBaseConfig: wrappedCoinBaseConfig{
			SymbolMap: coinbaseSymbolMap,
		},
	}

	bz, _ := yaml.Marshal(config)
	fmt.Println(string(bz))

	return config
}

func (c WrappedOracleConfig) GetConfigs() (oracleconfig.OracleConfig, oracleconfig.MetricsConfig) {
	return oracleconfig.OracleConfig{
		Enabled:        c.Enabled,
		InProcess:      c.InProcess,
		Production:     c.Production,
		RemoteAddress:  c.RemoteAddress,
		ClientTimeout:  c.ClientTimeout,
		UpdateInterval: c.UpdateInterval,
		Providers:      c.Providers,
		CurrencyPairs:  c.CurrencyPairs,
	}, c.MetricsConfig
}

func (c WrappedOracleConfig) GetCoinGeckoConfig() coingecko.Config {
	return coingecko.Config{
		APIKey:          c.CoinGeckoConfig.APIKey,
		SupportedBases:  keyToUpperCase(c.CoinGeckoConfig.SupportedBases),
		SupportedQuotes: keyToUpperCase(c.CoinGeckoConfig.SupportedQuotes),
	}
}

func (c WrappedOracleConfig) GetCoinBaseConfig() coinbase.Config {
	return coinbase.Config{
		SymbolMap: keyToUpperCase(c.CoinBaseConfig.SymbolMap),
	}
}

func keyToUpperCase(m map[string]string) map[string]string {
	m2 := make(map[string]string)
	for key, val := range m {
		m2[strings.ToUpper(key)] = val
	}

	return m2
}

func (c WrappedOracleConfig) ValidateBasic() error {
	oracleConfig, metricConfig := c.GetConfigs()
	if err := oracleConfig.ValidateBasic(); err != nil {
		return err
	}

	if err := metricConfig.ValidateBasic(); err != nil {
		return err
	}

	return nil
}
