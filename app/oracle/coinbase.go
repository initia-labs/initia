package oracle

// Config is the configuration for the Coinbase APIDataHandler.
type wrappedCoinBaseConfig struct {
	// SymbolMap maps the oracle's equivalent of an asset to the expected coinbase
	// representation of the asset.
	SymbolMap map[string]string `mapstructure:"symbol_map" toml:"symbol_map"`
}
