package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	abci "github.com/cometbft/cometbft/abci/types"
	customtypes "github.com/initia-labs/initia/x/distribution/types"
	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"
	"github.com/stretchr/testify/require"
)

func TestLoadRewardWeights(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	weights := []customtypes.RewardWeight{
		{
			Denom:  "aaa",
			Weight: sdk.NewDecWithPrec(3, 1),
		},
		{
			Denom:  "bar",
			Weight: sdk.NewDecWithPrec(4, 1),
		},
		{
			Denom:  "foo",
			Weight: sdk.NewDecWithPrec(3, 1),
		},
	}
	input.DistKeeper.SetRewardWeights(ctx, weights)

	_, loadedWeights, sum := input.DistKeeper.LoadRewardWeights(ctx)
	require.Equal(t, sdk.NewDecWithPrec(3, 1), loadedWeights["aaa"])
	require.Equal(t, sdk.NewDecWithPrec(4, 1), loadedWeights["bar"])
	require.Equal(t, sdk.NewDecWithPrec(3, 1), loadedWeights["foo"])
	require.Equal(t, sdk.OneDec(), sum)
}

func TestLoadBondedTokens(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	input.StakingKeeper.SetBondDenoms(ctx, []string{"foo", "bar"})
	input.DistKeeper.SetRewardWeights(ctx, []customtypes.RewardWeight{
		{
			Denom:  "foo",
			Weight: sdk.NewDecWithPrec(4, 1),
		},
		{
			Denom:  "bar",
			Weight: sdk.NewDecWithPrec(6, 1),
		},
	})

	valAddr1 := createValidatorWithCoin(ctx, input,
		sdk.NewCoins(sdk.NewInt64Coin("foo", 100_000_000), sdk.NewInt64Coin("bar", 100_000_000)),
		sdk.NewCoins(sdk.NewInt64Coin("foo", 3_000_000), sdk.NewInt64Coin("bar", 5_000_000)), 1)
	valAddr2 := createValidatorWithCoin(ctx, input,
		sdk.NewCoins(sdk.NewInt64Coin("foo", 100_000_000), sdk.NewInt64Coin("bar", 100_000_000)),
		sdk.NewCoins(sdk.NewInt64Coin("foo", 5_000_000), sdk.NewInt64Coin("bar", 3_000_000)), 2)

	validator1, found := input.StakingKeeper.GetValidator(ctx, valAddr1)
	require.True(t, found)
	validator2, found := input.StakingKeeper.GetValidator(ctx, valAddr2)
	require.True(t, found)

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
			Validator:       abciValA,
			SignedLastBlock: true,
		},
		{
			Validator:       abciValB,
			SignedLastBlock: true,
		},
	}

	_, rewardWeight, _ := input.DistKeeper.LoadRewardWeights(ctx)
	validators, bondedTokens, bondedTokensSum := input.DistKeeper.LoadBondedTokens(ctx, votes, rewardWeight)
	require.Equal(t, validator1, validators[string(valConsPk1.Address())])
	require.Equal(t, validator2, validators[string(valConsPk2.Address())])
	for _, val := range bondedTokens["foo"] {
		if val.ValAddr == string(valConsPk1.Address()) {
			require.Equal(t, sdk.NewInt(3_000_000), val.Amount)
		} else {
			sdk.NewInt(5_000_000)
		}
	}

	for _, val := range bondedTokens["bar"] {
		if val.ValAddr == string(valConsPk1.Address()) {
			require.Equal(t, sdk.NewInt(5_000_000), val.Amount)
		} else {
			sdk.NewInt(3_000_000)
		}
	}
	require.Equal(t, sdk.NewInt(8_000_000), bondedTokensSum["foo"])
	require.Equal(t, sdk.NewInt(8_000_000), bondedTokensSum["bar"])
}

func TestAllocateTokensToValidatorWithCommission(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)

	validator, found := input.StakingKeeper.GetValidator(ctx, valAddr)
	require.True(t, found)

	tokens := sdk.DecCoins{{Denom: bondDenom, Amount: sdk.NewDec(10)}}
	input.DistKeeper.AllocateTokensToValidatorPool(ctx, validator, bondDenom, tokens)
	expected := customtypes.DecPools{{Denom: bondDenom, DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: sdk.NewDec(5)}}}}

	// check commission
	require.Equal(t, expected, input.DistKeeper.GetValidatorAccumulatedCommission(ctx, validator.GetOperator()).Commissions)
	// check current rewards
	require.Equal(t, expected, input.DistKeeper.GetValidatorCurrentRewards(ctx, validator.GetOperator()).Rewards)
}

