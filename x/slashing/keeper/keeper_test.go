package keeper_test

import (
	"testing"
	"time"

	staking "github.com/initia-labs/initia/x/mstaking"
	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestUnJailNotBonded(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	p := input.StakingKeeper.GetParams(ctx)
	p.MaxValidators = 5
	input.StakingKeeper.SetParams(ctx, p)

	createValidatorWithBalance(ctx, input, 100_000_000, 10_000_000, 1)
	createValidatorWithBalance(ctx, input, 100_000_000, 10_000_000, 2)
	createValidatorWithBalance(ctx, input, 100_000_000, 10_000_000, 3)
	createValidatorWithBalance(ctx, input, 100_000_000, 10_000_000, 4)
	createValidatorWithBalance(ctx, input, 100_000_000, 10_000_000, 5)

	staking.EndBlocker(ctx, input.StakingKeeper)
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	valAddr6 := createValidatorWithBalance(ctx, input, 10_000_000, 5_000_000, 6)

	staking.EndBlocker(ctx, input.StakingKeeper)
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	_, found := input.StakingKeeper.GetValidator(ctx, valAddr6)
	require.True(t, found)

	// unbond below minimum self-delegation
	_, err := input.StakingKeeper.Undelegate(ctx, valAddr6.Bytes(), valAddr6, sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, sdk.NewInt(5_000_000))))
	require.NoError(t, err)

	staking.EndBlocker(ctx, input.StakingKeeper)
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// verify that validator is jailed and removed from whitelist
	_, found = input.StakingKeeper.GetValidator(ctx, valAddr6)
	require.False(t, found)

	// verify we cannot unjail
	require.Error(t, input.SlashingKeeper.Unjail(ctx, valAddr6))

	staking.EndBlocker(ctx, input.StakingKeeper)
}

// Test a new validator entering the validator set
// Ensure that SigningInfo.StartHeight is set correctly
// and that they are not immediately jailed
func TestHandleNewValidator(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	ctx = ctx.WithBlockHeight(input.SlashingKeeper.SignedBlocksWindow(ctx) + 1)

	valAddr, valPubKey := createValidatorWithBalanceAndGetPk(ctx, input, 100_000_000, 10_000_000, 1)
	validator, found := input.StakingKeeper.GetValidator(ctx, valAddr)
	require.True(t, found)

	staking.EndBlocker(ctx, input.StakingKeeper)

	require.Equal(
		t, input.BankKeeper.GetAllBalances(ctx, sdk.AccAddress(valAddr)),
		sdk.NewCoins(sdk.NewCoin(bondDenom, sdk.NewInt(90_000_000))),
	)
	require.Equal(t, sdk.NewInt(10_000_000), input.StakingKeeper.Validator(ctx, valAddr).GetBondedTokens().AmountOf(bondDenom))

	// Now a validator, for two blocks
	input.SlashingKeeper.HandleValidatorSignature(ctx, valPubKey.Address(), 10, true)
	ctx = ctx.WithBlockHeight(input.SlashingKeeper.SignedBlocksWindow(ctx) + 2)
	input.SlashingKeeper.HandleValidatorSignature(ctx, valPubKey.Address(), 10, false)

	valConsPk, err := validator.ConsPubKey()
	require.NoError(t, err)

	info, found := input.SlashingKeeper.GetValidatorSigningInfo(ctx, sdk.ConsAddress(valConsPk.Address()))
	require.True(t, found)
	require.Equal(t, input.SlashingKeeper.SignedBlocksWindow(ctx)+1, info.StartHeight)
	require.Equal(t, int64(2), info.IndexOffset)
	require.Equal(t, int64(1), info.MissedBlocksCounter)
	require.Equal(t, time.Unix(0, 0).UTC(), info.JailedUntil)

	// validator should be bonded still, should not have been jailed or slashed
	validator, _ = input.StakingKeeper.GetValidator(ctx, valAddr)
	require.Equal(t, stakingtypes.Bonded, validator.GetStatus())
	bondPool := input.StakingKeeper.GetBondedPool(ctx)
	expTokens := input.StakingKeeper.VotingPowerFromConsensusPower(ctx, 10)
	require.True(t, expTokens.Equal(input.BankKeeper.GetBalance(ctx, bondPool.GetAddress(), bondDenom).Amount))
}

