package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"

	customtypes "github.com/initia-labs/initia/x/distribution/types"
	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"
)

// initialize starting info for a new delegation
func (k Keeper) initializeDelegation(ctx context.Context, val sdk.ValAddress, del sdk.AccAddress) error {
	currentRewards, err := k.GetValidatorCurrentRewards(ctx, val)
	if err != nil {
		return err
	}

	// period has already been incremented - we want to store the period ended by this delegation action
	previousPeriod := currentRewards.Period - 1

	// increment reference count for the period we're going to track
	err = k.incrementReferenceCount(ctx, val, previousPeriod)
	if err != nil {
		return err
	}

	validator, err := k.stakingKeeper.Validator(ctx, val)
	if err != nil {
		return err
	}

	delegation, err := k.stakingKeeper.Delegation(ctx, del, val)
	if err != nil {
		return err
	}

	// calculate delegation stake in tokens
	// we don't store directly, so multiply delegation shares * (tokens per share)
	// note: necessary to truncate so we don't allow withdrawing more rewards than owed
	stake := validator.TokensFromSharesTruncated(delegation.GetShares())

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return k.DelegatorStartingInfos.Set(ctx, collections.Join[[]byte, []byte](val, del), customtypes.NewDelegatorStartingInfo(previousPeriod, stake, uint64(sdkCtx.BlockHeight())))
}

// calculate the rewards accrued by a delegation between two periods
func (k Keeper) calculateDelegationRewardsBetween(ctx context.Context, val stakingtypes.ValidatorI,
	startingPeriod, endingPeriod uint64, stakes sdk.DecCoins) (rewards customtypes.DecPools, err error) {
	// sanity check
	if startingPeriod > endingPeriod {
		panic("startingPeriod cannot be greater than endingPeriod")
	}

	// sanity check
	if stakes.IsAnyNegative() {
		panic("stake should not be negative")
	}

	valAddr, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(val.GetOperator())
	if err != nil {
		return nil, err
	}

	// return staking * (ending - starting)
	starting, err := k.ValidatorHistoricalRewards.Get(ctx, collections.Join(valAddr, startingPeriod))
	if err != nil {
		return nil, err
	}
	ending, err := k.ValidatorHistoricalRewards.Get(ctx, collections.Join(valAddr, endingPeriod))
	if err != nil {
		return nil, err
	}
	differences := ending.CumulativeRewardRatios.Sub(starting.CumulativeRewardRatios)
	if differences.IsAnyNegative() {
		panic("negative rewards should not be possible")
	}

	// note: necessary to truncate so we don't allow withdrawing more rewards than owed
	for _, diff := range differences {
		rewards = append(rewards, customtypes.NewDecPool(
			diff.Denom,
			diff.DecCoins.MulDecTruncate(stakes.AmountOf(diff.Denom)),
		))
	}

	return
}

