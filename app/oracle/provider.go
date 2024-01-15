package oracle

import (
	"math/big"

	"cosmossdk.io/log"

	"github.com/skip-mev/slinky/abci/preblock/oracle/math"
	"github.com/skip-mev/slinky/aggregator"
	"github.com/skip-mev/slinky/service"
	oracletypes "github.com/skip-mev/slinky/x/oracle/types"

	stakingkeeper "github.com/initia-labs/initia/x/mstaking/keeper"
)

// NewOracleClient reads a config and instantiates either a grpc-client.
func NewOracleClient(
	oracleCfg WrappedOracleConfig,
) (service.OracleService, error) {
	if !oracleCfg.Enabled {
		return service.NewNoopOracleService(), nil
	}

	return newGRPCClient(oracleCfg.RemoteAddress, oracleCfg.ClientTimeout), nil
}

func GetOracleAggregationFN(logger log.Logger, stakingKeeper *stakingkeeper.Keeper) aggregator.AggregateFnFromContext[string, map[oracletypes.CurrencyPair]*big.Int] {
	return math.VoteWeightedMedianFromContext(
		logger,
		stakingkeeper.NewCompatibilityKeeper(stakingKeeper),
		math.DefaultPowerThreshold,
	)
}
