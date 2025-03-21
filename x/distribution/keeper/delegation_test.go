package keeper_test

import (
	"testing"

	"cosmossdk.io/math"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/proto/tendermint/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	customtypes "github.com/initia-labs/initia/x/distribution/types"
	staking "github.com/initia-labs/initia/x/mstaking"
	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"
	"github.com/stretchr/testify/require"
)

func TestCalculateRewardsBasic(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	input.StakingKeeper.SetBondDenoms(ctx, []string{"foo", "bar", "aaa"})
	setRewardWeights(t, ctx, input, []customtypes.RewardWeight{
		{
			Denom:  "foo",
			Weight: math.LegacyNewDecWithPrec(4, 1),
		},
		{
			Denom:  "bar",
			Weight: math.LegacyNewDecWithPrec(6, 1),
		},
		{
			Denom:  "aaa",
			Weight: math.LegacyNewDecWithPrec(1, 0),
		},
	})
	input.VotingPowerKeeper.SetVotingPowerWeights(sdk.NewDecCoins(sdk.NewDecCoin("foo", math.NewInt(1)), sdk.NewDecCoin("bar", math.NewInt(4)), sdk.NewDecCoin("aaa", math.NewInt(10))))

	valAddr1 := createValidatorWithCoin(ctx, input,
		sdk.NewCoins(sdk.NewInt64Coin("foo", 100_000_000), sdk.NewInt64Coin("bar", 100_000_000), sdk.NewInt64Coin("aaa", 100_000_000)),
		sdk.NewCoins(sdk.NewInt64Coin("foo", 2_000_000), sdk.NewInt64Coin("bar", 1_000_000), sdk.NewInt64Coin("aaa", 40_000)), 1)
	valAddr2 := createValidatorWithCoin(ctx, input,
		sdk.NewCoins(sdk.NewInt64Coin("foo", 100_000_000), sdk.NewInt64Coin("bar", 100_000_000), sdk.NewInt64Coin("aaa", 100_000_000)),
		sdk.NewCoins(sdk.NewInt64Coin("foo", 2_000_000), sdk.NewInt64Coin("bar", 4_000_000), sdk.NewInt64Coin("aaa", 10_000)), 2)

	// historical count should be 4 (once for validator init and once for delegation init per validator)
	refCount, err := input.DistKeeper.GetValidatorHistoricalReferenceCount(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(4), refCount)

	// end block to bond validator and start new block
	_, err = staking.EndBlocker(ctx, input.StakingKeeper)
	require.NoError(t, err)
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// fetch validator and delegation
	val, err := input.StakingKeeper.Validator(ctx, valAddr1)
	require.NoError(t, err)
	val2, err := input.StakingKeeper.Validator(ctx, valAddr2)
	require.NoError(t, err)
	del, err := input.StakingKeeper.Delegation(ctx, sdk.AccAddress(valAddr1), valAddr1)
	require.NoError(t, err)

	valConsPk1, err := val.ConsPubKey()
	require.NoError(t, err)
	valConsPk2, err := val2.ConsPubKey()
	require.NoError(t, err)

	// end period
	endingPeriod, err := input.DistKeeper.IncrementValidatorPeriod(ctx, val)
	require.NoError(t, err)

	// historical count should be 4 still
	refCount, err = input.DistKeeper.GetValidatorHistoricalReferenceCount(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(4), refCount)

	// calculate delegation rewards
	rewards, err := input.DistKeeper.CalculateDelegationRewards(ctx, val, del, endingPeriod)
	require.NoError(t, err)

	// rewards should be zero
	require.True(t, rewards.Sum().IsZero())

	fees := sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(100)))
	feeCollector := input.AccountKeeper.GetModuleAccount(ctx, authtypes.FeeCollectorName)
	require.NotNil(t, feeCollector)
	input.Faucet.Fund(ctx, authtypes.NewModuleAddress(authtypes.FeeCollectorName), fees...)

	votes := []abci.VoteInfo{
		{
			Validator: abci.Validator{
				Address: valConsPk1.Address(),
				Power:   100,
			},
			BlockIdFlag: types.BlockIDFlagCommit,
		},
		{
			Validator: abci.Validator{
				Address: valConsPk2.Address(),
				Power:   400,
			},
			BlockIdFlag: types.BlockIDFlagCommit,
		},
	}
	err = input.DistKeeper.AllocateTokens(ctx, 500, votes)
	require.NoError(t, err)

	// end period
	endingPeriod, err = input.DistKeeper.IncrementValidatorPeriod(ctx, val)
	require.NoError(t, err)

	// calculate delegation rewards
	rewards, err = input.DistKeeper.CalculateDelegationRewards(ctx, val, del, endingPeriod)
	require.NoError(t, err)

	require.Equal(t,
		customtypes.DecPools{
			// 98 * (40_000 / 50_000) * (10 / 20) * (9 / 10) = 35.28
			{Denom: "aaa", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDecWithPrec(3528, 2)}}},
			// 98 * (1_000_000 / 5_000_000) * (6 / 20) * (9 / 10) = 5.292
			{Denom: "bar", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDecWithPrec(5292, 3)}}},
			// 98 * (2_000_000 / 4_000_000) * (4 / 20) * (9 / 10) = 8.82
			{Denom: "foo", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDecWithPrec(882, 2)}}},
		},
		rewards)

	val1Commission, err := input.DistKeeper.ValidatorAccumulatedCommissions.Get(ctx, valAddr1)
	require.NoError(t, err)

	require.Equal(t,
		customtypes.DecPools{
			// 98 * (40_000 / 50_000) * (10 / 20) * (1 / 10) = 3.92
			{Denom: "aaa", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDecWithPrec(392, 2)}}},
			// 98 * (1_000_000 / 5_000_000) * (6 / 20) * (1 / 10) = 0.588
			{Denom: "bar", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDecWithPrec(588, 3)}}},
			// 98 * (2_000_000 / 4_000_000) * (4 / 20) * (1 / 10) = 0.98
			{Denom: "foo", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDecWithPrec(98, 2)}}},
		},
		val1Commission.Commissions)
}