// calculate the total rewards accrued by a delegation
func (k Keeper) CalculateDelegationRewards(ctx context.Context, val stakingtypes.ValidatorI, del stakingtypes.DelegationI, endingPeriod uint64) (rewards customtypes.DecPools, err error) {
	addrCodec := k.authKeeper.AddressCodec()
	delAddr, err := addrCodec.StringToBytes(del.GetDelegatorAddr())
	if err != nil {
		return nil, err
	}

	valAddr, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(del.GetValidatorAddr())
	if err != nil {
		return nil, err
	}

	// fetch starting info for delegation
	startingInfo, err := k.DelegatorStartingInfos.Get(ctx, collections.Join(valAddr, delAddr))
	if err != nil {
		return nil, err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if startingInfo.Height == uint64(sdkCtx.BlockHeight()) {
		// started this height, no rewards yet
		return
	}

	startingPeriod := startingInfo.PreviousPeriod
	stakes := startingInfo.Stakes

	// Iterate through slashes and withdraw with calculated staking for
	// distribution periods. These period offsets are dependent on *when* slashes
	// happen - namely, in BeginBlock, after rewards are allocated...
	// Slashes which happened in the first block would have been before this
	// delegation existed, UNLESS they were slashes of a redelegation to this
	// validator which was itself slashed (from a fault committed by the
	// redelegation source validator) earlier in the same BeginBlock.
	startingHeight := startingInfo.Height
	// Slashes this block happened after reward allocation, but we have to account
	// for them for the stake sanity check below.
	endingHeight := uint64(sdkCtx.BlockHeight())
	if endingHeight > startingHeight {
		err = k.ValidatorSlashEvents.Walk(ctx, new(collections.Range[collections.Triple[[]byte, uint64, uint64]]).
			StartInclusive(collections.Join3[[]byte, uint64, uint64](valAddr, startingHeight, 0)).
			EndExclusive(collections.Join3[[]byte, uint64, uint64](valAddr, endingHeight+1, 0)),
			func(key collections.Triple[[]byte, uint64, uint64], event customtypes.ValidatorSlashEvent) (stop bool, err error) {
				endingPeriod := event.ValidatorPeriod
				if endingPeriod > startingPeriod {
					rewardsBetween, err := k.calculateDelegationRewardsBetween(ctx, val, startingPeriod, endingPeriod, stakes)
					if err != nil {
						return false, err
					}

					rewards = rewards.Add(rewardsBetween...)

					// Note: It is necessary to truncate so we don't allow withdrawing
					// more rewards than owed.
					for i, stake := range stakes {
						stakes[i].Amount = stake.Amount.MulTruncate(math.LegacyOneDec().Sub(event.Fractions.AmountOf(stake.Denom)))
					}
					startingPeriod = endingPeriod
				}

				return false, nil
			},
		)
		if err != nil {
			return
		}
	}

	// A total stake sanity check; Recalculated final stake should be less than or
	// equal to current stake here. We cannot use Equals because stake is truncated
	// when multiplied by slash fractions (see above). We could only use equals if
	// we had arbitrary-precision rationals.
	currentStakes := val.TokensFromShares(del.GetShares())

	for i, stake := range stakes {
		currentStake := currentStakes.AmountOf(stake.Denom)
		if stake.Amount.GT(currentStake) {
			// AccountI for rounding inconsistencies between:
			//
			//     currentStake: calculated as in staking with a single computation
			//     stake:        calculated as an accumulation of stake
			//                   calculations across validator's distribution periods
			//
			// These inconsistencies are due to differing order of operations which
			// will inevitably have different accumulated rounding and may lead to
			// the smallest decimal place being one greater in stake than
			// currentStake. When we calculated slashing by period, even if we
			// round down for each slash fraction, it's possible due to how much is
			// being rounded that we slash less when slashing by period instead of
			// for when we slash without periods. In other words, the single slash,
			// and the slashing by period could both be rounding down but the
			// slashing by period is simply rounding down less, thus making stake >
			// currentStake
			//
			// A small amount of this error is tolerated and corrected for,
			// however any greater amount should be considered a breach in expected
			// behavior.
			marginOfErr := math.LegacySmallestDec().MulInt64(3)
			if stake.Amount.LTE(currentStake.Add(marginOfErr)) {
				stakes[i].Amount = currentStake
			} else {
				panic(fmt.Sprintf("calculated final stake for delegator %s greater than current stake"+
					"\n\tstake denom:\t%s"+
					"\n\tfinal stake:\t%s"+
					"\n\tcurrent stake:\t%s",
					del.GetDelegatorAddr(), stake.Denom, stake.Amount, currentStake))
			}
		}
	}

	// calculate rewards for final period
	rewardsBetween, err := k.calculateDelegationRewardsBetween(ctx, val, startingPeriod, endingPeriod, stakes)
	if err != nil {
		return nil, err
	}

	rewards = rewards.Add(rewardsBetween...)
	return rewards, nil
}

func (k Keeper) withdrawDelegationRewards(ctx context.Context, val stakingtypes.ValidatorI, del stakingtypes.DelegationI) (customtypes.Pools, error) {
	delAddr, err := k.authKeeper.AddressCodec().StringToBytes(del.GetDelegatorAddr())
	if err != nil {
		return nil, err
	}
	valAddr, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(del.GetValidatorAddr())
	if err != nil {
		return nil, err
	}

	// check existence of delegator starting info
	if ok, err := k.DelegatorStartingInfos.Has(ctx, collections.Join(valAddr, delAddr)); err != nil {
		return nil, err
	} else if !ok {
		return nil, types.ErrEmptyDelegationDistInfo
	}

	// end current period and calculate rewards
	endingPeriod, err := k.IncrementValidatorPeriod(ctx, val)
	if err != nil {
		return nil, err
	}

	rewardsRaw, err := k.CalculateDelegationRewards(ctx, val, del, endingPeriod)
	if err != nil {
		return nil, err
	}

	outstanding, err := k.GetValidatorOutstandingRewardsPools(ctx, valAddr)
	if err != nil {
		return nil, err
	}

	// defensive edge case may happen on the very final digits
	// of the decCoins due to operation order of the distribution mechanism.
	rewards := rewardsRaw.Intersect(outstanding)
	if !rewards.IsEqual(rewardsRaw) {
		logger := k.Logger(ctx)
		logger.Info(
			"rounding error withdrawing rewards from validator",
			"delegator", del.GetDelegatorAddr(),
			"validator", val.GetOperator(),
			"got", rewards.String(),
			"expected", rewardsRaw.String(),
		)
	}

	// truncate pools, return remainder to community pool
	pools, remainder := rewards.TruncateDecimal()
	coins := pools.Sum()

	// add pools to user account
	if !pools.IsEmpty() {
		withdrawAddr, err := k.GetDelegatorWithdrawAddr(ctx, delAddr)
		if err != nil {
			return nil, err
		}

		err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, withdrawAddr, coins)
		if err != nil {
			return nil, err
		}
	}

	// update the outstanding rewards and the community pool only if the
	// transaction was successful
	err = k.ValidatorOutstandingRewards.Set(ctx, valAddr, customtypes.ValidatorOutstandingRewards{Rewards: outstanding.Sub(rewards)})
	if err != nil {
		return nil, err
	}

	feePool, err := k.FeePool.Get(ctx)
	if err != nil {
		return nil, err
	}

	feePool.CommunityPool = feePool.CommunityPool.Add(remainder.Sum()...)
	err = k.FeePool.Set(ctx, feePool)
	if err != nil {
		return nil, err
	}

	// decrement reference count of starting period
	startingInfo, err := k.DelegatorStartingInfos.Get(ctx, collections.Join(valAddr, delAddr))
	if err != nil {
		return nil, err
	}

	startingPeriod := startingInfo.PreviousPeriod
	err = k.decrementReferenceCount(ctx, valAddr, startingPeriod)
	if err != nil {
		return nil, err
	}

	// remove delegator starting info
	err = k.DelegatorStartingInfos.Remove(ctx, collections.Join(valAddr, delAddr))
	if err != nil {
		return nil, err
	}

	return pools, nil
}