// Test a jailed validator being "down" twice
// Ensure that they're only slashed once
func TestHandleAlreadyJailed(t *testing.T) {
	// initial setup
	ctx, input := createDefaultTestInput(t)
	ctx = ctx.WithBlockHeight(input.SlashingKeeper.SignedBlocksWindow(ctx) + 1)

	p := input.SlashingKeeper.GetParams(ctx)
	p.SignedBlocksWindow = 1000
	input.SlashingKeeper.SetParams(ctx, p)

	power := int64(100)
	amt := sdk.NewInt(100_000_000)
	_, valPubKey := createValidatorWithBalanceAndGetPk(ctx, input, 100_000_000, 100_000_000, 1)

	staking.EndBlocker(ctx, input.StakingKeeper)

	// 1000 first blocks OK
	height := int64(0)
	for ; height < input.SlashingKeeper.SignedBlocksWindow(ctx); height++ {
		ctx = ctx.WithBlockHeight(height)
		input.SlashingKeeper.HandleValidatorSignature(ctx, valPubKey.Address(), power, true)
	}
	// 501 blocks missed
	for ; height < input.SlashingKeeper.SignedBlocksWindow(ctx)+(input.SlashingKeeper.SignedBlocksWindow(ctx)-input.SlashingKeeper.MinSignedPerWindow(ctx))+1; height++ {
		ctx = ctx.WithBlockHeight(height)
		input.SlashingKeeper.HandleValidatorSignature(ctx, valPubKey.Address(), power, false)
	}

	// end block
	staking.EndBlocker(ctx, input.StakingKeeper)

	// validator should have been jailed and slashed
	validator, _ := input.StakingKeeper.GetValidatorByConsAddr(ctx, sdk.GetConsAddress(valPubKey))
	require.Equal(t, stakingtypes.Unbonding, validator.GetStatus())

	// validator should have been slashed
	resultingTokens := amt.Sub(input.StakingKeeper.VotingPowerFromConsensusPower(ctx, 1))
	require.Equal(t, resultingTokens, validator.GetTokens().AmountOf(bondDenom))

	// another block missed
	ctx = ctx.WithBlockHeight(height)
	input.SlashingKeeper.HandleValidatorSignature(ctx, valPubKey.Address(), power, false)

	// validator should not have been slashed twice
	validator, _ = input.StakingKeeper.GetValidatorByConsAddr(ctx, sdk.GetConsAddress(valPubKey))
	require.Equal(t, resultingTokens, validator.GetTokens().AmountOf(bondDenom))
}

