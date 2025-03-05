package reward

import (
	"context"
	"time"

	"cosmossdk.io/math"
	"github.com/initia-labs/initia/v1/x/reward/keeper"
	"github.com/initia-labs/initia/v1/x/reward/types"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BeginBlocker mints new tokens for the previous block.
func BeginBlocker(ctx context.Context, k keeper.Keeper) error {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyBeginBlocker)

	// fetch stored minter & params
	lastReleaseTimestamp, err := k.GetLastReleaseTimestamp(ctx)
	if err != nil {
		return err
	}
	lastDilutionTimestamp, err := k.GetLastDilutionTimestamp(ctx)
	if err != nil {
		return err
	}
	annualProvisions, err := k.GetAnnualProvisions(ctx)
	if err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	timeDiff := sdkCtx.BlockTime().Sub(lastReleaseTimestamp)
	if timeDiff <= 0 {
		return nil
	}

	params, err := k.GetParams(ctx)
	if err != nil {
		return err
	}

	if !params.ReleaseEnabled {
		if err := k.SetLastReleaseTimestamp(ctx, sdkCtx.BlockTime()); err != nil {
			return err
		}
		if err := k.SetLastDilutionTimestamp(ctx, sdkCtx.BlockTime()); err != nil {
			return err
		}

		return nil
	}

	remainRewardAmt := k.GetRemainRewardAmount(ctx, params.RewardDenom)
	blockProvisionAmt := math.MinInt(remainRewardAmt, annualProvisions.Mul(math.LegacyNewDec(int64(timeDiff)).QuoInt64(int64(time.Hour*24*365))).TruncateInt())
	blockProvisionCoin := sdk.NewCoin(params.RewardDenom, blockProvisionAmt)
	blockProvisionCoins := sdk.NewCoins(blockProvisionCoin)

	// send the minted coins to the fee collector account
	err = k.AddCollectedFees(ctx, blockProvisionCoins)
	if err != nil {
		return err
	}

	// update release rate on every year
	if sdkCtx.BlockTime().Sub(lastDilutionTimestamp) >= params.DilutionPeriod {
		// dilute release rate
		releaseRate := params.ReleaseRate.Sub(params.ReleaseRate.Mul(params.DilutionRate))
		if releaseRate.IsNegative() {
			releaseRate = math.LegacyZeroDec()
		}

		// update store
		if err := k.SetReleaseRate(ctx, releaseRate); err != nil {
			return err
		}
		if err := k.SetLastDilutionTimestamp(ctx, sdkCtx.BlockTime()); err != nil {
			return err
		}
	}

	// update last mint timestamp
	if err := k.SetLastReleaseTimestamp(ctx, sdkCtx.BlockTime()); err != nil {
		return err
	}

	if blockProvisionAmt.IsInt64() {
		defer telemetry.ModuleSetGauge(types.ModuleName, float32(blockProvisionAmt.Int64()), "reward_tokens")
	}

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeReward,
			sdk.NewAttribute(types.AttributeKeyReleaseRate, params.ReleaseRate.String()),
			sdk.NewAttribute(types.AttributeKeyAnnualProvisions, annualProvisions.String()),
			sdk.NewAttribute(sdk.AttributeKeyAmount, blockProvisionAmt.String()),
		),
	)

	return nil
}