func TestAllocateTokensToManyValidators(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	input.StakingKeeper.SetBondDenoms(ctx, []string{"foo", "bar"})
	input.DistKeeper.SetRewardWeights(ctx, []customtypes.RewardWeight{
		{
			Denom:  "foo",
			Weight: sdk.NewDecWithPrec(4, 1),
		},
		{
			Denom:  "bar",
			Weight: sdk.NewDecWithPrec(6, 1),
		},
	})

	valAddr1 := createValidatorWithCoin(ctx, input,
		sdk.NewCoins(sdk.NewInt64Coin("foo", 100_000_000), sdk.NewInt64Coin("bar", 100_000_000)),
		sdk.NewCoins(sdk.NewInt64Coin("foo", 3_000_000), sdk.NewInt64Coin("bar", 5_000_000)), 1)
	valAddr2 := createValidatorWithCoin(ctx, input,
		sdk.NewCoins(sdk.NewInt64Coin("foo", 100_000_000), sdk.NewInt64Coin("bar", 100_000_000)),
		sdk.NewCoins(sdk.NewInt64Coin("foo", 5_000_000), sdk.NewInt64Coin("bar", 3_000_000)), 2)

	validator1, found := input.StakingKeeper.GetValidator(ctx, valAddr1)
	require.True(t, found)
	validator2, found := input.StakingKeeper.GetValidator(ctx, valAddr2)
	require.True(t, found)

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
	require.True(t, input.DistKeeper.GetValidatorOutstandingRewards(ctx, valAddr1).Rewards.Sum().IsZero())
	require.True(t, input.DistKeeper.GetValidatorOutstandingRewards(ctx, valAddr2).Rewards.Sum().IsZero())
	require.True(t, input.DistKeeper.GetFeePool(ctx).CommunityPool.IsZero())
	require.True(t, input.DistKeeper.GetValidatorAccumulatedCommission(ctx, valAddr1).Commissions.Sum().Empty())
	require.True(t, input.DistKeeper.GetValidatorAccumulatedCommission(ctx, valAddr2).Commissions.Sum().Empty())
	require.True(t, input.DistKeeper.GetValidatorCurrentRewards(ctx, valAddr1).Rewards.Sum().IsZero())
	require.True(t, input.DistKeeper.GetValidatorCurrentRewards(ctx, valAddr2).Rewards.Sum().IsZero())

	votes := []abci.VoteInfo{
		{
			Validator:       abciValA,
			SignedLastBlock: true,
		},
		{
			Validator:       abciValB,
			SignedLastBlock: true,
		},
	}

	// allocate tokens as if both had voted and second was proposer
	fees := sdk.NewCoins(sdk.NewCoin(bondDenom, sdk.NewInt(100)))

	feeCollector := input.AccountKeeper.GetModuleAccount(ctx, authtypes.FeeCollectorName)
	require.NotNil(t, feeCollector)
	input.Faucet.Fund(ctx, authtypes.NewModuleAddress(authtypes.FeeCollectorName), fees...)

	input.DistKeeper.AllocateTokens(ctx, 200, votes)

	// 98 outstanding rewards (100 less 2 to community pool)
	require.Equal(t,
		customtypes.DecPools{
			{Denom: "bar", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: sdk.NewDecWithPrec(3675, 2)}}},
			{Denom: "foo", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: sdk.NewDecWithPrec(147, 1)}}},
		},
		input.DistKeeper.GetValidatorOutstandingRewards(ctx, valAddr1).Rewards)
	require.Equal(t,
		customtypes.DecPools{
			{Denom: "bar", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: sdk.NewDecWithPrec(2205, 2)}}},
			{Denom: "foo", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: sdk.NewDecWithPrec(245, 1)}}},
		},
		input.DistKeeper.GetValidatorOutstandingRewards(ctx, valAddr2).Rewards)
	// 2 community pool coins
	require.Equal(t, sdk.DecCoins{{Denom: bondDenom, Amount: sdk.NewDec(2)}}, input.DistKeeper.GetFeePool(ctx).CommunityPool)

	// 50% commission for first proposer, (0.5 * 98%) * 100 / 2 = 24.5
	require.Equal(t,
		customtypes.DecPools{
			{Denom: "bar", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: sdk.NewDecWithPrec(18375, 3)}}},
			{Denom: "foo", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: sdk.NewDecWithPrec(735, 2)}}},
		},
		input.DistKeeper.GetValidatorAccumulatedCommission(ctx, valAddr1).Commissions)
	require.Equal(t,
		customtypes.DecPools{
			{Denom: "bar", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: sdk.NewDecWithPrec(11025, 3)}}},
			{Denom: "foo", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: sdk.NewDecWithPrec(1225, 2)}}},
		},
		input.DistKeeper.GetValidatorAccumulatedCommission(ctx, valAddr2).Commissions)

	// just staking.proportional for first proposer less commission = (0.5 * 98%) * 100 / 2 = 24.5
	require.Equal(t,
		customtypes.DecPools{
			{Denom: "bar", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: sdk.NewDecWithPrec(18375, 3)}}},
			{Denom: "foo", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: sdk.NewDecWithPrec(735, 2)}}},
		},
		input.DistKeeper.GetValidatorCurrentRewards(ctx, validator1.GetOperator()).Rewards)
	require.Equal(t,
		customtypes.DecPools{
			{Denom: "bar", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: sdk.NewDecWithPrec(11025, 3)}}},
			{Denom: "foo", DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: sdk.NewDecWithPrec(1225, 2)}}},
		},
		input.DistKeeper.GetValidatorCurrentRewards(ctx, validator2.GetOperator()).Rewards)
}