func TestCalculateRewardsAfterSlash(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	input.StakingKeeper.SetBondDenoms(ctx, []string{"foo", "bar", "aaa"})
	setRewardWeights(t, ctx, input, []customtypes.RewardWeight{
		{
			Denom:  "foo",
			Weight: math.LegacyNewDecWithPrec(4, 1),
		},
		{
			Denom:  "bar",
			Weight: math.LegacyNewDecWithPrec(6, 1),
		},
		{
			Denom:  "aaa",
			Weight: math.LegacyNewDecWithPrec(1, 0),
		},
	})

	input.VotingPowerKeeper.SetVotingPowerWeights(sdk.NewDecCoins(sdk.NewDecCoin("foo", math.NewInt(1)), sdk.NewDecCoin("bar", math.NewInt(4)), sdk.NewDecCoin("aaa", math.NewInt(10))))

	valAddr1 := createValidatorWithCoin(ctx, input,
		sdk.NewCoins(sdk.NewInt64Coin("foo", 100_000_000), sdk.NewInt64Coin("bar", 100_000_000), sdk.NewInt64Coin("aaa", 100_000_000)),
		sdk.NewCoins(sdk.NewInt64Coin("foo", 2_000_000), sdk.NewInt64Coin("bar", 1_000_000), sdk.NewInt64Coin("aaa", 40_000)), 1)
	valAddr2 := createValidatorWithCoin(ctx, input,
		sdk.NewCoins(sdk.NewInt64Coin("foo", 100_000_000), sdk.NewInt64Coin("bar", 100_000_000), sdk.NewInt64Coin("aaa", 100_000_000)),
		sdk.NewCoins(sdk.NewInt64Coin("foo", 2_000_000), sdk.NewInt64Coin("bar", 4_000_000), sdk.NewInt64Coin("aaa", 10_000)), 2)

	// end block to bond validator and start new block
	_, err := staking.EndBlocker(ctx, input.StakingKeeper)
	require.NoError(t, err)
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// fetch validator and delegation
	val, err := input.StakingKeeper.Validator(ctx, valAddr1)
	require.NoError(t, err)
	val2, err := input.StakingKeeper.Validator(ctx, valAddr2)
	require.NoError(t, err)
	del, err := input.StakingKeeper.Delegation(ctx, sdk.AccAddress(valAddr1), valAddr1)
	require.NoError(t, err)

	valConsPk1, err := val.ConsPubKey()
	require.NoError(t, err)
	valConsPk2, err := val2.ConsPubKey()
	require.NoError(t, err)

	// end period
	endingPeriod, err := input.DistKeeper.IncrementValidatorPeriod(ctx, val)
	require.NoError(t, err)
	// calculate delegation rewards
	rewards, err := input.DistKeeper.CalculateDelegationRewards(ctx, val, del, endingPeriod)
	require.NoError(t, err)
	// rewards should be zero
	require.True(t, rewards.Sum().IsZero())

	pubkey, err := val.ConsPubKey()
	require.NoError(t, err)

	// start out block height
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 3)

	// slash the validator by 50%
	slashedTokens, err := input.StakingKeeper.Slash(ctx, pubkey.Address().Bytes(), ctx.BlockHeight(), math.LegacyNewDecWithPrec(5, 1))
	require.NoError(t, err)
	require.True(t, slashedTokens.IsAllPositive(), "expected positive slashed tokens, got: %s", slashedTokens)

	// increase block height
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 3)

	// retrieve validator
	val, err = input.StakingKeeper.Validator(ctx, valAddr1)
	require.NoError(t, err)

	fees := sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(100)))
	feeCollector := input.AccountKeeper.GetModuleAccount(ctx, authtypes.FeeCollectorName)
	require.NotNil(t, feeCollector)
	input.Faucet.Fund(ctx, authtypes.NewModuleAddress(authtypes.FeeCollectorName), fees...)

	votes := []abci.VoteInfo{
		{
			Validator: abci.Validator{
				Address: valConsPk1.Address(),
				Power:   100,
			},
			BlockIdFlag: types.BlockIDFlagCommit,
		},
		{
			Validator: abci.Validator{
				Address: valConsPk2.Address(),
				Power:   400,
			},
			BlockIdFlag: types.BlockIDFlagCommit,
		},
	}
	err = input.DistKeeper.AllocateTokens(ctx, 500, votes)
	require.NoError(t, err)

	// end period
	endingPeriod, err = input.DistKeeper.IncrementValidatorPeriod(ctx, val)
	require.NoError(t, err)

	// calculate delegation rewards
	rewards, err = input.DistKeeper.CalculateDelegationRewards(ctx, val, del, endingPeriod)
	require.NoError(t, err)

	// load commission
	commission, err := input.DistKeeper.ValidatorAccumulatedCommissions.Get(ctx, valAddr1)
	require.NoError(t, err)

	require.Equal(t,
		// 98 * (20_000 / 30_000) * (10 / 20) * (9 / 10)
		// + 98 * (500_000 / 4_500_000) * (6 / 20) * (9 / 10)
		// + 98 * (1_000_000 / 3_000_000) * (4 / 20) * (9 / 10)
		// = 38.22
		int64(38),
		rewards.Sum().AmountOf(bondDenom).TruncateInt64())

	require.Equal(t,
		// 98 * (20_000 / 30_000) * (10 / 20) * (1 / 10)
		// + 98 * (500_000 / 4_500_000) * (6 / 20) * (1 / 10)
		// + 98 * (1_000_000 / 3_000_000) * (4 / 20) * (1 / 10)
		// = 4.246666666666666
		int64(4),
		commission.Commissions.Sum().AmountOf(bondDenom).TruncateInt64())
}