// Test a validator dipping in and out of the validator set
// Ensure that missed blocks are tracked correctly and that
// the start height of the signing info is reset correctly
func TestValidatorDippingInAndOut(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	ctx = ctx.WithBlockHeight(input.SlashingKeeper.SignedBlocksWindow(ctx) + 1)

	stakingParams := input.StakingKeeper.GetParams(ctx)
	stakingParams.MaxValidators = 1
	input.StakingKeeper.SetParams(ctx, stakingParams)

	slashingParams := input.SlashingKeeper.GetParams(ctx)
	slashingParams.SignedBlocksWindow = 1000
	input.SlashingKeeper.SetParams(ctx, slashingParams)

	power := int64(100)

	valAddr1, valPubKey1 := createValidatorWithBalanceAndGetPk(ctx, input, 200_000_000, 100_000_000, 1)
	validator1, found := input.StakingKeeper.GetValidator(ctx, valAddr1)
	require.True(t, found)
	require.True(t, input.StakingKeeper.IsWhitelist(ctx, validator1))

	staking.EndBlocker(ctx, input.StakingKeeper)

	// 100 first blocks OK
	height := int64(0)
	for ; height < int64(100); height++ {
		ctx = ctx.WithBlockHeight(height)
		input.SlashingKeeper.HandleValidatorSignature(ctx, valPubKey1.Address(), power, true)
	}

	// kick first validator out of validator set
	createValidatorWithBalance(ctx, input, 200_000_000, 101_000_000, 2)
	staking.EndBlocker(ctx, input.StakingKeeper)

	validator1, found = input.StakingKeeper.GetValidator(ctx, valAddr1)
	require.True(t, found)
	require.Equal(t, stakingtypes.Unbonding, validator1.GetStatus())

	// 600 more blocks happened
	height = height + 600
	ctx = ctx.WithBlockHeight(height)

	// validator added back in
	valAddr3 := createValidatorWithBalance(ctx, input, 200_000_000, 50_000_000, 3)
	bondCoins := sdk.NewCoins(sdk.NewCoin(bondDenom, sdk.NewInt(50_000_000)))
	require.True(t, found)
	input.StakingKeeper.Delegate(ctx, valAddr3.Bytes(), bondCoins, stakingtypes.Unbonded, validator1, true)
	staking.EndBlocker(ctx, input.StakingKeeper)

	validator1, _ = input.StakingKeeper.GetValidator(ctx, valAddr1)
	require.Equal(t, stakingtypes.Bonded, validator1.GetStatus())

	newPower := power + 50

	// validator misses a block
	input.SlashingKeeper.HandleValidatorSignature(ctx, valPubKey1.Address(), newPower, false)
	height++

	// shouldn't be jailed/kicked yet
	validator1, _ = input.StakingKeeper.GetValidator(ctx, valAddr1)
	require.Equal(t, stakingtypes.Bonded, validator1.GetStatus())

	// validator misses an additional 500 more blocks within the SignedBlockWindow (here 1000 blocks).
	latest := input.SlashingKeeper.SignedBlocksWindow(ctx) + height
	// misses 500 blocks + within the signing windows i.e. 700-1700
	// validators misses all 1000 block of a SignedBlockWindows
	for ; height < latest+1; height++ {
		ctx = ctx.WithBlockHeight(height)
		input.SlashingKeeper.HandleValidatorSignature(ctx, valPubKey1.Address(), newPower, false)
	}

	// should now be jailed & kicked
	staking.EndBlocker(ctx, input.StakingKeeper)
	validator1, _ = input.StakingKeeper.GetValidator(ctx, valAddr1)
	require.Equal(t, stakingtypes.Unbonding, validator1.GetStatus())
	require.True(t, validator1.IsJailed())

	// array should be cleared
	for offset := int64(0); offset < input.SlashingKeeper.SignedBlocksWindow(ctx); offset++ {
		missed := input.SlashingKeeper.GetValidatorMissedBlockBitArray(ctx, sdk.ConsAddress(valPubKey1.Address()), offset)
		require.False(t, missed)
	}

	// some blocks pass
	height = int64(5000)
	ctx = ctx.WithBlockHeight(height)

	// // validator rejoins and starts signing again
	input.StakingKeeper.Unjail(ctx, sdk.ConsAddress(valPubKey1.Address()))
	input.SlashingKeeper.HandleValidatorSignature(ctx, valPubKey1.Address(), newPower, true)
	height++

	// validator should not be kicked since we reset counter/array when it was jailed
	staking.EndBlocker(ctx, input.StakingKeeper)
	validator1, _ = input.StakingKeeper.GetValidator(ctx, valAddr1)
	require.Equal(t, stakingtypes.Bonded, validator1.GetStatus())

	// validator misses 501 blocks after SignedBlockWindow period (1000 blocks)
	latest = input.SlashingKeeper.SignedBlocksWindow(ctx) + height
	for ; height < latest+1; height++ {
		ctx = ctx.WithBlockHeight(height)
		input.SlashingKeeper.HandleValidatorSignature(ctx, valPubKey1.Address(), newPower, false)
	}

	// validator should now be jailed & kicked
	staking.EndBlocker(ctx, input.StakingKeeper)
	validator1, _ = input.StakingKeeper.GetValidator(ctx, valAddr1)
	require.Equal(t, stakingtypes.Unbonding, validator1.GetStatus())
}
