package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/proto/tendermint/types"
	customtypes "github.com/initia-labs/initia/v1/x/distribution/types"
	stakingtypes "github.com/initia-labs/initia/v1/x/mstaking/types"
	"github.com/stretchr/testify/require"
)

func setRewardWeights(t *testing.T, ctx context.Context, input TestKeepers, weights []customtypes.RewardWeight) {
	// update reward weights
	params, err := input.DistKeeper.Params.Get(ctx)
	require.NoError(t, err)
	params.RewardWeights = weights
	err = input.DistKeeper.Params.Set(ctx, params)
	require.NoError(t, err)
}

func loadRewardsWeight(t *testing.T, ctx context.Context, input TestKeepers) ([]customtypes.RewardWeight, map[string]math.LegacyDec, math.LegacyDec) {
	params, err := input.DistKeeper.Params.Get(ctx)
	require.NoError(t, err)

	return input.DistKeeper.LoadRewardWeights(ctx, params)
}

func TestLoadRewardWeights(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	weights := []customtypes.RewardWeight{
		{
			Denom:  "aaa",
			Weight: math.LegacyNewDecWithPrec(3, 1),
		},
		{
			Denom:  "bar",
			Weight: math.LegacyNewDecWithPrec(4, 1),
		},
		{
			Denom:  "foo",
			Weight: math.LegacyNewDecWithPrec(3, 1),
		},
	}

	// update reward weights
	setRewardWeights(t, ctx, input, weights)

	_, loadedWeights, sum := loadRewardsWeight(t, ctx, input)
	require.Equal(t, math.LegacyNewDecWithPrec(3, 1), loadedWeights["aaa"])
	require.Equal(t, math.LegacyNewDecWithPrec(4, 1), loadedWeights["bar"])
	require.Equal(t, math.LegacyNewDecWithPrec(3, 1), loadedWeights["foo"])
	require.Equal(t, math.LegacyOneDec(), sum)
}

func TestLoadBondedTokens(t *testing.T) {
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
		sdk.NewCoins(sdk.NewInt64Coin("foo", 3_000_000), sdk.NewInt64Coin("bar", 1_000_000), sdk.NewInt64Coin("aaa", 20_000)), 1)
	valAddr2 := createValidatorWithCoin(ctx, input,
		sdk.NewCoins(sdk.NewInt64Coin("foo", 100_000_000), sdk.NewInt64Coin("bar", 100_000_000), sdk.NewInt64Coin("aaa", 100_000_000)),
		sdk.NewCoins(sdk.NewInt64Coin("foo", 6_000_000), sdk.NewInt64Coin("bar", 4_000_000), sdk.NewInt64Coin("aaa", 10_000)), 2)

	validator1, err := input.StakingKeeper.Validator(ctx, valAddr1)
	require.NoError(t, err)
	validator2, err := input.StakingKeeper.Validator(ctx, valAddr2)
	require.NoError(t, err)

	valConsPk1, err := validator1.ConsPubKey()
	require.NoError(t, err)
	valConsPk2, err := validator2.ConsPubKey()
	require.NoError(t, err)

	abciValA := abci.Validator{
		Address: valConsPk1.Address(),
		Power:   100,
	}
	abciValB := abci.Validator{
		Address: valConsPk2.Address(),
		Power:   400,
	}

	votes := []abci.VoteInfo{
		{
			Validator:   abciValA,
			BlockIdFlag: types.BlockIDFlagCommit,
		},
		{
			Validator:   abciValB,
			BlockIdFlag: types.BlockIDFlagCommit,
		},
	}

	_, rewardWeight, _ := loadRewardsWeight(t, ctx, input)
	validators, bondedTokens, bondedTokensSum, err := input.DistKeeper.LoadBondedTokens(ctx, votes, rewardWeight)
	require.NoError(t, err)
	require.Equal(t, validator1, validators[validator1.GetOperator()])
	require.Equal(t, validator2, validators[validator2.GetOperator()])
	for _, val := range bondedTokens["foo"] {
		if val.ValAddr == validator1.GetOperator() {
			require.Equal(t, math.NewInt(3_000_000), val.Amount)
		} else {
			require.Equal(t, math.NewInt(6_000_000), val.Amount)
		}
	}

	for _, val := range bondedTokens["bar"] {
		if val.ValAddr == validator1.GetOperator() {
			require.Equal(t, math.NewInt(1_000_000), val.Amount)
		} else {
			require.Equal(t, math.NewInt(4_000_000), val.Amount)
		}
	}

	for _, val := range bondedTokens["aaa"] {
		if val.ValAddr == validator1.GetOperator() {
			require.Equal(t, math.NewInt(20_000), val.Amount)
		} else {
			require.Equal(t, math.NewInt(10_000), val.Amount)
		}
	}
	require.Equal(t, math.NewInt(9_000_000), bondedTokensSum["foo"])
	require.Equal(t, math.NewInt(5_000_000), bondedTokensSum["bar"])
	require.Equal(t, math.NewInt(30_000), bondedTokensSum["aaa"])
}