func TestCalculateRewardsAfterManySlashes(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	input.StakingKeeper.SetBondDenoms(ctx, []string{"foo", "bar", "aaa"})
	setRewardWeights(t, ctx, input, []customtypes.RewardWeight{
		{
			Denom:  "foo",
			Weight: math.LegacyNewDecWithPrec(4, 1),
		},
		{
			Denom:  "bar",
			Weight: math.LegacyNewDecWithPrec(6, 1),
		},
		{
			Denom:  "aaa",
			Weight: math.LegacyNewDecWithPrec(1, 0),
		},
	})

	input.VotingPowerKeeper.SetVotingPowerWeights(sdk.NewDecCoins(sdk.NewDecCoin("foo", math.NewInt(1)), sdk.NewDecCoin("bar", math.NewInt(4)), sdk.NewDecCoin("aaa", math.NewInt(10))))

	valAddr1 := createValidatorWithCoin(ctx, input,
		sdk.NewCoins(sdk.NewInt64Coin("foo", 100_000_000), sdk.NewInt64Coin("bar", 100_000_000), sdk.NewInt64Coin("aaa", 100_000_000)),
		sdk.NewCoins(sdk.NewInt64Coin("foo", 2_000_000), sdk.NewInt64Coin("bar", 1_000_000), sdk.NewInt64Coin("aaa", 40_000)), 1)
	valAddr2 := createValidatorWithCoin(ctx, input,
		sdk.NewCoins(sdk.NewInt64Coin("foo", 100_000_000), sdk.NewInt64Coin("bar", 100_000_000), sdk.NewInt64Coin("aaa", 100_000_000)),
		sdk.NewCoins(sdk.NewInt64Coin("foo", 2_000_000), sdk.NewInt64Coin("bar", 4_000_000), sdk.NewInt64Coin("aaa", 10_000)), 2)

	validator1, err := input.StakingKeeper.Validator(ctx, valAddr1)
	require.NoError(t, err)

	pubkey, err := validator1.ConsPubKey()
	require.NoError(t, err)
	valConsAddr1 := pubkey.Address().Bytes()

	// end block to bond validator
	staking.EndBlocker(ctx, input.StakingKeeper)

	// next block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// fetch validator and delegation
	val, err := input.StakingKeeper.Validator(ctx, valAddr1)
	require.NoError(t, err)
	val2, err := input.StakingKeeper.Validator(ctx, valAddr2)
	require.NoError(t, err)
	del, err := input.StakingKeeper.Delegation(ctx, sdk.AccAddress(valAddr1), valAddr1)
	require.NoError(t, err)

	valConsPk1, err := val.ConsPubKey()
	require.NoError(t, err)
	valConsPk2, err := val2.ConsPubKey()
	require.NoError(t, err)

	// end period
	endingPeriod, err := input.DistKeeper.IncrementValidatorPeriod(ctx, val)
	require.NoError(t, err)

	// calculate delegation rewards
	rewards, err := input.DistKeeper.CalculateDelegationRewards(ctx, val, del, endingPeriod)
	require.NoError(t, err)

	// rewards should be zero
	require.True(t, rewards.Sum().IsZero())

	// start out block height
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 3)

	// slash the validator by 50%
	slashedTokens, err := input.StakingKeeper.Slash(ctx, valConsAddr1, ctx.BlockHeight(), math.LegacyNewDecWithPrec(5, 1))
	require.NoError(t, err)
	require.True(t, slashedTokens.IsAllPositive(), "expected positive slashed tokens, got: %s", slashedTokens)

	// fetch the validator again
	_, err = input.StakingKeeper.Validator(ctx, valAddr1)
	require.NoError(t, err)

	// increase block height
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 3)

	fees := sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(100)))
	feeCollector := input.AccountKeeper.GetModuleAccount(ctx, authtypes.FeeCollectorName)
	require.NotNil(t, feeCollector)
	input.Faucet.Fund(ctx, authtypes.NewModuleAddress(authtypes.FeeCollectorName), fees...)

	votes := []abci.VoteInfo{
		{
			Validator: abci.Validator{
				Address: valConsPk1.Address(),
				Power:   100,
			},
			BlockIdFlag: types.BlockIDFlagCommit,
		},
		{
			Validator: abci.Validator{
				Address: valConsPk2.Address(),
				Power:   400,
			},
			BlockIdFlag: types.BlockIDFlagCommit,
		},
	}
	err = input.DistKeeper.AllocateTokens(ctx, 500, votes)
	require.NoError(t, err)

	// slash the validator by 50% again
	slashedTokens, err = input.StakingKeeper.Slash(ctx, valConsAddr1, ctx.BlockHeight(), math.LegacyNewDecWithPrec(5, 1))
	require.NoError(t, err)
	require.True(t, slashedTokens.IsAllPositive(), "expected positive slashed tokens, got: %s", slashedTokens)

	// fetch the validator again
	val, err = input.StakingKeeper.Validator(ctx, valAddr1)
	require.NoError(t, err)

	// increase block height
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 3)

	input.Faucet.Fund(ctx, authtypes.NewModuleAddress(authtypes.FeeCollectorName), fees...)
	err = input.DistKeeper.AllocateTokens(ctx, 500, votes)
	require.NoError(t, err)

	// end period
	endingPeriod, err = input.DistKeeper.IncrementValidatorPeriod(ctx, val)
	require.NoError(t, err)

	// calculate delegation rewards
	rewards, err = input.DistKeeper.CalculateDelegationRewards(ctx, val, del, endingPeriod)
	require.NoError(t, err)

	commission, err := input.DistKeeper.ValidatorAccumulatedCommissions.Get(ctx, valAddr1)
	require.NoError(t, err)

	require.Equal(t,
		// 98 * (20_000 / 30_000) * (10 / 20) * (9 / 10)
		// + 98 * (500_000 / 4_500_000) * (6 / 20) * (9 / 10)
		// + 98 * (1_000_000 / 3_000_000) * (4 / 20) * (9 / 10)
		// = 38.22

		// 98 * (10_000 / 20_000) * (10 / 20) * (9 / 10)
		// + 98 * (250_000 / 4_250_000) * (6 / 20) * (9 / 10)
		// + 98 * (500_000 / 2_500_000) * (4 / 20) * (9 / 10)
		// = 27.134470588235295
		int64(65),
		rewards.Sum().AmountOf(bondDenom).TruncateInt64())

	require.Equal(t,
		// 98 * (20_000 / 30_000) * (10 / 20) * (1 / 10)
		// + 98 * (500_000 / 4_500_000) * (6 / 20) * (1 / 10)
		// + 98 * (1_000_000 / 3_000_000) * (4 / 20) * (1 / 10)
		// = 4.246666666666666

		// 98 * (10_000 / 20_000) * (10 / 20) * (1 / 10)
		// + 98 * (250_000 / 4_250_000) * (6 / 20) * (1 / 10)
		// + 98 * (500_000 / 2_500_000) * (4 / 20) * (1 / 10)
		// = 3.0149411764705882
		int64(7),
		commission.Commissions.Sum().AmountOf(bondDenom).TruncateInt64())
}

