package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	"github.com/initia-labs/initia/v1/x/mstaking/types"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func Test_MatureUnbondingRedelegation(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr1 := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)
	valAddr2 := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 2)

	_, _, err := input.StakingKeeper.Undelegate(ctx, valAddr1.Bytes(), valAddr1, sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, math.NewInt(500_000))))
	require.NoError(t, err)

	_, err = input.StakingKeeper.BeginRedelegation(ctx, valAddr1.Bytes(), valAddr1, valAddr2, sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, math.NewInt(500_000))))
	require.NoError(t, err)

	// update time to mature unbonding & redelegation
	unbondingTime, err := input.StakingKeeper.UnbondingTime(ctx)
	require.NoError(t, err)
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(unbondingTime))

	// mature unbonding & redelegation
	_, err = input.StakingKeeper.BlockValidatorUpdates(ctx)
	require.NoError(t, err)

	_, err = input.StakingKeeper.GetUnbondingDelegation(ctx, valAddr1.Bytes(), valAddr1)
	require.ErrorIs(t, err, collections.ErrNotFound)

	_, err = input.StakingKeeper.GetRedelegation(ctx, valAddr1.Bytes(), valAddr1, valAddr2)
	require.ErrorIs(t, err, collections.ErrNotFound)
}

type votingPowerKeeper struct {
	weights sdk.DecCoins
}

func (k *votingPowerKeeper) SetWeights(weights sdk.DecCoins) {
	k.weights = weights
}

func (k votingPowerKeeper) GetVotingPowerWeights(_ctx context.Context, _bondDenoms []string) (sdk.DecCoins, error) {
	return k.weights, nil
}

func Test_ApplyVotingPowerUpdates(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr1 := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)
	valAddr2 := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 2)

	vpk := &votingPowerKeeper{}
	input.StakingKeeper.VotingPowerKeeper = vpk
	params, err := input.StakingKeeper.GetParams(ctx)
	require.NoError(t, err)

	testDenom := testDenoms[0]
	params.BondDenoms = append(params.BondDenoms, testDenom)
	input.StakingKeeper.SetParams(ctx, params)

	bondCoins := sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(1_000_000)), sdk.NewCoin(testDenom, math.NewInt(1_000_000)))
	delAddr := input.Faucet.NewFundedAccount(ctx, bondCoins...)

	validator, err := input.StakingKeeper.Validators.Get(ctx, valAddr1)
	require.NoError(t, err)

	shares, err := input.StakingKeeper.Delegate(ctx, delAddr, bondCoins, types.Unbonded, validator, true)
	require.NoError(t, err)
	require.Equal(t, sdk.NewDecCoinsFromCoins(bondCoins...), shares)

	// set weights 1:1
	vpk.SetWeights(
		sdk.NewDecCoins(
			sdk.NewInt64DecCoin(bondDenom, 1),
			sdk.NewInt64DecCoin(testDenom, 1),
		),
	)

	// update voting power
	input.StakingKeeper.ApplyVotingPowerUpdates(ctx)

	validator1, err := input.StakingKeeper.Validators.Get(ctx, valAddr1)
	require.NoError(t, err)
	require.Equal(t, math.NewInt(3_000_000), validator1.VotingPower)
	require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin(bondDenom, 2_000_000), sdk.NewInt64Coin(testDenom, 1_000_000)), validator1.VotingPowers)

	validator2, err := input.StakingKeeper.Validators.Get(ctx, valAddr2)
	require.NoError(t, err)
	require.Equal(t, math.NewInt(2_000_000), validator2.VotingPower)
	require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin(bondDenom, 2_000_000)), validator2.VotingPowers)

	// set weight 2:1
	vpk.SetWeights(
		sdk.NewDecCoins(
			sdk.NewInt64DecCoin(bondDenom, 2),
			sdk.NewInt64DecCoin(testDenom, 1),
		),
	)

	// update voting power
	input.StakingKeeper.ApplyVotingPowerUpdates(ctx)

	validator1, err = input.StakingKeeper.Validators.Get(ctx, valAddr1)
	require.NoError(t, err)
	require.Equal(t, math.NewInt(5_000_000), validator1.VotingPower)
	require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin(bondDenom, 4_000_000), sdk.NewInt64Coin(testDenom, 1_000_000)), validator1.VotingPowers)

	validator2, err = input.StakingKeeper.Validators.Get(ctx, valAddr2)
	require.NoError(t, err)
	require.Equal(t, math.NewInt(4_000_000), validator2.VotingPower)
	require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin(bondDenom, 4_000_000)), validator2.VotingPowers)

	// increase minimum voting power
	params.MinVotingPower = 2_000_000
	require.NoError(t, input.StakingKeeper.SetParams(ctx, params))

	// back to 1:1
	vpk.SetWeights(
		sdk.NewDecCoins(
			sdk.NewInt64DecCoin(bondDenom, 1),
			sdk.NewInt64DecCoin(testDenom, 1),
		),
	)

	// make voting power smaller than minimum voting power
	_, _, err = input.StakingKeeper.Undelegate(ctx, valAddr2.Bytes(), valAddr2, sdk.NewDecCoins(sdk.NewInt64DecCoin(bondDenom, 500_000)))
	require.NoError(t, err)

	// update voting power
	require.NoError(t, input.StakingKeeper.ApplyVotingPowerUpdates(ctx))

	// validator2 should be out from whitelist
	isWhitelist, err := input.StakingKeeper.IsWhitelist(ctx, validator2)
	require.NoError(t, err)
	require.False(t, isWhitelist)
}

func Test_UnbondingToBonding(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)
	validator, err := input.StakingKeeper.Validators.Get(ctx, valAddr)
	require.NoError(t, err)

	consAddr, err := validator.GetConsAddr()
	require.NoError(t, err)

	// jail validator
	input.StakingKeeper.Jail(ctx, consAddr)

	updates, err := input.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(0), updates[0].Power)

	tmConsPubKey, err := validator.TmConsPublicKey()
	require.NoError(t, err)
	require.Equal(t, tmConsPubKey, updates[0].PubKey)

	require.True(t, input.BankKeeper.GetBalance(ctx, input.StakingKeeper.GetBondedPool(ctx).GetAddress(), bondDenom).IsZero())
	require.Equal(t, sdk.NewInt64Coin(bondDenom, 1_000_000), input.BankKeeper.GetBalance(ctx, input.StakingKeeper.GetNotBondedPool(ctx).GetAddress(), bondDenom))

	// unjail validator
	require.NoError(t, input.StakingKeeper.Unjail(ctx, consAddr))

	updates, err = input.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), updates[0].Power)

	tmConsPubKey, err = validator.TmConsPublicKey()
	require.NoError(t, err)
	require.Equal(t, tmConsPubKey, updates[0].PubKey)

	require.True(t, input.BankKeeper.GetBalance(ctx, input.StakingKeeper.GetNotBondedPool(ctx).GetAddress(), bondDenom).IsZero())
	require.Equal(t, sdk.NewInt64Coin(bondDenom, 1_000_000), input.BankKeeper.GetBalance(ctx, input.StakingKeeper.GetBondedPool(ctx).GetAddress(), bondDenom))
}
