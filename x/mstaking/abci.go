package staking

import (
	"context"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/cosmos-sdk/telemetry"

	"github.com/initia-labs/initia/x/mstaking/keeper"
	"github.com/initia-labs/initia/x/mstaking/types"
)

// BeginBlocker will persist the current header and validator set as a historical entry
// and prune the oldest entry based on the HistoricalEntries parameter
func BeginBlocker(ctx context.Context, k keeper.Keeper) error {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyBeginBlocker)
	return k.TrackHistoricalInfo(ctx)
}

// EndBlocker called every block, update validator set
func EndBlocker(ctx context.Context, k keeper.Keeper) ([]abci.ValidatorUpdate, error) {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyEndBlocker)
	return k.BlockValidatorUpdates(ctx)
}