func TestCalculateRewardsMultiDelegator(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	input.StakingKeeper.SetBondDenoms(ctx, []string{"foo", "bar", "aaa"})
	setRewardWeights(t, ctx, input, []customtypes.RewardWeight{
		{
			Denom:  "foo",
			Weight: math.LegacyNewDecWithPrec(4, 1),
		},
		{
			Denom:  "bar",
			Weight: math.LegacyNewDecWithPrec(6, 1),
		},
		{
			Denom:  "aaa",
			Weight: math.LegacyNewDecWithPrec(1, 0),
		},
	})

	input.VotingPowerKeeper.SetVotingPowerWeights(sdk.NewDecCoins(sdk.NewDecCoin("foo", math.NewInt(1)), sdk.NewDecCoin("bar", math.NewInt(4)), sdk.NewDecCoin("aaa", math.NewInt(10))))

	valAddr1 := createValidatorWithCoin(ctx, input,
		sdk.NewCoins(sdk.NewInt64Coin("foo", 100_000_000), sdk.NewInt64Coin("bar", 100_000_000), sdk.NewInt64Coin("aaa", 100_000_000)),
		sdk.NewCoins(sdk.NewInt64Coin("foo", 2_000_000), sdk.NewInt64Coin("bar", 1_000_000), sdk.NewInt64Coin("aaa", 40_000)), 1)
	valAddr2 := createValidatorWithCoin(ctx, input,
		sdk.NewCoins(sdk.NewInt64Coin("foo", 100_000_000), sdk.NewInt64Coin("bar", 100_000_000), sdk.NewInt64Coin("aaa", 100_000_000)),
		sdk.NewCoins(sdk.NewInt64Coin("foo", 2_000_000), sdk.NewInt64Coin("bar", 4_000_000), sdk.NewInt64Coin("aaa", 10_000)), 2)

	validator, err := input.StakingKeeper.GetValidator(ctx, valAddr1)
	require.NoError(t, err)
	del1, err := input.StakingKeeper.GetDelegation(ctx, sdk.AccAddress(valAddr1), valAddr1)
	require.NoError(t, err)

	// fetch validator and delegation
	val, err := input.StakingKeeper.Validator(ctx, valAddr1)
	require.NoError(t, err)
	val2, err := input.StakingKeeper.Validator(ctx, valAddr2)
	require.NoError(t, err)

	valConsPk1, err := val.ConsPubKey()
	require.NoError(t, err)
	valConsPk2, err := val2.ConsPubKey()
	require.NoError(t, err)

	fees := sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(100)))
	feeCollector := input.AccountKeeper.GetModuleAccount(ctx, authtypes.FeeCollectorName)
	require.NotNil(t, feeCollector)
	input.Faucet.Fund(ctx, authtypes.NewModuleAddress(authtypes.FeeCollectorName), fees...)

	votes := []abci.VoteInfo{
		{
			Validator: abci.Validator{
				Address: valConsPk1.Address(),
				Power:   100,
			},
			BlockIdFlag: types.BlockIDFlagCommit,
		},
		{
			Validator: abci.Validator{
				Address: valConsPk2.Address(),
				Power:   400,
			},
			BlockIdFlag: types.BlockIDFlagCommit,
		},
	}
	err = input.DistKeeper.AllocateTokens(ctx, 500, votes)
	require.NoError(t, err)

	// delegate to validator
	bondCoins := sdk.NewCoins(sdk.NewCoin("foo", math.NewInt(1_000_000)))
	delAddr := input.Faucet.NewFundedAccount(ctx, bondCoins...)

	shares, err := input.StakingKeeper.Delegate(ctx, delAddr, bondCoins, stakingtypes.Unbonded, validator, true)
	require.NoError(t, err)
	require.Equal(t, sdk.NewDecCoinsFromCoins(bondCoins...), shares)
	del2, err := input.StakingKeeper.GetDelegation(ctx, delAddr, valAddr1)
	require.NoError(t, err)

	// fetch validator and delegation
	val, err = input.StakingKeeper.Validator(ctx, valAddr1)
	require.NoError(t, err)

	// end block
	staking.EndBlocker(ctx, input.StakingKeeper)
	// next block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// allocate some more rewards
	input.Faucet.Fund(ctx, authtypes.NewModuleAddress(authtypes.FeeCollectorName), fees...)
	err = input.DistKeeper.AllocateTokens(ctx, 500, votes)
	require.NoError(t, err)

	// end period
	endingPeriod, err := input.DistKeeper.IncrementValidatorPeriod(ctx, val)
	require.NoError(t, err)

	// calculate delegation rewards for del1
	rewards, err := input.DistKeeper.CalculateDelegationRewards(ctx, val, del1, endingPeriod)
	require.NoError(t, err)

	require.Equal(t,
		// 98 * (40_000 / 50_000) * (10 / 20) * (9 / 10)
		// + 98 * (1_000_000 / 5_000_000) * (6 / 20) * (9 / 10)
		// + 98 * (2_000_000 / 4_000_000) * (4 / 20) * (9 / 10)
		// = 49.392

		// 98 * (40_000 / 50_000) * (10 / 20) * (9 / 10)
		// + 98 * (1_000_000 / 5_000_000) * (6 / 20) * (9 / 10)
		// + 98 * (2_000_000 / 5_000_000) * (4 / 20) * (9 / 10)
		// = 47.628

		int64(97),
		rewards.Sum().AmountOf(bondDenom).TruncateInt64())

	// calculate delegation rewards for del2
	rewards, err = input.DistKeeper.CalculateDelegationRewards(ctx, val, del2, endingPeriod)
	require.NoError(t, err)

	// load commission
	commission, err := input.DistKeeper.ValidatorAccumulatedCommissions.Get(ctx, valAddr1)
	require.NoError(t, err)

	require.Equal(t,
		// 98 * (1_000_000 / 5_000_000) * (4 / 20) * (9 / 10)
		// = 3.528
		int64(3),
		rewards.Sum().AmountOf(bondDenom).TruncateInt64())

	require.Equal(t,
		// 98 * (40_000 / 50_000) * (10 / 20) * (1 / 10)
		// + 98 * (1_000_000 / 5_000_000) * (6 / 20) * (1 / 10)
		// + 98 * (2_000_000 / 4_000_000) * (4 / 20) * (1 / 10)
		// = 5.488

		// 98 * (40_000 / 50_000) * (10 / 20) * (1 / 10)
		// + 98 * (1_000_000 / 5_000_000) * (6 / 20) * (1 / 10)
		// + 98 * (3_000_000 / 5_000_000) * (4 / 20) * (1 / 10)
		// = 5.684
		int64(11),
		commission.Commissions.Sum().AmountOf(bondDenom).TruncateInt64())
}

func TestWithdrawDelegationRewardsBasic(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	input.StakingKeeper.SetBondDenoms(ctx, []string{"foo", "bar", "aaa"})
	setRewardWeights(t, ctx, input, []customtypes.RewardWeight{
		{
			Denom:  "foo",
			Weight: math.LegacyNewDecWithPrec(4, 1),
		},
		{
			Denom:  "bar",
			Weight: math.LegacyNewDecWithPrec(6, 1),
		},
		{
			Denom:  "aaa",
			Weight: math.LegacyNewDecWithPrec(1, 0),
		},
	})

	input.VotingPowerKeeper.SetVotingPowerWeights(sdk.NewDecCoins(sdk.NewDecCoin("foo", math.NewInt(1)), sdk.NewDecCoin("bar", math.NewInt(4)), sdk.NewDecCoin("aaa", math.NewInt(10))))

	valAddr1 := createValidatorWithCoin(ctx, input,
		sdk.NewCoins(sdk.NewInt64Coin("foo", 100_000_000), sdk.NewInt64Coin("bar", 100_000_000), sdk.NewInt64Coin("aaa", 100_000_000)),
		sdk.NewCoins(sdk.NewInt64Coin("foo", 2_000_000), sdk.NewInt64Coin("bar", 1_000_000), sdk.NewInt64Coin("aaa", 40_000)), 1)
	valAddr2 := createValidatorWithCoin(ctx, input,
		sdk.NewCoins(sdk.NewInt64Coin("foo", 100_000_000), sdk.NewInt64Coin("bar", 100_000_000), sdk.NewInt64Coin("aaa", 100_000_000)),
		sdk.NewCoins(sdk.NewInt64Coin("foo", 2_000_000), sdk.NewInt64Coin("bar", 4_000_000), sdk.NewInt64Coin("aaa", 10_000)), 2)

	// end block to bond validator and start new block
	_, err := staking.EndBlocker(ctx, input.StakingKeeper)
	require.NoError(t, err)
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// fetch validator and delegation
	val, err := input.StakingKeeper.Validator(ctx, valAddr1)
	require.NoError(t, err)
	val2, err := input.StakingKeeper.Validator(ctx, valAddr2)
	require.NoError(t, err)

	valConsPk1, err := val.ConsPubKey()
	require.NoError(t, err)
	valConsPk2, err := val2.ConsPubKey()
	require.NoError(t, err)

	fees := sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(100)))
	feeCollector := input.AccountKeeper.GetModuleAccount(ctx, authtypes.FeeCollectorName)
	require.NotNil(t, feeCollector)
	input.Faucet.Fund(ctx, authtypes.NewModuleAddress(authtypes.FeeCollectorName), fees...)

	votes := []abci.VoteInfo{
		{
			Validator: abci.Validator{
				Address: valConsPk1.Address(),
				Power:   100,
			},
			BlockIdFlag: types.BlockIDFlagCommit,
		},
		{
			Validator: abci.Validator{
				Address: valConsPk2.Address(),
				Power:   400,
			},
			BlockIdFlag: types.BlockIDFlagCommit,
		},
	}
	err = input.DistKeeper.AllocateTokens(ctx, 500, votes)
	require.NoError(t, err)

	// historical count should be 4 (initial + latest: 2 for delegation per validator)
	refCount, err := input.DistKeeper.GetValidatorHistoricalReferenceCount(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(4), refCount)

	// withdraw rewards
	rewards, err := input.DistKeeper.WithdrawDelegationRewards(ctx, sdk.AccAddress(valAddr1), valAddr1)
	require.NoError(t, err)

	require.Equal(t,
		customtypes.Pools{
			// 98 * (40_000 / 50_000) * (10 / 20) * (9 / 10) = 35.28
			{Denom: "aaa", Coins: sdk.Coins{{Denom: bondDenom, Amount: math.NewInt(35)}}},
			// 98 * (1_000_000 / 5_000_000) * (6 / 20) * (9 / 10) = 5.292
			{Denom: "bar", Coins: sdk.Coins{{Denom: bondDenom, Amount: math.NewInt(5)}}},
			// 98 * (2_000_000 / 4_000_000) * (4 / 20) * (9 / 10) = 8.82
			{Denom: "foo", Coins: sdk.Coins{{Denom: bondDenom, Amount: math.NewInt(8)}}},
		},
		rewards,
	)

	// historical count should still be 4 (added one record, cleared one)
	refCount, err = input.DistKeeper.GetValidatorHistoricalReferenceCount(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(4), refCount)

	// assert correct balance
	require.Equal(t,
		rewards.Sum().AmountOf(bondDenom).Int64(),
		input.BankKeeper.GetBalance(ctx, sdk.AccAddress(valAddr1), bondDenom).Amount.Int64(),
	)

	// withdraw commission
	_, err = input.DistKeeper.WithdrawValidatorCommission(ctx, valAddr1)
	require.Nil(t, err)
}

