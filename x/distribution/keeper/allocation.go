package keeper

import (
	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"

	customtypes "github.com/initia-labs/initia/x/distribution/types"
	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"
)

// beforeAllocateTokens swap fee tokens to base coin
func (k Keeper) beforeAllocateTokens(ctx sdk.Context) error {
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
func (k Keeper) AllocateTokens(ctx sdk.Context, totalPreviousPower int64, bondedVotes []abci.VoteInfo) {
	k.beforeAllocateTokens(ctx)

	// fetch and clear the collected fees for distribution, since this is
	// called in BeginBlock, collected fees will be from the previous block
	// (and distributed to the previous proposer)
	feeCollector := k.authKeeper.GetModuleAccount(ctx, k.feeCollectorName)
	feesCollectedInt := k.bankKeeper.GetAllBalances(ctx, feeCollector.GetAddress())
	feesCollected := sdk.NewDecCoinsFromCoins(feesCollectedInt...)

	// transfer collected fees to the distribution module account
	err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, k.feeCollectorName, types.ModuleName, feesCollectedInt)
	if err != nil {
		panic(err)
	}

	// temporary workaround to keep CanWithdrawInvariant happy
	// general discussions here: https://github.com/cosmos/cosmos-sdk/issues/2906#issuecomment-441867634
	feePool := k.GetFeePool(ctx)
	if totalPreviousPower == 0 {
		feePool.CommunityPool = feePool.CommunityPool.Add(feesCollected...)
		k.SetFeePool(ctx, feePool)
		return
	}

	// calculate fraction allocated to validators
	remaining := feesCollected
	communityTax := k.GetCommunityTax(ctx)
	voteMultiplier := sdk.OneDec().Sub(communityTax)

	rewardWeights, weightsSum := k.LoadRewardWeights(ctx)
	validators, bondedTokens, bondedTokensSum := k.LoadBondedTokens(ctx, bondedVotes, rewardWeights)

	// allocate rewards proportionally to reward power
	for rewardDenom, rewardWeight := range rewardWeights {
		poolFraction := rewardWeight.Quo(weightsSum)
		poolReward := feesCollected.MulDecTruncate(voteMultiplier).MulDecTruncate(poolFraction)

		poolDenom := rewardDenom
		poolSize := bondedTokensSum[poolDenom]
		for valAddr, amount := range bondedTokens[poolDenom] {
			validator := validators[valAddr]

			amountFraction := math.LegacyNewDecFromInt(amount).QuoInt(poolSize)
			reward := poolReward.MulDecTruncate(amountFraction)

			k.AllocateTokensToValidatorPool(ctx, validator, poolDenom, reward)
			remaining = remaining.Sub(reward)
		}
	}

	// allocate community funding
	feePool.CommunityPool = feePool.CommunityPool.Add(remaining...)
	k.SetFeePool(ctx, feePool)
}

// LoadRewardWeights load reward weights with its sum
func (k Keeper) LoadRewardWeights(ctx sdk.Context) (
	map[string]sdk.Dec, sdk.Dec,
) {
	rewardWeights := k.GetRewardWeights(ctx)

	weightsSum := math.LegacyZeroDec()
	weightsMap := make(map[string]sdk.Dec, len(rewardWeights))

	for _, rewardWeight := range rewardWeights {
		weightsSum = weightsSum.Add(rewardWeight.Weight)
		weightsMap[rewardWeight.Denom] = rewardWeight.Weight
	}

	return weightsMap, weightsSum
}

// LoadBondedTokens build denom:(validator:amount) map
func (k Keeper) LoadBondedTokens(ctx sdk.Context, bondedVotes []abci.VoteInfo, rewardWeights map[string]sdk.Dec) (
	map[string]stakingtypes.ValidatorI, map[string]map[string]math.Int, map[string]math.Int,
) {
	numOfValidators := len(bondedVotes)
	numOfDenoms := len(rewardWeights)

	validators := make(map[string]stakingtypes.ValidatorI, numOfValidators)
	bondedTokens := make(map[string]map[string]math.Int, numOfDenoms)
	bondedTokensSum := make(map[string]math.Int, numOfDenoms)

	for _, vote := range bondedVotes {
		valAddr := string(vote.Validator.Address)
		validator := k.stakingKeeper.ValidatorByConsAddr(ctx, vote.Validator.Address)
		validators[valAddr] = validator

		// we don't need to check bonded status, so use val.GetTokens()
		for _, token := range validator.GetTokens() {
			// skip ops; denom != reward denom
			if _, found := rewardWeights[token.Denom]; !found {
				continue
			}

			if _, found := bondedTokens[token.Denom]; !found {
				bondedTokens[token.Denom] = make(map[string]math.Int, numOfValidators)
			}
			if _, found := bondedTokensSum[token.Denom]; !found {
				bondedTokensSum[token.Denom] = sdk.ZeroInt()
			}

			bondedTokens[token.Denom][valAddr] = token.Amount
			bondedTokensSum[token.Denom] = bondedTokensSum[token.Denom].Add(token.Amount)
		}
	}

	return validators, bondedTokens, bondedTokensSum
}

// AllocateTokensToValidatorPool allocate tokens to a particular validator's a particular pool, splitting according to commission
func (k Keeper) AllocateTokensToValidatorPool(ctx sdk.Context, val stakingtypes.ValidatorI, denom string, tokens sdk.DecCoins) {
	valAddr := val.GetOperator().String()
	// split tokens between validator and delegators according to commission
	commissions := tokens.MulDec(val.GetCommission())
	shared := tokens.Sub(commissions)

	// update current commission
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeCommission,
			sdk.NewAttribute(types.AttributeKeyValidator, valAddr),
			sdk.NewAttribute(customtypes.AttributeKeyPool, denom),
			sdk.NewAttribute(sdk.AttributeKeyAmount, commissions.String()),
		),
	)

	// validator was updated at EndBlock of mstaking module,
	// so we can think this is the previous block state.
	currentCommission := k.GetValidatorAccumulatedCommission(ctx, val.GetOperator())
	currentRewards := k.GetValidatorCurrentRewards(ctx, val.GetOperator())
	outstanding := k.GetValidatorOutstandingRewards(ctx, val.GetOperator())

	currentCommission.Commissions = currentCommission.Commissions.Add(customtypes.NewDecPool(denom, commissions))
	currentRewards.Rewards = currentRewards.Rewards.Add(customtypes.NewDecPool(denom, shared))
	outstanding.Rewards = outstanding.Rewards.Add(customtypes.NewDecPool(denom, tokens))

	// update commission, current rewards, and outstanding rewards
	k.SetValidatorAccumulatedCommission(ctx, val.GetOperator(), currentCommission)
	k.SetValidatorCurrentRewards(ctx, val.GetOperator(), currentRewards)
	k.SetValidatorOutstandingRewards(ctx, val.GetOperator(), outstanding)

	// update outstanding rewards
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeRewards,
			sdk.NewAttribute(types.AttributeKeyValidator, valAddr),
			sdk.NewAttribute(customtypes.AttributeKeyPool, denom),
			sdk.NewAttribute(sdk.AttributeKeyAmount, tokens.String()),
		),
	)
}
