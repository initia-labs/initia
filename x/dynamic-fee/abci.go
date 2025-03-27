package dynamicfee

import (
	"context"
	"time"

	"github.com/initia-labs/initia/x/dynamic-fee/keeper"
	"github.com/initia-labs/initia/x/dynamic-fee/types"

	"github.com/cosmos/cosmos-sdk/telemetry"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func EndBlocker(ctx context.Context, k keeper.Keeper) error {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyEndBlocker)

	// update base fee
	return k.UpdateBaseGasPrice(sdk.UnwrapSDKContext(ctx))
}