func TestWithdrawDelegationZeroRewards(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	input.StakingKeeper.SetBondDenoms(ctx, []string{"foo", "bar", "aaa"})
	setRewardWeights(t, ctx, input, []customtypes.RewardWeight{
		{
			Denom:  "foo",
			Weight: math.LegacyNewDecWithPrec(4, 1),
		},
		{
			Denom:  "bar",
			Weight: math.LegacyNewDecWithPrec(6, 1),
		},
		{
			Denom:  "aaa",
			Weight: math.LegacyNewDecWithPrec(1, 0),
		},
	})

	input.VotingPowerKeeper.SetVotingPowerWeights(sdk.NewDecCoins(sdk.NewDecCoin("foo", math.NewInt(1)), sdk.NewDecCoin("bar", math.NewInt(4)), sdk.NewDecCoin("aaa", math.NewInt(10))))

	valAddr1 := createValidatorWithCoin(ctx, input,
		sdk.NewCoins(sdk.NewInt64Coin("foo", 100_000_000), sdk.NewInt64Coin("bar", 100_000_000), sdk.NewInt64Coin("aaa", 100_000_000)),
		sdk.NewCoins(sdk.NewInt64Coin("foo", 2_000_000), sdk.NewInt64Coin("bar", 1_000_000), sdk.NewInt64Coin("aaa", 40_000)), 1)
	valAddr2 := createValidatorWithCoin(ctx, input,
		sdk.NewCoins(sdk.NewInt64Coin("foo", 100_000_000), sdk.NewInt64Coin("bar", 100_000_000), sdk.NewInt64Coin("aaa", 100_000_000)),
		sdk.NewCoins(sdk.NewInt64Coin("foo", 2_000_000), sdk.NewInt64Coin("bar", 4_000_000), sdk.NewInt64Coin("aaa", 10_000)), 2)

	// fetch validator and delegation
	val, err := input.StakingKeeper.Validator(ctx, valAddr1)
	require.NoError(t, err)
	val2, err := input.StakingKeeper.Validator(ctx, valAddr2)
	require.NoError(t, err)

	valConsPk1, err := val.ConsPubKey()
	require.NoError(t, err)
	valConsPk2, err := val2.ConsPubKey()
	require.NoError(t, err)

	fees := sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(100)))
	feeCollector := input.AccountKeeper.GetModuleAccount(ctx, authtypes.FeeCollectorName)
	require.NotNil(t, feeCollector)
	input.Faucet.Fund(ctx, authtypes.NewModuleAddress(authtypes.FeeCollectorName), fees...)

	votes := []abci.VoteInfo{
		{
			Validator: abci.Validator{
				Address: valConsPk1.Address(),
				Power:   100,
			},
			BlockIdFlag: types.BlockIDFlagCommit,
		},
		{
			Validator: abci.Validator{
				Address: valConsPk2.Address(),
				Power:   400,
			},
			BlockIdFlag: types.BlockIDFlagCommit,
		},
	}
	err = input.DistKeeper.AllocateTokens(ctx, 500, votes)
	require.NoError(t, err)

	// withdraw rewards -- should be 0
	pool, err := input.DistKeeper.WithdrawDelegationRewards(ctx, sdk.AccAddress(valAddr1), valAddr1)
	require.NoError(t, err)
	require.True(t, pool.Sum().IsZero(), "expected withdraw rewards to be zero")
	require.True(t, pool.Sum().IsValid(), "expected returned coins to be valid")
}

