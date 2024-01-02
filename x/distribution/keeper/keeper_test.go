package keeper_test

import (
	"encoding/hex"
	"testing"

	"cosmossdk.io/math"
	customtypes "github.com/initia-labs/initia/x/distribution/types"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"
)

func TestSetWithdrawAddr(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	bz, err := hex.DecodeString("0000000000000000000000000000000000000001")
	require.NoError(t, err)
	oneAddr := sdk.AccAddress(bz)

	bz, err = hex.DecodeString("0000000000000000000000000000000000000002")
	require.NoError(t, err)
	twoAddr := sdk.AccAddress(bz)

	params, err := input.DistKeeper.Params.Get(ctx)
	require.NoError(t, err)
	params.WithdrawAddrEnabled = false
	err = input.DistKeeper.Params.Set(ctx, params)
	require.NoError(t, err)

	err = input.DistKeeper.SetWithdrawAddr(ctx, oneAddr, twoAddr)
	require.NotNil(t, err)

	params.WithdrawAddrEnabled = true
	err = input.DistKeeper.Params.Set(ctx, params)
	require.NoError(t, err)

	err = input.DistKeeper.SetWithdrawAddr(ctx, oneAddr, twoAddr)
	require.Nil(t, err)

	distrAcc := authtypes.NewEmptyModuleAccount(types.ModuleName)
	require.Error(t, input.DistKeeper.SetWithdrawAddr(ctx, oneAddr, distrAcc.GetAddress()))
}

func TestWithdrawValidatorCommission(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)
	validator, err := input.StakingKeeper.Validators.Get(ctx, valAddr)
	require.NoError(t, err)

	// set module account coins
	distrAcc := input.DistKeeper.GetDistributionAccount(ctx)
	coins := sdk.NewCoins(sdk.NewCoin("mytoken", math.NewInt(200)), sdk.NewCoin(bondDenom, math.NewInt(200)))
	input.BankKeeper.MintCoins(ctx, authtypes.Minter, coins)
	input.BankKeeper.SendCoinsFromModuleToModule(ctx, authtypes.Minter, distrAcc.GetName(), coins)

	// check initial balance
	balance := input.BankKeeper.GetAllBalances(ctx, sdk.AccAddress(valAddr))
	expTokens := input.StakingKeeper.VotingPowerFromConsensusPower(ctx, 99) // 100 - 1
	expCoins := sdk.NewCoins(sdk.NewCoin(bondDenom, expTokens))
	require.Equal(t, expCoins, balance)

	tokens := sdk.DecCoins{
		{Denom: "mytoken", Amount: math.LegacyNewDec(2)},
		{Denom: bondDenom, Amount: math.LegacyNewDec(2)},
	}
	input.DistKeeper.AllocateTokensToValidatorPool(ctx, validator, bondDenom, tokens)

	valCommissions := customtypes.DecPools{
		{Denom: bondDenom, DecCoins: sdk.DecCoins{
			{Denom: "mytoken", Amount: math.LegacyNewDec(5).Quo(math.LegacyNewDec(4))},
			{Denom: bondDenom, Amount: math.LegacyNewDec(3).Quo(math.LegacyNewDec(2))},
		}},
	}
	err = input.DistKeeper.ValidatorOutstandingRewards.Set(ctx, valAddr, customtypes.ValidatorOutstandingRewards{Rewards: valCommissions})
	require.NoError(t, err)
	err = input.DistKeeper.ValidatorAccumulatedCommissions.Set(ctx, valAddr, customtypes.ValidatorAccumulatedCommission{Commissions: valCommissions})
	require.NoError(t, err)

	// withdraw commission
	_, err = input.DistKeeper.WithdrawValidatorCommission(ctx, valAddr)
	require.NoError(t, err)

	// check balance increase
	balance = input.BankKeeper.GetAllBalances(ctx, sdk.AccAddress(valAddr))
	require.Equal(t, sdk.NewCoins(
		sdk.NewCoin("mytoken", math.NewInt(1)),
		sdk.NewCoin(bondDenom, expTokens.AddRaw(1)),
	), balance)

	// check remainder
	commissions, err := input.DistKeeper.ValidatorAccumulatedCommissions.Get(ctx, valAddr)
	require.NoError(t, err)
	require.Equal(t,
		customtypes.DecPools{
			{Denom: bondDenom, DecCoins: sdk.DecCoins{
				{Denom: "mytoken", Amount: math.LegacyNewDec(1).Quo(math.LegacyNewDec(4))},
				{Denom: bondDenom, Amount: math.LegacyNewDec(1).Quo(math.LegacyNewDec(2))},
			}}}, commissions.Commissions)
}

func TestGetTotalRewards(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	valAddr1 := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)
	valAddr2 := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 2)

	valCommissions := customtypes.DecPools{
		{Denom: bondDenom, DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDecWithPrec(500, 1)}}},
		{Denom: bondDenom, DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDecWithPrec(700, 1)}}},
	}

	err := input.DistKeeper.ValidatorOutstandingRewards.Set(ctx, valAddr1, customtypes.ValidatorOutstandingRewards{Rewards: valCommissions})
	require.NoError(t, err)
	err = input.DistKeeper.ValidatorOutstandingRewards.Set(ctx, valAddr2, customtypes.ValidatorOutstandingRewards{Rewards: valCommissions})
	require.NoError(t, err)

	expectedRewards := valCommissions.Sum().MulDec(math.LegacyNewDec(2))
	totalRewards := input.DistKeeper.GetTotalRewards(ctx)
	require.Equal(t, expectedRewards, totalRewards)
}

func TestFundCommunityPool(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	bz, err := hex.DecodeString("0000000000000000000000000000000000000001")
	require.NoError(t, err)
	oneAddr := sdk.AccAddress(bz)

	fees := sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(100)))
	input.Faucet.Fund(ctx, oneAddr, fees...)

	initPool, err := input.DistKeeper.FeePool.Get(ctx)
	require.NoError(t, err)
	require.Empty(t, initPool.CommunityPool)

	err = input.DistKeeper.FundCommunityPool(ctx, fees, oneAddr)
	require.Nil(t, err)

	feePool, err := input.DistKeeper.FeePool.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, initPool.CommunityPool.Add(sdk.NewDecCoinsFromCoins(fees...)...), feePool.CommunityPool)
	require.Empty(t, input.BankKeeper.GetAllBalances(ctx, oneAddr))
}
