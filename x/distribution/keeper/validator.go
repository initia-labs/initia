package keeper

import (
	"context"
	"errors"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	customtypes "github.com/initia-labs/initia/x/distribution/types"
	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"
)

// initialize rewards for a new validator
func (k Keeper) initializeValidator(ctx context.Context, val stakingtypes.ValidatorI) error {
	valAddr, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(val.GetOperator())
	if err != nil {
		return err
	}

	// set initial historical rewards (period 0) with reference count of 1
	err = k.ValidatorHistoricalRewards.Set(ctx, collections.Join[[]byte, uint64](valAddr, 0), customtypes.NewValidatorHistoricalRewards(customtypes.DecPools{}, 1))
	if err != nil {
		return err
	}

	// set current rewards (starting at period 1)
	err = k.ValidatorCurrentRewards.Set(ctx, valAddr, customtypes.NewValidatorCurrentRewards(customtypes.DecPools{}, 1))
	if err != nil {
		return err
	}

	// set accumulated commission
	err = k.ValidatorAccumulatedCommissions.Set(ctx, valAddr, customtypes.InitialValidatorAccumulatedCommission())
	if err != nil {
		return err
	}

	// set outstanding rewards
	err = k.ValidatorOutstandingRewards.Set(ctx, valAddr, customtypes.ValidatorOutstandingRewards{Rewards: customtypes.DecPools{}})
	if err != nil {
		return err
	}

	return nil
}

// IncrementValidatorPeriod increments validator period, returning the period just ended
func (k Keeper) IncrementValidatorPeriod(ctx context.Context, val stakingtypes.ValidatorI) (uint64, error) {
	valAddr, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(val.GetOperator())
	if err != nil {
		return 0, err
	}

	// fetch current rewards
	rewards, err := k.GetValidatorCurrentRewards(ctx, valAddr)
	if err != nil {
		return 0, err
	}

	// calculate current ratios
	var current customtypes.DecPools

	tokens := val.GetTokens()
	communityFunding := customtypes.DecPools{}
	for _, token := range tokens {
		rewardCoins := rewards.Rewards.CoinsOf(token.Denom)
		if token.IsZero() {
			// can't calculate ratio for zero-token validators
			// ergo we instead add to the community pool
			communityFunding = communityFunding.Add(customtypes.NewDecPool(token.Denom, rewardCoins))
		} else {
			current = current.Add(customtypes.NewDecPool(token.Denom, rewardCoins.QuoDecTruncate(math.LegacyNewDecFromInt(token.Amount))))
		}
	}

	feePool, err := k.FeePool.Get(ctx)
	if err != nil {
		return 0, err
	}

	outstanding, err := k.ValidatorOutstandingRewards.Get(ctx, valAddr)
	if err != nil {
		return 0, err
	}

	feePool.CommunityPool = feePool.CommunityPool.Add(communityFunding.Sum()...)
	outstanding.Rewards = outstanding.Rewards.Sub(communityFunding)
	err = k.FeePool.Set(ctx, feePool)
	if err != nil {
		return 0, err
	}

	err = k.ValidatorOutstandingRewards.Set(ctx, valAddr, outstanding)
	if err != nil {
		return 0, err
	}

	// fetch historical rewards for last period
	historicalRewards, err := k.ValidatorHistoricalRewards.Get(ctx, collections.Join[[]byte, uint64](valAddr, rewards.Period-1))
	if err != nil {
		return 0, err
	}

	// decrement reference count
	err = k.decrementReferenceCount(ctx, valAddr, rewards.Period-1)
	if err != nil {
		return 0, err
	}

	// set new historical rewards with reference count of 1
	err = k.ValidatorHistoricalRewards.Set(ctx, collections.Join[[]byte, uint64](valAddr, rewards.Period), customtypes.NewValidatorHistoricalRewards(historicalRewards.CumulativeRewardRatios.Add(current...), 1))
	if err != nil {
		return 0, err
	}

	// set current rewards, incrementing period by 1
	err = k.ValidatorCurrentRewards.Set(ctx, valAddr, customtypes.NewValidatorCurrentRewards(customtypes.DecPools{}, rewards.Period+1))
	if err != nil {
		return 0, err
	}

	return rewards.Period, nil
}

// increment the reference count for a historical rewards value
func (k Keeper) incrementReferenceCount(ctx context.Context, valAddr sdk.ValAddress, period uint64) error {
	historical, err := k.ValidatorHistoricalRewards.Get(ctx, collections.Join[[]byte, uint64](valAddr, period))
	if err != nil {
		return err
	}

	if historical.ReferenceCount > 2 {
		return errors.New("reference count should never exceed 2")
	}

	historical.ReferenceCount++
	return k.ValidatorHistoricalRewards.Set(ctx, collections.Join[[]byte, uint64](valAddr, period), historical)
}

// decrement the reference count for a historical rewards value, and delete if zero references remain
func (k Keeper) decrementReferenceCount(ctx context.Context, valAddr sdk.ValAddress, period uint64) error {
	historical, err := k.ValidatorHistoricalRewards.Get(ctx, collections.Join[[]byte, uint64](valAddr, period))
	if err != nil {
		return err
	}
	if historical.ReferenceCount == 0 {
		return errors.New("cannot set negative reference count")
	}

	historical.ReferenceCount--
	if historical.ReferenceCount == 0 {
		return k.ValidatorHistoricalRewards.Remove(ctx, collections.Join[[]byte, uint64](valAddr, period))
	} else {
		return k.ValidatorHistoricalRewards.Set(ctx, collections.Join[[]byte, uint64](valAddr, period), historical)
	}
}

func (k Keeper) updateValidatorSlashFraction(ctx context.Context, valAddr sdk.ValAddress, fractions sdk.DecCoins) error {
	for _, fraction := range fractions {
		if fraction.Amount.GT(math.LegacyOneDec()) || fraction.Amount.IsNegative() {
			return fmt.Errorf("fraction must be >=0 and <=1, current fraction: %v", fraction)
		}
	}

	val, err := k.stakingKeeper.Validator(ctx, valAddr)
	if err != nil {
		return err
	}

	// increment current period
	newPeriod, err := k.IncrementValidatorPeriod(ctx, val)
	if err != nil {
		return err
	}

	// increment reference count on period we need to track
	err = k.incrementReferenceCount(ctx, valAddr, newPeriod)
	if err != nil {
		return err
	}

	slashEvent := customtypes.NewValidatorSlashEvent(newPeriod, fractions)

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	height := uint64(sdkCtx.BlockHeight())

	err = k.ValidatorSlashEvents.Set(ctx, collections.Join3[[]byte, uint64, uint64](valAddr, height, newPeriod), slashEvent)
	if err != nil {
		return err
	}

	return nil
}

// GetValidatorHistoricalReferenceCount returns historical reference count (used for testcases)
func (k Keeper) GetValidatorHistoricalReferenceCount(ctx context.Context) (count uint64, err error) {
	err = k.ValidatorHistoricalRewards.Walk(ctx, nil, func(key collections.Pair[[]byte, uint64], rewards customtypes.ValidatorHistoricalRewards) (stop bool, err error) {
		count += uint64(rewards.ReferenceCount)
		return false, nil
	})

	return
}
