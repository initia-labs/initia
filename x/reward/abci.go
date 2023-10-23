package reward

import (
	"time"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/initia-labs/initia/x/reward/keeper"
	"github.com/initia-labs/initia/x/reward/types"
)

// BeginBlocker mints new tokens for the previous block.
func BeginBlocker(ctx sdk.Context, k keeper.Keeper) {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyBeginBlocker)

	// fetch stored minter & params
	lastReleaseTimestamp := k.GetLastReleaseTimestamp(ctx)
	lastDilutionTimestamp := k.GetLastDilutionTimestamp(ctx)
	annualProvisions := k.GetAnnualProvisions(ctx)

	timeDiff := ctx.BlockTime().Sub(lastReleaseTimestamp)
	if timeDiff <= 0 {
		return
	}

	params := k.GetParams(ctx)
	if !params.ReleaseEnabled {
		k.SetLastReleaseTimestamp(ctx, ctx.BlockTime())
		k.SetLastDilutionTimestamp(ctx, ctx.BlockTime())

		return
	}

	remainRewardAmt := k.GetRemainRewardAmount(ctx, params.RewardDenom)
	blockProvisionAmt := sdk.MinInt(remainRewardAmt, annualProvisions.Mul(sdk.NewDec(int64(timeDiff)).QuoInt64(int64(time.Hour*24*365))).TruncateInt())
	blockProvisionCoin := sdk.NewCoin(params.RewardDenom, blockProvisionAmt)
	blockProvisionCoins := sdk.NewCoins(blockProvisionCoin)

	// send the minted coins to the fee collector account
	err := k.AddCollectedFees(ctx, blockProvisionCoins)
	if err != nil {
		panic(err)
	}

	// update release rate on every year
	if ctx.BlockTime().Sub(lastDilutionTimestamp) >= time.Duration(params.DilutionPeriod) {
		// dilute release rate
		releaseRate := params.ReleaseRate.Sub(params.ReleaseRate.Mul(params.DilutionRate))

		// update store
		if err := k.SetReleaseRate(ctx, releaseRate); err != nil {
			panic(err)
		}
		k.SetLastDilutionTimestamp(ctx, ctx.BlockTime())
	}

	// update last mint timestamp
	k.SetLastReleaseTimestamp(ctx, ctx.BlockTime())

	if blockProvisionAmt.IsInt64() {
		defer telemetry.ModuleSetGauge(types.ModuleName, float32(blockProvisionAmt.Int64()), "reward_tokens")
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeReward,
			sdk.NewAttribute(types.AttributeKeyReleaseRate, params.ReleaseRate.String()),
			sdk.NewAttribute(types.AttributeKeyAnnualProvisions, annualProvisions.String()),
			sdk.NewAttribute(sdk.AttributeKeyAmount, blockProvisionAmt.String()),
		),
	)
}
