package oracle

import (
	"fmt"
	"math/big"
	"net/http"

	"go.uber.org/zap"

	"cosmossdk.io/log"

	"github.com/skip-mev/slinky/abci/preblock/oracle/math"
	"github.com/skip-mev/slinky/aggregator"
	"github.com/skip-mev/slinky/oracle/config"
	slinkymath "github.com/skip-mev/slinky/pkg/math"
	"github.com/skip-mev/slinky/providers/base"
	"github.com/skip-mev/slinky/providers/base/handlers"
	"github.com/skip-mev/slinky/providers/base/metrics"
	"github.com/skip-mev/slinky/providers/coinbase"
	"github.com/skip-mev/slinky/providers/coingecko"
	providertypes "github.com/skip-mev/slinky/providers/types"
	oracletypes "github.com/skip-mev/slinky/x/oracle/types"

	stakingkeeper "github.com/initia-labs/initia/x/mstaking/keeper"
)

func GetOracleAggregationFN(logger log.Logger, stakingKeeper *stakingkeeper.Keeper) aggregator.AggregateFnFromContext[string, map[oracletypes.CurrencyPair]*big.Int] {
	return math.VoteWeightedMedianFromContext(
		logger,
		stakingkeeper.NewCompatibilityKeeper(stakingKeeper),
		math.DefaultPowerThreshold,
	)
}

// DefaultAPIProviderFactory returns a sample implementation of the provider factory. This provider
// factory function only returns providers that are API based.
func DefaultAPIProviderFactory(wrappedOracleConfig WrappedOracleConfig) providertypes.ProviderFactory[oracletypes.CurrencyPair, *big.Int] {
	return func(logger *zap.Logger, oracleCfg config.OracleConfig, metricsCfg config.OracleMetricsConfig) ([]providertypes.Provider[oracletypes.CurrencyPair, *big.Int], error) {
		if err := oracleCfg.ValidateBasic(); err != nil {
			return nil, err
		}

		m := metrics.NewAPIMetricsFromConfig(metricsCfg)
		cps := oracleCfg.CurrencyPairs

		var (
			err       error
			providers = make([]providertypes.Provider[oracletypes.CurrencyPair, *big.Int], len(oracleCfg.Providers))
		)
		for i, providerCfg := range oracleCfg.Providers {
			if providers[i], err = providerFromProviderConfig(logger, providerCfg, wrappedOracleConfig, cps, m); err != nil {
				return nil, err
			}
		}

		return providers, nil
	}
}

func providerFromProviderConfig(
	logger *zap.Logger,
	providerCfg config.ProviderConfig,
	wrappedOracleCfg WrappedOracleConfig,
	cps []oracletypes.CurrencyPair,
	m metrics.APIMetrics,
) (providertypes.Provider[oracletypes.CurrencyPair, *big.Int], error) {
	// Validate the provider config.
	err := providerCfg.ValidateBasic()
	if err != nil {
		return nil, err
	}

	// Create the underlying client that will be used to fetch data from the API. This client
	// will limit the number of concurrent connections and uses the configured timeout to
	// ensure requests do not hang.
	maxCons := slinkymath.Min(len(cps), providerCfg.MaxQueries)
	client := &http.Client{
		Transport: &http.Transport{MaxConnsPerHost: maxCons},
		Timeout:   providerCfg.Timeout,
	}

	var (
		apiDataHandler handlers.APIDataHandler[oracletypes.CurrencyPair, *big.Int]
		requestHandler handlers.RequestHandler
	)

	switch providerCfg.Name {
	case "coingecko":
		cfg := wrappedOracleCfg.GetCoinGeckoConfig()
		if err := cfg.ValidateBasic(); err != nil {
			return nil, err
		}

		apiDataHandler = &coingecko.CoinGeckoAPIHandler{Config: cfg}
	case "coinbase":
		cfg := wrappedOracleCfg.GetCoinBaseConfig()
		if err := cfg.ValidateBasic(); err != nil {
			return nil, err
		}

		apiDataHandler = &coinbase.CoinBaseAPIHandler{Config: cfg}
	default:
		return nil, fmt.Errorf("unknown provider: %s", providerCfg.Name)
	}
	if err != nil {
		return nil, err
	}

	if apiDataHandler == nil {
		return nil, fmt.Errorf("failed to create api data handler for provider %s", providerCfg.Name)
	}

	// If a custom request handler is not provided, create a new default one.
	if requestHandler == nil {
		requestHandler = handlers.NewRequestHandlerImpl(client)
	}

	// Create the API query handler which encapsulates all the fetching and parsing logic.
	apiQueryHandler, err := handlers.NewAPIQueryHandler[oracletypes.CurrencyPair, *big.Int](
		logger,
		requestHandler,
		apiDataHandler,
		m,
	)
	if err != nil {
		return nil, err
	}

	// Create the provider.
	return base.NewProvider[oracletypes.CurrencyPair, *big.Int](
		logger,
		providerCfg,
		apiQueryHandler,
		cps,
	)
}
