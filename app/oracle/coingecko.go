package oracle

// NOTE: To determine the list of supported base currencies, please see
// https://api.coingecko.com/api/v3/coins/list. To see the supported quote
// currencies, please see https://api.coingecko.com/api/v3/simple/supported_vs_currencies.
// Not all base currencies are allowed to be used as quote currencies.

// wrappedCoinGeckoConfig is the config struct for the CoinGecko provider.
type wrappedCoinGeckoConfig struct {
	// APIKey is the API key used to make requests to the CoinGecko API.
	APIKey string `mapstructure:"api_key" toml:"api_key"`

	// SupportedBases maps an oracle base currency to a CoinGecko base currency.
	SupportedBases map[string]string `mapstructure:"supported_bases" toml:"supported_bases"`

	// SupportedQuotes maps an oracle quote currency to a CoinGecko quote currency.
	SupportedQuotes map[string]string `mapstructure:"supported_quotes" toml:"supported_quotes"`
}