func TestCalculateRewardsAfterManySlashesInSameBlock(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	input.StakingKeeper.SetBondDenoms(ctx, []string{"foo", "bar", "aaa"})
	setRewardWeights(t, ctx, input, []customtypes.RewardWeight{
		{
			Denom:  "foo",
			Weight: math.LegacyNewDecWithPrec(4, 1),
		},
		{
			Denom:  "bar",
			Weight: math.LegacyNewDecWithPrec(6, 1),
		},
		{
			Denom:  "aaa",
			Weight: math.LegacyNewDecWithPrec(1, 0),
		},
	})

	input.VotingPowerKeeper.SetVotingPowerWeights(sdk.NewDecCoins(sdk.NewDecCoin("foo", math.NewInt(1)), sdk.NewDecCoin("bar", math.NewInt(4)), sdk.NewDecCoin("aaa", math.NewInt(10))))

	valAddr1 := createValidatorWithCoin(ctx, input,
		sdk.NewCoins(sdk.NewInt64Coin("foo", 100_000_000), sdk.NewInt64Coin("bar", 100_000_000), sdk.NewInt64Coin("aaa", 100_000_000)),
		sdk.NewCoins(sdk.NewInt64Coin("foo", 2_000_000), sdk.NewInt64Coin("bar", 1_000_000), sdk.NewInt64Coin("aaa", 40_000)), 1)
	valAddr2 := createValidatorWithCoin(ctx, input,
		sdk.NewCoins(sdk.NewInt64Coin("foo", 100_000_000), sdk.NewInt64Coin("bar", 100_000_000), sdk.NewInt64Coin("aaa", 100_000_000)),
		sdk.NewCoins(sdk.NewInt64Coin("foo", 2_000_000), sdk.NewInt64Coin("bar", 4_000_000), sdk.NewInt64Coin("aaa", 10_000)), 2)

	// end block to bond validator and start new block
	_, err := staking.EndBlocker(ctx, input.StakingKeeper)
	require.NoError(t, err)
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// fetch validator and delegation
	val, err := input.StakingKeeper.Validator(ctx, valAddr1)
	require.NoError(t, err)
	val2, err := input.StakingKeeper.Validator(ctx, valAddr2)
	require.NoError(t, err)

	valConsPk1, err := val.ConsPubKey()
	require.NoError(t, err)
	valConsPk2, err := val2.ConsPubKey()
	require.NoError(t, err)

	del, err := input.StakingKeeper.Delegation(ctx, sdk.AccAddress(valAddr1), valAddr1)
	require.NoError(t, err)

	// end period
	endingPeriod, err := input.DistKeeper.IncrementValidatorPeriod(ctx, val)
	require.NoError(t, err)
	// calculate delegation rewards
	rewards, err := input.DistKeeper.CalculateDelegationRewards(ctx, val, del, endingPeriod)
	require.NoError(t, err)

	// rewards should be zero
	require.True(t, rewards.Sum().IsZero())

	fees := sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(100)))
	feeCollector := input.AccountKeeper.GetModuleAccount(ctx, authtypes.FeeCollectorName)
	require.NotNil(t, feeCollector)
	votes := []abci.VoteInfo{
		{
			Validator: abci.Validator{
				Address: valConsPk1.Address(),
				Power:   100,
			},
			BlockIdFlag: types.BlockIDFlagCommit,
		},
		{
			Validator: abci.Validator{
				Address: valConsPk2.Address(),
				Power:   400,
			},
			BlockIdFlag: types.BlockIDFlagCommit,
		},
	}
	input.Faucet.Fund(ctx, authtypes.NewModuleAddress(authtypes.FeeCollectorName), fees...)
	err = input.DistKeeper.AllocateTokens(ctx, 500, votes)
	require.NoError(t, err)

	pubkey, err := val.ConsPubKey()
	require.NoError(t, err)
	valConsAddr := pubkey.Address().Bytes()

	// start out block height
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 3)

	// slash the validator by 50%
	slashedTokens, err := input.StakingKeeper.Slash(ctx, valConsAddr, ctx.BlockHeight(), math.LegacyNewDecWithPrec(5, 1))
	require.NoError(t, err)
	require.True(t, slashedTokens.IsAllPositive(), "expected positive slashed tokens, got: %s", slashedTokens)

	// slash the validator by 50% again
	slashedTokens, err = input.StakingKeeper.Slash(ctx, valConsAddr, ctx.BlockHeight(), math.LegacyNewDecWithPrec(5, 1))
	require.NoError(t, err)
	require.True(t, slashedTokens.IsAllPositive(), "expected positive slashed tokens, got: %s", slashedTokens)

	// increase block height
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 3)

	input.Faucet.Fund(ctx, authtypes.NewModuleAddress(authtypes.FeeCollectorName), fees...)
	err = input.DistKeeper.AllocateTokens(ctx, 500, votes)
	require.NoError(t, err)

	// fetch the validator again
	val, err = input.StakingKeeper.Validator(ctx, valAddr1)
	require.NoError(t, err)

	// end period
	endingPeriod, err = input.DistKeeper.IncrementValidatorPeriod(ctx, val)
	require.NoError(t, err)

	// calculate delegation rewards
	rewards, err = input.DistKeeper.CalculateDelegationRewards(ctx, val, del, endingPeriod)
	require.NoError(t, err)

	// load commission
	commission, err := input.DistKeeper.ValidatorAccumulatedCommissions.Get(ctx, valAddr1)
	require.NoError(t, err)

	require.Equal(t,
		// 98 * (40_000 / 50_000) * (10 / 20) * (9 / 10)
		// + 98 * (1_000_000 / 5_000_000) * (6 / 20) * (9 / 10)
		// + 98 * (2_000_000 / 4_000_000) * (4 / 20) * (9 / 10)
		// = 49.392

		// 98 * (10_000 / 20_000) * (10 / 20) * (9 / 10)
		// + 98 * (250_000 / 4_250_000) * (6 / 20) * (9 / 10)
		// + 98 * (500_000 / 2_500_000) * (4 / 20) * (9 / 10)
		// = 27.134470588235295
		int64(76),
		rewards.Sum().AmountOf(bondDenom).TruncateInt64())

	require.Equal(t,
		// 98 * (40_000 / 50_000) * (10 / 20) * (1 / 10)
		// + 98 * (1_000_000 / 5_000_000) * (6 / 20) * (1 / 10)
		// + 98 * (2_000_000 / 4_000_000) * (4 / 20) * (1 / 10)
		// = 5.488

		// 98 * (10_000 / 20_000) * (10 / 20) * (1 / 10)
		// + 98 * (250_000 / 4_250_000) * (6 / 20) * (1 / 10)
		// + 98 * (500_000 / 2_500_000) * (4 / 20) * (1 / 10)
		// = 3.0149411764705882
		int64(8),
		commission.Commissions.Sum().AmountOf(bondDenom).TruncateInt64())
}