func TestAllocateTokensToValidatorWithCommission(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	input.StakingKeeper.SetBondDenoms(ctx, []string{"foo", "bar"})

	// update reward weights
	// update reward weights
	setRewardWeights(t, ctx, input, []customtypes.RewardWeight{
		{
			Denom:  "foo",
			Weight: math.LegacyNewDecWithPrec(4, 1),
		},
		{
			Denom:  "bar",
			Weight: math.LegacyNewDecWithPrec(6, 1),
		},
	})
	input.VotingPowerKeeper.SetVotingPowerWeights(sdk.NewDecCoins(sdk.NewDecCoin("foo", math.NewInt(1)), sdk.NewDecCoin("bar", math.NewInt(4)), sdk.NewDecCoin("aaa", math.NewInt(10))))

	valAddr := createValidatorWithCoin(ctx, input,
		sdk.NewCoins(sdk.NewInt64Coin("foo", 100_000_000), sdk.NewInt64Coin("bar", 100_000_000)),
		sdk.NewCoins(sdk.NewInt64Coin("foo", 3_000_000), sdk.NewInt64Coin("bar", 5_000_000)), 1)

	validator, err := input.StakingKeeper.Validator(ctx, valAddr)
	require.NoError(t, err)

	tokens := sdk.DecCoins{{Denom: "reward1", Amount: math.LegacyNewDec(10)}, {Denom: "reward2", Amount: math.LegacyNewDec(20)}}
	input.DistKeeper.AllocateTokensToValidatorPool(ctx, validator, "bar", tokens)
	expectedCommission := customtypes.DecPools{{Denom: "bar", DecCoins: sdk.DecCoins{{Denom: "reward1", Amount: math.LegacyNewDec(1)}, {Denom: "reward2", Amount: math.LegacyNewDec(2)}}}}
	expectedRewards := customtypes.DecPools{{Denom: "bar", DecCoins: sdk.DecCoins{{Denom: "reward1", Amount: math.LegacyNewDec(9)}, {Denom: "reward2", Amount: math.LegacyNewDec(18)}}}}

	// check commission
	commission, err := input.DistKeeper.ValidatorAccumulatedCommissions.Get(ctx, valAddr)
	require.NoError(t, err)
	require.Equal(t, expectedCommission, commission.Commissions)
	// check current rewards

	currentRewards, err := input.DistKeeper.ValidatorCurrentRewards.Get(ctx, valAddr)
	require.NoError(t, err)
	require.Equal(t, expectedRewards, currentRewards.Rewards)
}

