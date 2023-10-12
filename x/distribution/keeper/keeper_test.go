package keeper_test

import (
	"encoding/hex"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"
	customtypes "github.com/initia-labs/initia/x/distribution/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetWithdrawAddr(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	bz, err := hex.DecodeString("0000000000000000000000000000000000000001")
	require.NoError(t, err)
	oneAddr := sdk.AccAddress(bz)

	bz, err = hex.DecodeString("0000000000000000000000000000000000000002")
	require.NoError(t, err)
	twoAddr := sdk.AccAddress(bz)

	params := input.DistKeeper.GetParams(ctx)
	params.WithdrawAddrEnabled = false
	input.DistKeeper.SetParams(ctx, params)

	err = input.DistKeeper.SetWithdrawAddr(ctx, oneAddr, twoAddr)
	require.NotNil(t, err)

	params.WithdrawAddrEnabled = true
	input.DistKeeper.SetParams(ctx, params)

	err = input.DistKeeper.SetWithdrawAddr(ctx, oneAddr, twoAddr)
	require.Nil(t, err)

	distrAcc := authtypes.NewEmptyModuleAccount(types.ModuleName)
	require.Error(t, input.DistKeeper.SetWithdrawAddr(ctx, oneAddr, distrAcc.GetAddress()))
}

func TestWithdrawValidatorCommission(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)
	validator, found := input.StakingKeeper.GetValidator(ctx, valAddr)
	require.True(t, found)

	// set module account coins
	distrAcc := input.DistKeeper.GetDistributionAccount(ctx)
	coins := sdk.NewCoins(sdk.NewCoin("mytoken", sdk.NewInt(200)), sdk.NewCoin(bondDenom, sdk.NewInt(200)))
	input.BankKeeper.MintCoins(ctx, authtypes.Minter, coins)
	input.BankKeeper.SendCoinsFromModuleToModule(ctx, authtypes.Minter, distrAcc.GetName(), coins)

	// check initial balance
	balance := input.BankKeeper.GetAllBalances(ctx, sdk.AccAddress(valAddr))
	expTokens := input.StakingKeeper.VotingPowerFromConsensusPower(ctx, 99) // 100 - 1
	expCoins := sdk.NewCoins(sdk.NewCoin(bondDenom, expTokens))
	require.Equal(t, expCoins, balance)

	tokens := sdk.DecCoins{
		{Denom: "mytoken", Amount: sdk.NewDec(2)},
		{Denom: bondDenom, Amount: sdk.NewDec(2)},
	}
	input.DistKeeper.AllocateTokensToValidatorPool(ctx, validator, bondDenom, tokens)

	valCommissions := customtypes.DecPools{
		{Denom: bondDenom, DecCoins: sdk.DecCoins{
			{Denom: "mytoken", Amount: sdk.NewDec(5).Quo(sdk.NewDec(4))},
			{Denom: bondDenom, Amount: sdk.NewDec(3).Quo(sdk.NewDec(2))},
		}},
	}
	input.DistKeeper.SetValidatorOutstandingRewards(ctx, valAddr, customtypes.ValidatorOutstandingRewards{Rewards: valCommissions})
	input.DistKeeper.SetValidatorAccumulatedCommission(ctx, valAddr, customtypes.ValidatorAccumulatedCommission{Commissions: valCommissions})

	// withdraw commission
	_, err := input.DistKeeper.WithdrawValidatorCommission(ctx, valAddr)
	require.NoError(t, err)

	// check balance increase
	balance = input.BankKeeper.GetAllBalances(ctx, sdk.AccAddress(valAddr))
	require.Equal(t, sdk.NewCoins(
		sdk.NewCoin("mytoken", math.NewInt(1)),
		sdk.NewCoin(bondDenom, expTokens.AddRaw(1)),
	), balance)

	// check remainder
	remainder := input.DistKeeper.GetValidatorAccumulatedCommission(ctx, valAddr).Commissions
	require.Equal(t,
		customtypes.DecPools{
			{Denom: bondDenom, DecCoins: sdk.DecCoins{
				{Denom: "mytoken", Amount: sdk.NewDec(1).Quo(sdk.NewDec(4))},
				{Denom: bondDenom, Amount: sdk.NewDec(1).Quo(sdk.NewDec(2))},
			}}}, remainder)

	require.True(t, true)
}

func TestGetTotalRewards(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	valAddr1 := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)
	valAddr2 := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 2)

	valCommissions := customtypes.DecPools{
		{Denom: bondDenom, DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: sdk.NewDecWithPrec(500, 1)}}},
		{Denom: bondDenom, DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: sdk.NewDecWithPrec(700, 1)}}},
	}

	input.DistKeeper.SetValidatorOutstandingRewards(ctx, valAddr1, customtypes.ValidatorOutstandingRewards{Rewards: valCommissions})
	input.DistKeeper.SetValidatorOutstandingRewards(ctx, valAddr2, customtypes.ValidatorOutstandingRewards{Rewards: valCommissions})

	expectedRewards := valCommissions.Sum().MulDec(sdk.NewDec(2))
	totalRewards := input.DistKeeper.GetTotalRewards(ctx)
	require.Equal(t, expectedRewards, totalRewards)
}

func TestFundCommunityPool(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	bz, err := hex.DecodeString("0000000000000000000000000000000000000001")
	require.NoError(t, err)
	oneAddr := sdk.AccAddress(bz)

	fees := sdk.NewCoins(sdk.NewCoin(bondDenom, sdk.NewInt(100)))
	input.Faucet.Fund(ctx, oneAddr, fees...)

	initPool := input.DistKeeper.GetFeePool(ctx)
	assert.Empty(t, initPool.CommunityPool)

	err = input.DistKeeper.FundCommunityPool(ctx, fees, oneAddr)
	assert.Nil(t, err)

	assert.Equal(t, initPool.CommunityPool.Add(sdk.NewDecCoinsFromCoins(fees...)...), input.DistKeeper.GetFeePool(ctx).CommunityPool)
	assert.Empty(t, input.BankKeeper.GetAllBalances(ctx, oneAddr))
}