func TestCalculateRewardsMultiDelegatorMultiSlash(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	input.StakingKeeper.SetBondDenoms(ctx, []string{"foo", "bar", "aaa"})
	setRewardWeights(t, ctx, input, []customtypes.RewardWeight{
		{
			Denom:  "foo",
			Weight: math.LegacyNewDecWithPrec(4, 1),
		},
		{
			Denom:  "bar",
			Weight: math.LegacyNewDecWithPrec(6, 1),
		},
		{
			Denom:  "aaa",
			Weight: math.LegacyNewDecWithPrec(1, 0),
		},
	})

	input.VotingPowerKeeper.SetVotingPowerWeights(sdk.NewDecCoins(sdk.NewDecCoin("foo", math.NewInt(1)), sdk.NewDecCoin("bar", math.NewInt(4)), sdk.NewDecCoin("aaa", math.NewInt(10))))

	valAddr1 := createValidatorWithCoin(ctx, input,
		sdk.NewCoins(sdk.NewInt64Coin("foo", 100_000_000), sdk.NewInt64Coin("bar", 100_000_000), sdk.NewInt64Coin("aaa", 100_000_000)),
		sdk.NewCoins(sdk.NewInt64Coin("foo", 2_000_000), sdk.NewInt64Coin("bar", 1_000_000), sdk.NewInt64Coin("aaa", 40_000)), 1)
	valAddr2 := createValidatorWithCoin(ctx, input,
		sdk.NewCoins(sdk.NewInt64Coin("foo", 100_000_000), sdk.NewInt64Coin("bar", 100_000_000), sdk.NewInt64Coin("aaa", 100_000_000)),
		sdk.NewCoins(sdk.NewInt64Coin("foo", 2_000_000), sdk.NewInt64Coin("bar", 4_000_000), sdk.NewInt64Coin("aaa", 10_000)), 2)

	// end block to bond validator and start new block
	_, err := staking.EndBlocker(ctx, input.StakingKeeper)
	require.NoError(t, err)
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// fetch validator and delegation
	val, err := input.StakingKeeper.Validator(ctx, valAddr1)
	require.NoError(t, err)
	val2, err := input.StakingKeeper.Validator(ctx, valAddr2)
	require.NoError(t, err)
	del1, err := input.StakingKeeper.Delegation(ctx, sdk.AccAddress(valAddr1), valAddr1)
	require.NoError(t, err)

	valConsPk1, err := val.ConsPubKey()
	require.NoError(t, err)
	valConsPk2, err := val2.ConsPubKey()
	require.NoError(t, err)

	valConsAddr := valConsPk1.Address().Bytes()

	fees := sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(100)))
	feeCollector := input.AccountKeeper.GetModuleAccount(ctx, authtypes.FeeCollectorName)
	require.NotNil(t, feeCollector)
	votes := []abci.VoteInfo{
		{
			Validator: abci.Validator{
				Address: valConsPk1.Address(),
				Power:   100,
			},
			BlockIdFlag: types.BlockIDFlagCommit,
		},
		{
			Validator: abci.Validator{
				Address: valConsPk2.Address(),
				Power:   400,
			},
			BlockIdFlag: types.BlockIDFlagCommit,
		},
	}
	input.Faucet.Fund(ctx, authtypes.NewModuleAddress(authtypes.FeeCollectorName), fees...)
	err = input.DistKeeper.AllocateTokens(ctx, 500, votes)
	require.NoError(t, err)

	// slash the validator
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 3)
	slashedTokens, err := input.StakingKeeper.Slash(ctx, valConsAddr, ctx.BlockHeight(), math.LegacyNewDecWithPrec(5, 1))
	require.NoError(t, err)
	require.True(t, slashedTokens.IsAllPositive(), "expected positive slashed tokens, got: %s", slashedTokens)
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 3)

	// second delegation
	bondCoins := sdk.NewCoins(sdk.NewCoin("foo", math.NewInt(1_000_000)))
	delAddr := input.Faucet.NewFundedAccount(ctx, bondCoins...)

	validator, err := input.StakingKeeper.GetValidator(ctx, valAddr1)
	require.NoError(t, err)
	shares, err := input.StakingKeeper.Delegate(ctx, delAddr, bondCoins, stakingtypes.Unbonded, validator, true)
	require.NoError(t, err)

	// existing 2_000_000 shares for 2_000_000 / 2 tokens => new shares == 2_000_000
	require.Equal(t, sdk.NewDecCoinsFromCoins(bondCoins...).MulDec(math.LegacyNewDec(2)), shares)
	del2, err := input.StakingKeeper.GetDelegation(ctx, delAddr, valAddr1)
	require.NoError(t, err)

	// end block
	_, err = staking.EndBlocker(ctx, input.StakingKeeper)
	require.NoError(t, err)

	// next block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// allocate some more rewards
	input.Faucet.Fund(ctx, authtypes.NewModuleAddress(authtypes.FeeCollectorName), fees...)
	err = input.DistKeeper.AllocateTokens(ctx, 500, votes)
	require.NoError(t, err)

	// slash the validator again
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 3)
	slashedTokens, err = input.StakingKeeper.Slash(ctx, valConsAddr, ctx.BlockHeight(), math.LegacyNewDecWithPrec(5, 1))
	require.NoError(t, err)
	require.True(t, slashedTokens.IsAllPositive(), "expected positive slashed tokens, got: %s", slashedTokens)
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 3)

	// fetch updated validator
	val, err = input.StakingKeeper.Validator(ctx, valAddr1)
	require.NoError(t, err)

	// end period
	endingPeriod, err := input.DistKeeper.IncrementValidatorPeriod(ctx, val)
	require.NoError(t, err)

	// calculate delegation rewards for del1
	rewards, err := input.DistKeeper.CalculateDelegationRewards(ctx, val, del1, endingPeriod)
	require.NoError(t, err)

	require.Equal(t,
		// 98 * (40_000 / 50_000) * (10 / 20) * (9 / 10)
		// + 98 * (1_000_000 / 5_000_000) * (6 / 20) * (9 / 10)
		// + 98 * (2_000_000 / 4_000_000) * (4 / 20) * (9 / 10)
		// = 49.392

		// 98 * (20_000 / 30_000) * (10 / 20) * (9 / 10)
		// + 98 * (500_000 / 4_500_000) * (6 / 20) * (9 / 10)
		// + 98 * (1_000_000 / 4_000_000) * (4 / 20) * (9 / 10)
		// = 36.75
		int64(86),
		rewards.Sum().AmountOf(bondDenom).TruncateInt64())

	// calculate delegation rewards for del2
	rewards, err = input.DistKeeper.CalculateDelegationRewards(ctx, val, del2, endingPeriod)
	require.NoError(t, err)

	require.Equal(t,
		// 98 * (1_000_000 / 4_000_000) * (4 / 20) * (9 / 10)
		// = 4.41
		int64(4),
		rewards.Sum().AmountOf(bondDenom).TruncateInt64())

	// load commission
	commission, err := input.DistKeeper.ValidatorAccumulatedCommissions.Get(ctx, valAddr1)
	require.NoError(t, err)

	require.Equal(t,
		// 98 * (40_000 / 50_000) * (10 / 20) * (1 / 10)
		// + 98 * (1_000_000 / 5_000_000) * (6 / 20) * (1 / 10)
		// + 98 * (2_000_000 / 4_000_000) * (4 / 20) * (1 / 10)
		// = 5.488

		// 98 * (20_000 / 30_000) * (10 / 20) * (1 / 10)
		// + 98 * (500_000 / 4_500_000) * (6 / 20) * (1 / 10)
		// + 98 * (3_000_000 / 4_000_000) * (4 / 20) * (1 / 10)
		// = 5.06333333
		int64(10),
		commission.Commissions.Sum().AmountOf(bondDenom).TruncateInt64())
}