func TestAllocateTokensToManyValidators(t *testing.T) {
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
	validator2, err := input.StakingKeeper.Validator(ctx, valAddr2)
	require.NoError(t, err)

	valConsPk1, err := validator1.ConsPubKey()
	require.NoError(t, err)
	valConsPk2, err := validator2.ConsPubKey()
	require.NoError(t, err)

	abciValA := abci.Validator{
		Address: valConsPk1.Address(),
		Power:   100,
	}
	abciValB := abci.Validator{
		Address: valConsPk2.Address(),
		Power:   400,
	}

	// assert initial state: zero outstanding rewards, zero community pool, zero commission, zero current rewards
	val1OutRewards, err := input.DistKeeper.ValidatorOutstandingRewards.Get(ctx, valAddr1)
	require.NoError(t, err)
	require.True(t, val1OutRewards.Rewards.Sum().IsZero())
	val2OutRewards, err := input.DistKeeper.ValidatorOutstandingRewards.Get(ctx, valAddr2)
	require.NoError(t, err)
	require.True(t, val2OutRewards.Rewards.Sum().IsZero())

	feePool, err := input.DistKeeper.FeePool.Get(ctx)
	require.NoError(t, err)
	require.True(t, feePool.CommunityPool.IsZero())

	val1Commission, err := input.DistKeeper.ValidatorAccumulatedCommissions.Get(ctx, valAddr1)
	require.NoError(t, err)
	require.True(t, val1Commission.Commissions.Sum().Empty())

	val2Commission, err := input.DistKeeper.ValidatorAccumulatedCommissions.Get(ctx, valAddr2)
	require.NoError(t, err)
	require.True(t, val2Commission.Commissions.Sum().Empty())

	val1CurRewards, err := input.DistKeeper.ValidatorCurrentRewards.Get(ctx, valAddr1)
	require.NoError(t, err)
	require.True(t, val1CurRewards.Rewards.Sum().IsZero())

	val2CurRewards, err := input.DistKeeper.ValidatorCurrentRewards.Get(ctx, valAddr2)
	require.NoError(t, err)
	require.True(t, val2CurRewards.Rewards.Sum().IsZero())

	votes := []abci.VoteInfo{
		{
			Validator:   abciValA,
			BlockIdFlag: types.BlockIDFlagCommit,
		},
		{
			Validator:   abciValB,
			BlockIdFlag: types.BlockIDFlagCommit,
		},
	}

	// allocate tokens as if both had voted and second was proposer
	fees := sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(100)))

	feeCollector := input.AccountKeeper.GetModuleAccount(ctx, authtypes.FeeCollectorName)
	require.NotNil(t, feeCollector)
	input.Faucet.Fund(ctx, authtypes.NewModuleAddress(authtypes.FeeCollectorName), fees...)

	input.DistKeeper.AllocateTokens(ctx, 500, votes)

	// 98 outstanding rewards (100 less 2 to community pool)
	val1OutRewards, err = input.DistKeeper.ValidatorOutstandingRewards.Get(ctx, valAddr1)
	require.NoError(t, err)
	require.Equal(t,
		customtypes.DecPools{
			// 98 * (40_000 / 50_000) * (10 / 20) = 39.2
			{Denom: "aaa", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDecWithPrec(392, 1)}}},
			// 98 * (1_000_000 / 5_000_000) * (6 / 20) = 5.88
			{Denom: "bar", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDecWithPrec(588, 2)}}},
			// 98 * (2_000_000 / 4_000_000) * (4 / 20) = 9.8
			{Denom: "foo", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDecWithPrec(98, 1)}}},
		},
		val1OutRewards.Rewards)
	val2OutRewards, err = input.DistKeeper.ValidatorOutstandingRewards.Get(ctx, valAddr2)
	require.NoError(t, err)
	require.Equal(t,
		customtypes.DecPools{
			// 98 * (10_000 / 50_000) * (10 / 20) = 9.8
			{Denom: "aaa", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDecWithPrec(98, 1)}}},
			// 98 * (4_000_000 / 5_000_000) * (6 / 20) = 23.52
			{Denom: "bar", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDecWithPrec(2352, 2)}}},
			// 98 * (2_000_000 / 4_000_000) * (4 / 20) = 9.8
			{Denom: "foo", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDecWithPrec(98, 1)}}},
		},
		val2OutRewards.Rewards)
	// 2 community pool coins
	feePool, err = input.DistKeeper.FeePool.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDec(2)}}, feePool.CommunityPool)

	// 10% commission for first proposer,
	val1Commission, err = input.DistKeeper.ValidatorAccumulatedCommissions.Get(ctx, valAddr1)
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
	val2Commission, err = input.DistKeeper.ValidatorAccumulatedCommissions.Get(ctx, valAddr2)
	require.NoError(t, err)
	require.Equal(t,
		customtypes.DecPools{
			// 98 * (10_000 / 50_000) * (10 / 20) * (1 / 10) = 0.98
			{Denom: "aaa", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDecWithPrec(98, 2)}}},
			// 98 * (4_000_000 / 5_000_000) * (6 / 20) * (1 / 10) = 2.352
			{Denom: "bar", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDecWithPrec(2352, 3)}}},
			// 98 * (2_000_000 / 4_000_000) * (4 / 20) * (1 / 10) = 0.98
			{Denom: "foo", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDecWithPrec(98, 2)}}},
		},
		val2Commission.Commissions)

	// just staking.proportional for first proposer less commission
	val1CurRewards, err = input.DistKeeper.ValidatorCurrentRewards.Get(ctx, valAddr1)
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
		val1CurRewards.Rewards)
	val2CurRewards, err = input.DistKeeper.ValidatorCurrentRewards.Get(ctx, valAddr2)
	require.NoError(t, err)
	require.Equal(t,
		customtypes.DecPools{
			// 98 * (10_000 / 50_000) * (10 / 20) * (9 / 10) = 8.82
			{Denom: "aaa", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDecWithPrec(882, 2)}}},
			// 98 * (4_000_000 / 5_000_000) * (6 / 20) * (9 / 10) = 21.168
			{Denom: "bar", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDecWithPrec(21168, 3)}}},
			// 98 * (2_000_000 / 4_000_000) * (4 / 20) * (9 / 10) = 8.82
			{Denom: "foo", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDecWithPrec(882, 2)}}},
		},
		val2CurRewards.Rewards)
}

