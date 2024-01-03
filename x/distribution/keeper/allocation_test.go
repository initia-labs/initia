package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/proto/tendermint/types"
	customtypes "github.com/initia-labs/initia/x/distribution/types"
	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"
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

	valAddr1 := createValidatorWithCoin(ctx, input,
		sdk.NewCoins(sdk.NewInt64Coin("foo", 100_000_000), sdk.NewInt64Coin("bar", 100_000_000)),
		sdk.NewCoins(sdk.NewInt64Coin("foo", 3_000_000), sdk.NewInt64Coin("bar", 5_000_000)), 1)
	valAddr2 := createValidatorWithCoin(ctx, input,
		sdk.NewCoins(sdk.NewInt64Coin("foo", 100_000_000), sdk.NewInt64Coin("bar", 100_000_000)),
		sdk.NewCoins(sdk.NewInt64Coin("foo", 5_000_000), sdk.NewInt64Coin("bar", 3_000_000)), 2)

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
		Power:   100,
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
	require.Equal(t, validator1, validators[string(valConsPk1.Address())])
	require.Equal(t, validator2, validators[string(valConsPk2.Address())])
	for _, val := range bondedTokens["foo"] {
		if val.ValAddr == string(valConsPk1.Address()) {
			require.Equal(t, math.NewInt(3_000_000), val.Amount)
		} else {
			math.NewInt(5_000_000)
		}
	}

	for _, val := range bondedTokens["bar"] {
		if val.ValAddr == string(valConsPk1.Address()) {
			require.Equal(t, math.NewInt(5_000_000), val.Amount)
		} else {
			math.NewInt(3_000_000)
		}
	}
	require.Equal(t, math.NewInt(8_000_000), bondedTokensSum["foo"])
	require.Equal(t, math.NewInt(8_000_000), bondedTokensSum["bar"])
}

func TestAllocateTokensToValidatorWithCommission(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)

	validator, err := input.StakingKeeper.Validator(ctx, valAddr)
	require.NoError(t, err)

	tokens := sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDec(10)}}
	input.DistKeeper.AllocateTokensToValidatorPool(ctx, validator, bondDenom, tokens)
	expected := customtypes.DecPools{{Denom: bondDenom, DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDec(5)}}}}

	// check commission
	commission, err := input.DistKeeper.ValidatorAccumulatedCommissions.Get(ctx, valAddr)
	require.NoError(t, err)
	require.Equal(t, expected, commission.Commissions)
	// check current rewards

	currentRewards, err := input.DistKeeper.ValidatorCurrentRewards.Get(ctx, valAddr)
	require.NoError(t, err)
	require.Equal(t, expected, currentRewards.Rewards)
}

func TestAllocateTokensToManyValidators(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	input.StakingKeeper.SetBondDenoms(ctx, []string{"foo", "bar"})
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

	valAddr1 := createValidatorWithCoin(ctx, input,
		sdk.NewCoins(sdk.NewInt64Coin("foo", 100_000_000), sdk.NewInt64Coin("bar", 100_000_000)),
		sdk.NewCoins(sdk.NewInt64Coin("foo", 3_000_000), sdk.NewInt64Coin("bar", 5_000_000)), 1)
	valAddr2 := createValidatorWithCoin(ctx, input,
		sdk.NewCoins(sdk.NewInt64Coin("foo", 100_000_000), sdk.NewInt64Coin("bar", 100_000_000)),
		sdk.NewCoins(sdk.NewInt64Coin("foo", 5_000_000), sdk.NewInt64Coin("bar", 3_000_000)), 2)

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
		Power:   100,
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

	input.DistKeeper.AllocateTokens(ctx, 200, votes)

	// 98 outstanding rewards (100 less 2 to community pool)
	val1OutRewards, err = input.DistKeeper.ValidatorOutstandingRewards.Get(ctx, valAddr1)
	require.NoError(t, err)
	require.Equal(t,
		customtypes.DecPools{
			{Denom: "bar", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDecWithPrec(3675, 2)}}},
			{Denom: "foo", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDecWithPrec(147, 1)}}},
		},
		val1OutRewards.Rewards)
	val2OutRewards, err = input.DistKeeper.ValidatorOutstandingRewards.Get(ctx, valAddr2)
	require.NoError(t, err)
	require.Equal(t,
		customtypes.DecPools{
			{Denom: "bar", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDecWithPrec(2205, 2)}}},
			{Denom: "foo", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDecWithPrec(245, 1)}}},
		},
		val2OutRewards.Rewards)
	// 2 community pool coins
	feePool, err = input.DistKeeper.FeePool.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDec(2)}}, feePool.CommunityPool)

	// 50% commission for first proposer, (0.5 * 98%) * 100 / 2 = 24.5
	val1Commission, err = input.DistKeeper.ValidatorAccumulatedCommissions.Get(ctx, valAddr1)
	require.NoError(t, err)
	require.Equal(t,
		customtypes.DecPools{
			{Denom: "bar", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDecWithPrec(18375, 3)}}},
			{Denom: "foo", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDecWithPrec(735, 2)}}},
		},
		val1Commission.Commissions)
	val2Commission, err = input.DistKeeper.ValidatorAccumulatedCommissions.Get(ctx, valAddr2)
	require.NoError(t, err)
	require.Equal(t,
		customtypes.DecPools{
			{Denom: "bar", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDecWithPrec(11025, 3)}}},
			{Denom: "foo", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDecWithPrec(1225, 2)}}},
		},
		val2Commission.Commissions)

	// just staking.proportional for first proposer less commission = (0.5 * 98%) * 100 / 2 = 24.5
	val1CurRewards, err = input.DistKeeper.ValidatorCurrentRewards.Get(ctx, valAddr1)
	require.NoError(t, err)
	require.Equal(t,
		customtypes.DecPools{
			{Denom: "bar", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDecWithPrec(18375, 3)}}},
			{Denom: "foo", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDecWithPrec(735, 2)}}},
		},
		val1CurRewards.Rewards)
	val2CurRewards, err = input.DistKeeper.ValidatorCurrentRewards.Get(ctx, valAddr2)
	require.NoError(t, err)
	require.Equal(t,
		customtypes.DecPools{
			{Denom: "bar", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDecWithPrec(11025, 3)}}},
			{Denom: "foo", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDecWithPrec(1225, 2)}}},
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