func TestCalculateRewardsMultiDelegatorMultWithdraw(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	input.StakingKeeper.SetBondDenoms(ctx, []string{"foo", "bar", "aaa"})
	setRewardWeights(t, ctx, input, []customtypes.RewardWeight{
		{
			Denom:  "foo",
			Weight: math.LegacyNewDecWithPrec(4, 1),
		},
		{
			Denom:  "bar",
			Weight: math.LegacyNewDecWithPrec(6, 1),
		},
		{
			Denom:  "aaa",
			Weight: math.LegacyNewDecWithPrec(1, 0),
		},
	})

	input.VotingPowerKeeper.SetVotingPowerWeights(sdk.NewDecCoins(sdk.NewDecCoin("foo", math.NewInt(1)), sdk.NewDecCoin("bar", math.NewInt(4)), sdk.NewDecCoin("aaa", math.NewInt(10))))

	valAddr1 := createValidatorWithCoin(ctx, input,
		sdk.NewCoins(sdk.NewInt64Coin("foo", 100_000_000), sdk.NewInt64Coin("bar", 100_000_000), sdk.NewInt64Coin("aaa", 100_000_000)),
		sdk.NewCoins(sdk.NewInt64Coin("foo", 2_000_000), sdk.NewInt64Coin("bar", 1_000_000), sdk.NewInt64Coin("aaa", 40_000)), 1)
	valAddr2 := createValidatorWithCoin(ctx, input,
		sdk.NewCoins(sdk.NewInt64Coin("foo", 100_000_000), sdk.NewInt64Coin("bar", 100_000_000), sdk.NewInt64Coin("aaa", 100_000_000)),
		sdk.NewCoins(sdk.NewInt64Coin("foo", 2_000_000), sdk.NewInt64Coin("bar", 4_000_000), sdk.NewInt64Coin("aaa", 10_000)), 2)

	// end block to bond validator and start new block
	_, err := staking.EndBlocker(ctx, input.StakingKeeper)
	require.NoError(t, err)
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// fetch validator and delegation
	val, err := input.StakingKeeper.Validator(ctx, valAddr1)
	require.NoError(t, err)
	val2, err := input.StakingKeeper.Validator(ctx, valAddr2)
	require.NoError(t, err)
	del1, err := input.StakingKeeper.Delegation(ctx, sdk.AccAddress(valAddr1), valAddr1)
	require.NoError(t, err)

	valConsPk1, err := val.ConsPubKey()
	require.NoError(t, err)
	valConsPk2, err := val2.ConsPubKey()
	require.NoError(t, err)

	fees := sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(100)))
	feeCollector := input.AccountKeeper.GetModuleAccount(ctx, authtypes.FeeCollectorName)
	require.NotNil(t, feeCollector)
	votes := []abci.VoteInfo{
		{
			Validator: abci.Validator{
				Address: valConsPk1.Address(),
				Power:   100,
			},
			BlockIdFlag: types.BlockIDFlagCommit,
		},
		{
			Validator: abci.Validator{
				Address: valConsPk2.Address(),
				Power:   400,
			},
			BlockIdFlag: types.BlockIDFlagCommit,
		},
	}

	// second delegation
	bondCoins := sdk.NewCoins(sdk.NewCoin("foo", math.NewInt(1_000_000)))
	delAddr := input.Faucet.NewFundedAccount(ctx, bondCoins...)

	validator, err := input.StakingKeeper.GetValidator(ctx, valAddr1)
	require.NoError(t, err)
	_, err = input.StakingKeeper.Delegate(ctx, delAddr, bondCoins, stakingtypes.Unbonded, validator, true)
	require.NoError(t, err)

	del2, err := input.StakingKeeper.GetDelegation(ctx, delAddr, valAddr1)
	require.NoError(t, err)

	// end block
	_, err = staking.EndBlocker(ctx, input.StakingKeeper)
	require.NoError(t, err)

	// next block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// allocate some more rewards
	input.Faucet.Fund(ctx, authtypes.NewModuleAddress(authtypes.FeeCollectorName), fees...)
	err = input.DistKeeper.AllocateTokens(ctx, 500, votes)
	require.NoError(t, err)

	// first delegator withdraws
	_, err = input.DistKeeper.WithdrawDelegationRewards(ctx, sdk.AccAddress(valAddr1), valAddr1)
	require.NoError(t, err)

	// second delegator withdraws
	_, err = input.DistKeeper.WithdrawDelegationRewards(ctx, delAddr, valAddr1)
	require.NoError(t, err)

	// validator withdraws commission (1)
	_, err = input.DistKeeper.WithdrawValidatorCommission(ctx, valAddr1)
	require.NoError(t, err)

	// end period
	endingPeriod, err := input.DistKeeper.IncrementValidatorPeriod(ctx, val)
	require.NoError(t, err)

	// calculate delegation rewards for del1
	rewards, err := input.DistKeeper.CalculateDelegationRewards(ctx, val, del1, endingPeriod)
	require.NoError(t, err)

	// rewards for del1 should be zero
	require.True(t, rewards.Sum().IsZero())

	// calculate delegation rewards for del2
	rewards, err = input.DistKeeper.CalculateDelegationRewards(ctx, val, del2, endingPeriod)
	require.NoError(t, err)

	// rewards for del2 should be zero
	require.True(t, rewards.Sum().IsZero())

	val1Commission, err := input.DistKeeper.ValidatorAccumulatedCommissions.Get(ctx, valAddr1)
	require.NoError(t, err)
	require.Equal(t,
		// 98 * (40_000 / 50_000) * (10 / 20) * (1 / 10) = 3.92
		// 98 * (1_000_000 / 5_000_000) * (6 / 20) * (1 / 10) = 0.588
		// 98 * (3_000_000 / 5_000_000) * (4 / 20) * (1 / 10) = 1.176

		// 0.92 + 0.588 + 0.176 = 1.684
		int64(1),
		val1Commission.Commissions.Sum().AmountOf(bondDenom).TruncateInt64())

	// next block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	input.Faucet.Fund(ctx, authtypes.NewModuleAddress(authtypes.FeeCollectorName), fees...)
	err = input.DistKeeper.AllocateTokens(ctx, 500, votes)
	require.NoError(t, err)

	// first delegator withdraws again
	_, err = input.DistKeeper.WithdrawDelegationRewards(ctx, sdk.AccAddress(valAddr1), valAddr1)
	require.NoError(t, err)

	// end period
	endingPeriod, err = input.DistKeeper.IncrementValidatorPeriod(ctx, val)
	require.NoError(t, err)

	// calculate delegation rewards for del2
	rewards, err = input.DistKeeper.CalculateDelegationRewards(ctx, val, del2, endingPeriod)
	require.NoError(t, err)

	require.Equal(t,
		// 98 * (1_000_000 / 5_000_000) * (4 / 20) * (9 / 10)
		// = 3.5280000000000005
		int64(3),
		rewards.Sum().AmountOf(bondDenom).TruncateInt64())

	val1Commission, err = input.DistKeeper.ValidatorAccumulatedCommissions.Get(ctx, valAddr1)
	require.NoError(t, err)
	require.Equal(t,
		// 98 * (40_000 / 50_000) * (10 / 20) * (1 / 10) + 0.92
		// + 98 * (1_000_000 / 5_000_000) * (6 / 20) * (1 / 10) + 0.588
		// + 98 * (3_000_000 / 5_000_000) * (4 / 20) * (1 / 10) + 0.176
		// 7.368

		int64(7),
		val1Commission.Commissions.Sum().AmountOf(bondDenom).TruncateInt64())

	// next block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	input.Faucet.Fund(ctx, authtypes.NewModuleAddress(authtypes.FeeCollectorName), fees...)
	err = input.DistKeeper.AllocateTokens(ctx, 500, votes)
	require.NoError(t, err)

	// withdraw commission (2)
	_, err = input.DistKeeper.WithdrawValidatorCommission(ctx, valAddr1)
	require.NoError(t, err)

	// next block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// fetch validator again
	val, err = input.StakingKeeper.Validator(ctx, valAddr1)
	require.NoError(t, err)

	// end period
	endingPeriod, err = input.DistKeeper.IncrementValidatorPeriod(ctx, val)
	require.NoError(t, err)

	// calculate delegation rewards for del1
	rewards, err = input.DistKeeper.CalculateDelegationRewards(ctx, val, del1, endingPeriod)
	require.NoError(t, err)

	require.Equal(t,
		// 98 * (40_000 / 50_000) * (10 / 20) * (9 / 10)
		// + 98 * (1_000_000 / 5_000_000) * (6 / 20) * (9 / 10)
		// + 98 * (2_000_000 / 5_000_000) * (4 / 20) * (9 / 10)
		// = 47.628
		int64(47),
		rewards.Sum().AmountOf(bondDenom).TruncateInt64())

	// calculate delegation rewards for del2
	rewards, err = input.DistKeeper.CalculateDelegationRewards(ctx, val, del2, endingPeriod)
	require.NoError(t, err)

	require.Equal(t,
		// 98 * (1_000_000 / 5_000_000) * (4 / 20) * (9 / 10)
		// = 3.5280000000000005

		// 98 * (1_000_000 / 5_000_000) * (4 / 20) * (9 / 10)
		// = 3.5280000000000005

		int64(7),
		rewards.Sum().AmountOf(bondDenom).TruncateInt64())

	// commission should be zero
	val1Commission, err = input.DistKeeper.ValidatorAccumulatedCommissions.Get(ctx, valAddr1)
	require.NoError(t, err)
	require.Equal(t,
		// 98 * (40_000 / 50_000) * (10 / 20) * (1 / 10) * 2 + 0.92 = 8.760000000000002
		// 98 * (1_000_000 / 5_000_000) * (6 / 20) * (1 / 10) * 2 + 0.588 = 1.7639999999999998
		// 98 * (3_000_000 / 5_000_000) * (4 / 20) * (1 / 10) * 2 + 0.176 = 2.528

		// 0.760000000000002 + 0.7639999999999998 + 0.528 = 2.052
		int64(2),
		val1Commission.Commissions.Sum().AmountOf(bondDenom).TruncateInt64())
}
