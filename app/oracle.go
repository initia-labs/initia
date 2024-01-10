package app

import (
	"context"
	"fmt"
	"math/big"
	"net/http"

	"go.uber.org/zap"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/skip-mev/slinky/abci/preblock/oracle/math"
	"github.com/skip-mev/slinky/aggregator"
	"github.com/skip-mev/slinky/oracle/config"
	slinkymath "github.com/skip-mev/slinky/pkg/math"
	"github.com/skip-mev/slinky/providers/base"
	"github.com/skip-mev/slinky/providers/base/handlers"
	"github.com/skip-mev/slinky/providers/base/metrics"
	"github.com/skip-mev/slinky/providers/coinbase"
	"github.com/skip-mev/slinky/providers/coingecko"
	"github.com/skip-mev/slinky/providers/static"
	providertypes "github.com/skip-mev/slinky/providers/types"
	oracletypes "github.com/skip-mev/slinky/x/oracle/types"

	mstakingkeeper "github.com/initia-labs/initia/x/mstaking/keeper"
	mstakingtypes "github.com/initia-labs/initia/x/mstaking/types"
)

type wrappedStakingKeeper struct {
	mstakingkeeper.Keeper
}

func (w wrappedStakingKeeper) ValidatorByConsAddr(ctx context.Context, addr sdk.ConsAddress) (stakingtypes.ValidatorI, error) {
	val, err := w.GetValidatorByConsAddr(ctx, addr)
	return wrappedValidator{val}, err
}

type wrappedValidator struct {
	mstakingtypes.Validator
}

// GetMinSelfDelegation is required to fulfill the cosmos-sdk.x.staking.ValidatorI interface
// but the oracle doesn't use it.
func (w wrappedValidator) GetMinSelfDelegation() sdkmath.Int { panic("not implemented") }

// GetStatus is required to fulfill the cosmos-sdk.x.staking.ValidatorI interface
// but the oracle doesn't use it.
func (w wrappedValidator) GetStatus() stakingtypes.BondStatus { panic("not implemented") }

// SharesFromTokensTruncated is required to fulfill the cosmos-sdk.x.staking.ValidatorI interface
// but the oracle doesn't use it.
func (w wrappedValidator) SharesFromTokensTruncated(shares sdkmath.Int) (sdkmath.LegacyDec, error) {
	panic("not implemented")
}

func (w wrappedValidator) TokensFromSharesRoundUp(dec sdkmath.LegacyDec) sdkmath.LegacyDec {
	panic("not implemented")
}

func (w wrappedValidator) TokensFromShares(dec sdkmath.LegacyDec) sdkmath.LegacyDec {
	panic("not implemented")
}

func (w wrappedValidator) GetTokens() sdkmath.Int {
	panic("not implemented")
}

func (w wrappedValidator) GetDelegatorShares() sdkmath.LegacyDec {
	panic("not implemented")
}

func (w wrappedValidator) TokensFromSharesTruncated(dec sdkmath.LegacyDec) sdkmath.LegacyDec {
	panic("not implemented")
}

func (w wrappedValidator) GetConsAddr() ([]byte, error) {
	panic("not implemented")
}

func (w wrappedValidator) GetBondedTokens() sdkmath.Int {
	var tokens sdkmath.Int
	for _, coin := range w.BondedTokens() {
		tokens = tokens.Add(sdkmath.NewIntFromBigInt(coin.Amount.BigInt()))
	}
	return tokens
}

// SharesFromTokens is required to fulfill the cosmos-sdk.x.staking.ValidatorI interface
// but the oracle doesn't use it.
func (w wrappedValidator) SharesFromTokens(_ sdkmath.Int) (sdkmath.LegacyDec, error) {
	panic("not implemented")
}

// GetOracleAggregationFN returns the vote aggregation function used by the oracle
// We use the default stake weighted median w/ a required greater than 2/3 stake threshold for acceptance
func (app *InitiaApp) GetOracleAggregationFN() aggregator.AggregateFnFromContext[string, map[oracletypes.CurrencyPair]*big.Int] {
	return math.VoteWeightedMedianFromContext(
		app.Logger(),
		// Their staking keeper doesn't have TotalBondedTokens method
		wrappedStakingKeeper{*app.StakingKeeper},
		math.DefaultPowerThreshold,
	)
}

// DefaultAPIProviderFactory returns a sample implementation of the provider factory. This provider
// factory function only returns providers that are API based.
func DefaultAPIProviderFactory() providertypes.ProviderFactory[oracletypes.CurrencyPair, *big.Int] {
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
		for i, p := range oracleCfg.Providers {
			if providers[i], err = providerFromProviderConfig(logger, p, cps, m); err != nil {
				return nil, err
			}
		}

		return providers, nil
	}
}

func providerFromProviderConfig(
	logger *zap.Logger,
	cfg config.ProviderConfig,
	cps []oracletypes.CurrencyPair,
	m metrics.APIMetrics,
) (providertypes.Provider[oracletypes.CurrencyPair, *big.Int], error) {
	// Validate the provider config.
	err := cfg.ValidateBasic()
	if err != nil {
		return nil, err
	}

	// Create the underlying client that will be used to fetch data from the API. This client
	// will limit the number of concurrent connections and uses the configured timeout to
	// ensure requests do not hang.
	maxCons := slinkymath.Min(len(cps), cfg.MaxQueries)
	client := &http.Client{
		Transport: &http.Transport{MaxConnsPerHost: maxCons},
		Timeout:   cfg.Timeout,
	}

	var (
		apiDataHandler handlers.APIDataHandler[oracletypes.CurrencyPair, *big.Int]
		requestHandler handlers.RequestHandler
	)

	switch cfg.Name {
	case "coingecko":
		apiDataHandler, err = coingecko.NewCoinGeckoAPIHandler(cfg)
	case "coinbase":
		apiDataHandler, err = coinbase.NewCoinBaseAPIHandler(cfg)

		requestHandler = static.NewStaticMockClient()
	default:
		return nil, fmt.Errorf("unknown provider: %s", cfg.Name)
	}
	if err != nil {
		return nil, err
	}

	if apiDataHandler == nil {
		return nil, fmt.Errorf("failed to create api data handler for provider %s", cfg.Name)
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
		cfg,
		apiQueryHandler,
		cps,
	)
}
