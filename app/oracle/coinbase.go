package oracle

import "fmt"

// Config is the configuration for the Coinbase APIDataHandler.
type wrappedCoinBaseConfig struct {
	// SymbolMap maps the oracle's equivalent of an asset to the expected coinbase
	// representation of the asset.
	SymbolMap map[string]string `mapstructure:"symbol_map" toml:"symbol_map"`
}

func defaultCoinBaseConfig() wrappedCoinBaseConfig {
	return wrappedCoinBaseConfig{
		SymbolMap: map[string]string{
			"BITCOIN":  "BTC",
			"USD":      "USD",
			"ETHEREUM": "ETH",
			"ATOM":     "ATOM",
			"SOLANA":   "SOL",
			"POLKADOT": "DOT",
			"DYDX":     "DYDX",
		},
	}
}

func (cfg wrappedCoinBaseConfig) SymbolMapString() string {
	str := ""
	for key, val := range cfg.SymbolMap {
		str += fmt.Sprintf("\"%s\" = \"%s\"\n", key, val)
	}

	return str
}
