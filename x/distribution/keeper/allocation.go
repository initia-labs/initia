package keeper

import (
	"context"

	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"

	customtypes "github.com/initia-labs/initia/x/distribution/types"
	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"
)

// beforeAllocateTokens swap fee tokens to base coin
func (k Keeper) beforeAllocateTokens(ctx context.Context) error {
	feeCollectorAddr := k.authKeeper.GetModuleAccount(ctx, k.feeCollectorName).GetAddress()
	feesCollected := k.bankKeeper.GetAllBalances(ctx, feeCollectorAddr)

	for _, coin := range feesCollected {
		if err := k.dexKeeper.SwapToBase(ctx, feeCollectorAddr, coin); err != nil {
			return err
		}
	}

	return nil
}

// AllocateTokens handles distribution of the collected fees
// bondedVotes is a list of (validator address, validator voted on last block flag) for all
// validators in the bonded set.
func (k Keeper) AllocateTokens(ctx context.Context, totalPreviousPower int64, bondedVotes []abci.VoteInfo) error {
	if err := k.beforeAllocateTokens(ctx); err != nil {
		return err
	}

	// fetch and clear the collected fees for distribution, since this is
	// called in BeginBlock, collected fees will be from the previous block
	// (and distributed to the previous proposer)
	feeCollector := k.authKeeper.GetModuleAccount(ctx, k.feeCollectorName)

	// Distribute only fees collected in the base denomination (e.g. INIT).
	// Other fee denominations have been swapped to base denom in beforeAllocateTokens.
	baseDenom, err := k.dexKeeper.BaseDenom(ctx)
	if err != nil {
		return err
	}
	feesCollectedInt := k.bankKeeper.GetBalance(ctx, feeCollector.GetAddress(), baseDenom)
	feesCollected := sdk.NewDecCoinsFromCoins(feesCollectedInt)

	// transfer collected fees to the distribution module account
	err = k.bankKeeper.SendCoinsFromModuleToModule(ctx, k.feeCollectorName, types.ModuleName, sdk.NewCoins(feesCollectedInt))
	if err != nil {
		return err
	}

	// temporary workaround to keep CanWithdrawInvariant happy
	// general discussions here: https://github.com/cosmos/cosmos-sdk/issues/2906#issuecomment-441867634
	feePool, err := k.FeePool.Get(ctx)
	if err != nil {
		return err
	}

	params, err := k.Params.Get(ctx)
	if err != nil {
		return err
	}

	if totalPreviousPower == 0 {
		feePool.CommunityPool = feePool.CommunityPool.Add(feesCollected...)
		return k.FeePool.Set(ctx, feePool)
	}

	// calculate fraction allocated to validators
	remaining := feesCollected
	communityTax := params.CommunityTax
	voteMultiplier := math.LegacyOneDec().Sub(communityTax)

	// map iteration not guarantee the ordering,
	// so we have to use array for iteration.
	rewardWeights, rewardWeightMap, weightsSum := k.LoadRewardWeights(ctx, params)
	validators, bondedTokens, bondedTokensSum, err := k.LoadBondedTokens(ctx, bondedVotes, rewardWeightMap)
	if err != nil {
		return err
	}

	// allocate rewards proportionally to reward power
	for _, rewardWeight := range rewardWeights {
		poolFraction := rewardWeight.Weight.QuoTruncate(weightsSum)
		poolReward := feesCollected.MulDecTruncate(voteMultiplier).MulDecTruncate(poolFraction)

		poolDenom := rewardWeight.Denom
		poolSize, ok := bondedTokensSum[poolDenom]

		// if poolSize is zero, skip allocation and then the poolReward will be allocated to community pool
		if !ok || poolSize.IsZero() {
			continue
		}

		for _, bondedTokens := range bondedTokens[poolDenom] {
			if bondedTokens.Amount.IsZero() {
				continue
			}

			validator := validators[bondedTokens.ValAddr]

			amountFraction := math.LegacyNewDecFromInt(bondedTokens.Amount).QuoInt(poolSize)
			reward := poolReward.MulDecTruncate(amountFraction)
			if reward.IsZero() {
				continue
			}

			err = k.AllocateTokensToValidatorPool(ctx, validator, poolDenom, reward)
			if err != nil {
				return err
			}

			remaining = remaining.Sub(reward)
		}
	}

	// allocate community funding
	feePool.CommunityPool = feePool.CommunityPool.Add(remaining...)
	return k.FeePool.Set(ctx, feePool)
}