func TestAllocateTokensTruncation(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	valAddr1 := createValidatorWithBalance(ctx, input, 100_000_000, 110, 1)
	valAddr2 := createValidatorWithBalance(ctx, input, 100_000_000, 100, 2)
	valAddr3 := createValidatorWithBalance(ctx, input, 100_000_000, 100, 3)

	validator1, found := input.StakingKeeper.GetValidator(ctx, valAddr1)
	require.True(t, found)
	validator2, found := input.StakingKeeper.GetValidator(ctx, valAddr2)
	require.True(t, found)
	validator3, found := input.StakingKeeper.GetValidator(ctx, valAddr3)
	require.True(t, found)

	valConsPk1, err := validator1.ConsPubKey()
	require.NoError(t, err)
	valConsPk2, err := validator2.ConsPubKey()
	require.NoError(t, err)
	valConsPk3, err := validator3.ConsPubKey()
	require.NoError(t, err)

	// create validator with 10% commission
	validator1.Commission = stakingtypes.NewCommission(
		sdk.NewDecWithPrec(1, 1),
		sdk.NewDecWithPrec(1, 1),
		sdk.NewDec(0),
	)

	validator2.Commission = stakingtypes.NewCommission(
		sdk.NewDecWithPrec(1, 1),
		sdk.NewDecWithPrec(1, 1),
		sdk.NewDec(0),
	)

	validator3.Commission = stakingtypes.NewCommission(
		sdk.NewDecWithPrec(1, 1),
		sdk.NewDecWithPrec(1, 1),
		sdk.NewDec(0),
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
	require.True(t, input.DistKeeper.GetValidatorOutstandingRewards(ctx, valAddr1).Rewards.Sum().IsZero())
	require.True(t, input.DistKeeper.GetValidatorOutstandingRewards(ctx, valAddr2).Rewards.Sum().IsZero())
	require.True(t, input.DistKeeper.GetValidatorOutstandingRewards(ctx, valAddr2).Rewards.Sum().IsZero())
	require.True(t, input.DistKeeper.GetFeePool(ctx).CommunityPool.IsZero())
	require.True(t, input.DistKeeper.GetValidatorAccumulatedCommission(ctx, valAddr1).Commissions.Sum().IsZero())
	require.True(t, input.DistKeeper.GetValidatorAccumulatedCommission(ctx, valAddr2).Commissions.Sum().IsZero())
	require.True(t, input.DistKeeper.GetValidatorCurrentRewards(ctx, valAddr1).Rewards.Sum().IsZero())
	require.True(t, input.DistKeeper.GetValidatorCurrentRewards(ctx, valAddr2).Rewards.Sum().IsZero())

	// allocate tokens as if both had voted and second was proposer
	fees := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(634195840)))
	feeCollector := input.AccountKeeper.GetModuleAccount(ctx, authtypes.FeeCollectorName)
	require.NotNil(t, feeCollector)

	err = input.BankKeeper.MintCoins(ctx, authtypes.Minter, fees)
	require.NoError(t, err)
	err = input.BankKeeper.SendCoinsFromModuleToModule(ctx, authtypes.Minter, feeCollector.GetName(), fees)
	require.NoError(t, err)
	input.AccountKeeper.SetModuleAccount(ctx, feeCollector)

	votes := []abci.VoteInfo{
		{
			Validator:       abciValA,
			SignedLastBlock: true,
		},
		{
			Validator:       abciValB,
			SignedLastBlock: true,
		},
		{
			Validator:       abciValС,
			SignedLastBlock: true,
		},
	}
	input.DistKeeper.AllocateTokens(ctx, 31, votes)

	require.True(t, input.DistKeeper.GetValidatorOutstandingRewards(ctx, valAddr1).Rewards.Sum().IsValid())
	require.True(t, input.DistKeeper.GetValidatorOutstandingRewards(ctx, valAddr2).Rewards.Sum().IsValid())
	require.True(t, input.DistKeeper.GetValidatorOutstandingRewards(ctx, valAddr3).Rewards.Sum().IsValid())
}