func TestAllocateTokensTruncation(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	valAddr1 := createValidatorWithBalance(ctx, input, 100_000_000, 110, 1)
	valAddr2 := createValidatorWithBalance(ctx, input, 100_000_000, 100, 2)
	valAddr3 := createValidatorWithBalance(ctx, input, 100_000_000, 100, 3)

	validator1, err := input.StakingKeeper.Validators.Get(ctx, valAddr1)
	require.NoError(t, err)
	validator2, err := input.StakingKeeper.Validators.Get(ctx, valAddr2)
	require.NoError(t, err)
	validator3, err := input.StakingKeeper.Validators.Get(ctx, valAddr3)
	require.NoError(t, err)

	valConsPk1, err := validator1.ConsPubKey()
	require.NoError(t, err)
	valConsPk2, err := validator2.ConsPubKey()
	require.NoError(t, err)
	valConsPk3, err := validator3.ConsPubKey()
	require.NoError(t, err)

	// create validator with 10% commission
	validator1.Commission = stakingtypes.NewCommission(
		math.LegacyNewDecWithPrec(1, 1),
		math.LegacyNewDecWithPrec(1, 1),
		math.LegacyNewDec(0),
	)

	validator2.Commission = stakingtypes.NewCommission(
		math.LegacyNewDecWithPrec(1, 1),
		math.LegacyNewDecWithPrec(1, 1),
		math.LegacyNewDec(0),
	)

	validator3.Commission = stakingtypes.NewCommission(
		math.LegacyNewDecWithPrec(1, 1),
		math.LegacyNewDecWithPrec(1, 1),
		math.LegacyNewDec(0),
	)

	abciValA := abci.Validator{
		Address: valConsPk1.Address(),
		Power:   11,
	}
	abciValB := abci.Validator{
		Address: valConsPk2.Address(),
		Power:   10,
	}
	abciValС := abci.Validator{
		Address: valConsPk3.Address(),
		Power:   10,
	}

	// assert initial state: zero outstanding rewards, zero community pool, zero commission, zero current rewards
	val1OutRewards, err := input.DistKeeper.ValidatorOutstandingRewards.Get(ctx, valAddr1)
	require.NoError(t, err)
	require.True(t, val1OutRewards.Rewards.Sum().IsZero())
	val2OutRewards, err := input.DistKeeper.ValidatorOutstandingRewards.Get(ctx, valAddr2)
	require.NoError(t, err)
	require.True(t, val2OutRewards.Rewards.Sum().IsZero())
	val3OutRewards, err := input.DistKeeper.ValidatorOutstandingRewards.Get(ctx, valAddr3)
	require.NoError(t, err)
	require.True(t, val3OutRewards.Rewards.Sum().IsZero())

	feePool, err := input.DistKeeper.FeePool.Get(ctx)
	require.NoError(t, err)
	require.True(t, feePool.CommunityPool.IsZero())

	val1Commission, err := input.DistKeeper.ValidatorAccumulatedCommissions.Get(ctx, valAddr1)
	require.NoError(t, err)
	require.True(t, val1Commission.Commissions.Sum().Empty())

	val2Commission, err := input.DistKeeper.ValidatorAccumulatedCommissions.Get(ctx, valAddr1)
	require.NoError(t, err)
	require.True(t, val2Commission.Commissions.Sum().Empty())

	val3Commission, err := input.DistKeeper.ValidatorAccumulatedCommissions.Get(ctx, valAddr3)
	require.NoError(t, err)
	require.True(t, val3Commission.Commissions.Sum().Empty())

	val1CurRewards, err := input.DistKeeper.ValidatorCurrentRewards.Get(ctx, valAddr1)
	require.NoError(t, err)
	require.True(t, val1CurRewards.Rewards.Sum().IsZero())

	val2CurRewards, err := input.DistKeeper.ValidatorCurrentRewards.Get(ctx, valAddr2)
	require.NoError(t, err)
	require.True(t, val2CurRewards.Rewards.Sum().IsZero())

	val3CurRewards, err := input.DistKeeper.ValidatorCurrentRewards.Get(ctx, valAddr3)
	require.NoError(t, err)
	require.True(t, val3CurRewards.Rewards.Sum().IsZero())

	// allocate tokens as if both had voted and second was proposer
	fees := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(634195840)))
	feeCollector := input.AccountKeeper.GetModuleAccount(ctx, authtypes.FeeCollectorName)
	require.NotNil(t, feeCollector)

	err = input.BankKeeper.MintCoins(ctx, authtypes.Minter, fees)
	require.NoError(t, err)
	err = input.BankKeeper.SendCoinsFromModuleToModule(ctx, authtypes.Minter, feeCollector.GetName(), fees)
	require.NoError(t, err)
	input.AccountKeeper.SetModuleAccount(ctx, feeCollector)

	votes := []abci.VoteInfo{
		{
			Validator:   abciValA,
			BlockIdFlag: types.BlockIDFlagCommit,
		},
		{
			Validator:   abciValB,
			BlockIdFlag: types.BlockIDFlagCommit,
		},
		{
			Validator:   abciValС,
			BlockIdFlag: types.BlockIDFlagCommit,
		},
	}
	input.DistKeeper.AllocateTokens(ctx, 31, votes)

	val1OutRewards, err = input.DistKeeper.ValidatorOutstandingRewards.Get(ctx, valAddr1)
	require.NoError(t, err)
	val2OutRewards, err = input.DistKeeper.ValidatorOutstandingRewards.Get(ctx, valAddr2)
	require.NoError(t, err)
	val3OutRewards, err = input.DistKeeper.ValidatorOutstandingRewards.Get(ctx, valAddr3)
	require.NoError(t, err)

	require.True(t, val1OutRewards.Rewards.Sum().IsValid())
	require.True(t, val2OutRewards.Rewards.Sum().IsValid())
	require.True(t, val3OutRewards.Rewards.Sum().IsValid())
}