// LoadRewardWeights load reward weights with its sum
func (k Keeper) LoadRewardWeights(ctx context.Context, params customtypes.Params) (
	[]customtypes.RewardWeight, map[string]math.LegacyDec, math.LegacyDec,
) {
	rewardWeights := params.RewardWeights

	weightsSum := math.LegacyZeroDec()
	weightsMap := make(map[string]math.LegacyDec, len(rewardWeights))

	for _, rewardWeight := range rewardWeights {
		weightsSum = weightsSum.Add(rewardWeight.Weight)
		weightsMap[rewardWeight.Denom] = rewardWeight.Weight
	}

	return rewardWeights, weightsMap, weightsSum
}

type validatorBondedToken struct {
	ValAddr string
	Amount  math.Int
}

// LoadBondedTokens build denom:(validator:amount) map
func (k Keeper) LoadBondedTokens(ctx context.Context, bondedVotes []abci.VoteInfo, rewardWeights map[string]math.LegacyDec) (
	map[string]stakingtypes.ValidatorI, map[string][]validatorBondedToken, map[string]math.Int, error,
) {
	numOfValidators := len(bondedVotes)
	numOfDenoms := len(rewardWeights)

	validators := make(map[string]stakingtypes.ValidatorI, numOfValidators)
	bondedTokens := make(map[string][]validatorBondedToken, numOfDenoms)
	bondedTokensSum := make(map[string]math.Int, numOfDenoms)

	for _, vote := range bondedVotes {
		validator, err := k.stakingKeeper.ValidatorByConsAddr(ctx, vote.Validator.Address)
		if err != nil {
			return nil, nil, nil, err
		}

		valAddr := validator.GetOperator()
		validators[valAddr] = validator

		// we don't need to check bonded status, so use val.GetTokens()
		for _, token := range validator.GetTokens() {
			// skip ops; denom != reward denom
			if _, found := rewardWeights[token.Denom]; !found {
				continue
			}

			if _, found := bondedTokens[token.Denom]; !found {
				bondedTokens[token.Denom] = make([]validatorBondedToken, 0, numOfValidators)
			}
			if _, found := bondedTokensSum[token.Denom]; !found {
				bondedTokensSum[token.Denom] = math.ZeroInt()
			}

			bondedTokens[token.Denom] = append(bondedTokens[token.Denom], validatorBondedToken{
				ValAddr: valAddr,
				Amount:  token.Amount,
			})
			bondedTokensSum[token.Denom] = bondedTokensSum[token.Denom].Add(token.Amount)
		}
	}

	return validators, bondedTokens, bondedTokensSum, nil
}

// AllocateTokensToValidatorPool allocate tokens to a particular validator's a particular pool, splitting according to commission
func (k Keeper) AllocateTokensToValidatorPool(ctx context.Context, val stakingtypes.ValidatorI, denom string, tokens sdk.DecCoins) error {
	valAddrStr := val.GetOperator()
	valAddr, err := k.stakingKeeper.ValidatorAddressCodec().StringToBytes(valAddrStr)
	if err != nil {
		return err
	}

	// split tokens between validator and delegators according to commission
	commissions := tokens.MulDec(val.GetCommission())
	shared := tokens.Sub(commissions)

	// update current commission
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeCommission,
			sdk.NewAttribute(types.AttributeKeyValidator, valAddrStr),
			sdk.NewAttribute(customtypes.AttributeKeyPool, denom),
			sdk.NewAttribute(sdk.AttributeKeyAmount, commissions.String()),
		),
	)

	// validator was updated at EndBlock of mstaking module,
	// so we can think this is the previous block state.
	currentCommission, err := k.GetValidatorAccumulatedCommission(ctx, valAddr)
	if err != nil {
		return err
	}
	currentRewards, err := k.GetValidatorCurrentRewards(ctx, valAddr)
	if err != nil {
		return err
	}
	outstanding, err := k.GetValidatorOutstandingRewards(ctx, valAddr)
	if err != nil {
		return err
	}

	currentCommission.Commissions = currentCommission.Commissions.Add(customtypes.NewDecPool(denom, commissions))
	currentRewards.Rewards = currentRewards.Rewards.Add(customtypes.NewDecPool(denom, shared))
	outstanding.Rewards = outstanding.Rewards.Add(customtypes.NewDecPool(denom, tokens))

	// update commission, current rewards, and outstanding rewards
	err = k.ValidatorAccumulatedCommissions.Set(ctx, valAddr, currentCommission)
	if err != nil {
		return err
	}
	err = k.ValidatorCurrentRewards.Set(ctx, valAddr, currentRewards)
	if err != nil {
		return err
	}
	err = k.ValidatorOutstandingRewards.Set(ctx, valAddr, outstanding)
	if err != nil {
		return err
	}

	// update outstanding rewards
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeRewards,
			sdk.NewAttribute(types.AttributeKeyValidator, valAddrStr),
			sdk.NewAttribute(customtypes.AttributeKeyPool, denom),
			sdk.NewAttribute(sdk.AttributeKeyAmount, tokens.String()),
		),
	)

	return nil
}