func Test_SwapToBase(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	valAddr1 := createValidatorWithBalance(ctx, input, 100_000_000, 110, 1)

	validator1, found := input.StakingKeeper.GetValidator(ctx, valAddr1)
	require.True(t, found)

	valConsPk1, err := validator1.ConsPubKey()
	require.NoError(t, err)

	// create validator with 10% commission
	validator1.Commission = stakingtypes.NewCommission(
		sdk.NewDecWithPrec(1, 1),
		sdk.NewDecWithPrec(1, 1),
		sdk.NewDec(0),
	)

	abciValA := abci.Validator{
		Address: valConsPk1.Address(),
		Power:   10,
	}

	// assert initial state: zero outstanding rewards, zero community pool, zero commission, zero current rewards
	require.True(t, input.DistKeeper.GetValidatorOutstandingRewards(ctx, valAddr1).Rewards.Sum().IsZero())
	require.True(t, input.DistKeeper.GetFeePool(ctx).CommunityPool.IsZero())
	require.True(t, input.DistKeeper.GetValidatorAccumulatedCommission(ctx, valAddr1).Commissions.Sum().IsZero())
	require.True(t, input.DistKeeper.GetValidatorCurrentRewards(ctx, valAddr1).Rewards.Sum().IsZero())

	// allocate tokens as if both had voted and second was proposer
	fees := sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(1_000_000_000_000)))
	feeCollector := input.AccountKeeper.GetModuleAccount(ctx, authtypes.FeeCollectorName)
	require.NotNil(t, feeCollector)

	err = input.BankKeeper.MintCoins(ctx, authtypes.Minter, fees)
	require.NoError(t, err)
	err = input.BankKeeper.SendCoinsFromModuleToModule(ctx, authtypes.Minter, feeCollector.GetName(), fees)
	require.NoError(t, err)
	input.AccountKeeper.SetModuleAccount(ctx, feeCollector)

	votes := []abci.VoteInfo{
		{
			Validator:       abciValA,
			SignedLastBlock: true,
		},
	}
	// set dex price
	input.DexKeeper.SetPrice(sdk.DefaultBondDenom, sdk.OneDec())
	input.DistKeeper.AllocateTokens(ctx, 31, votes)

	taxRate := input.DistKeeper.GetCommunityTax(ctx)
	baseDenom := input.MoveKeeper.BaseDenom(ctx)

	require.Equal(t,
		input.DistKeeper.GetValidatorOutstandingRewards(ctx, valAddr1).Rewards.CoinsOf(baseDenom),
		sdk.NewDecCoins(sdk.NewDecCoin(baseDenom, sdk.OneDec().Sub(taxRate).MulInt(fees[0].Amount).TruncateInt())),
	)
}