func Test_SwapToBase(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	valAddr1 := createValidatorWithBalance(ctx, input, 100_000_000, 110, 1)

	validator1, err := input.StakingKeeper.Validators.Get(ctx, valAddr1)
	require.NoError(t, err)

	valConsPk1, err := validator1.ConsPubKey()
	require.NoError(t, err)

	// create validator with 10% commission
	validator1.Commission = stakingtypes.NewCommission(
		math.LegacyNewDecWithPrec(1, 1),
		math.LegacyNewDecWithPrec(1, 1),
		math.LegacyNewDec(0),
	)

	abciValA := abci.Validator{
		Address: valConsPk1.Address(),
		Power:   10,
	}

	// assert initial state: zero outstanding rewards, zero community pool, zero commission, zero current rewards
	val1OutRewards, err := input.DistKeeper.ValidatorOutstandingRewards.Get(ctx, valAddr1)
	require.NoError(t, err)
	require.True(t, val1OutRewards.Rewards.Sum().IsZero())
	feePool, err := input.DistKeeper.FeePool.Get(ctx)
	require.NoError(t, err)
	require.True(t, feePool.CommunityPool.IsZero())
	val1Commission, err := input.DistKeeper.ValidatorAccumulatedCommissions.Get(ctx, valAddr1)
	require.NoError(t, err)
	require.True(t, val1Commission.Commissions.Sum().IsZero())
	val1CurRewards, err := input.DistKeeper.ValidatorCurrentRewards.Get(ctx, valAddr1)
	require.NoError(t, err)
	require.True(t, val1CurRewards.Rewards.Sum().IsZero())

	// allocate tokens as if both had voted and second was proposer
	fees := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(1_000_000_000_000)))
	feeCollector := input.AccountKeeper.GetModuleAccount(ctx, authtypes.FeeCollectorName)
	require.NotNil(t, feeCollector)

	err = input.BankKeeper.MintCoins(ctx, authtypes.Minter, fees)
	require.NoError(t, err)
	err = input.BankKeeper.SendCoinsFromModuleToModule(ctx, authtypes.Minter, feeCollector.GetName(), fees)
	require.NoError(t, err)
	input.AccountKeeper.SetModuleAccount(ctx, feeCollector)

	votes := []abci.VoteInfo{
		{
			Validator:   abciValA,
			BlockIdFlag: types.BlockIDFlagCommit,
		},
	}
	// set dex price
	input.DexKeeper.SetPrice(sdk.DefaultBondDenom, math.LegacyOneDec())
	input.DistKeeper.AllocateTokens(ctx, 31, votes)

	params, err := input.DistKeeper.Params.Get(ctx)
	require.NoError(t, err)
	taxRate := params.CommunityTax
	baseDenom, err := input.MoveKeeper.BaseDenom(ctx)
	require.NoError(t, err)

	val1OutRewards, err = input.DistKeeper.ValidatorOutstandingRewards.Get(ctx, valAddr1)
	require.NoError(t, err)
	require.Equal(t,
		val1OutRewards.Rewards.CoinsOf(baseDenom),
		sdk.NewDecCoins(sdk.NewDecCoin(baseDenom, math.LegacyOneDec().Sub(taxRate).MulInt(fees[0].Amount).TruncateInt())),
	)
}
