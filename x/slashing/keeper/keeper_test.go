package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/core/comet"
	"cosmossdk.io/math"
	staking "github.com/initia-labs/initia/x/mstaking"
	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestUnJailNotBonded(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	p, err := input.StakingKeeper.GetParams(ctx)
	require.NoError(t, err)

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

	_, err = input.StakingKeeper.GetValidator(ctx, valAddr6)
	require.NoError(t, err)

	// unbond below minimum self-delegation
	_, _, err = input.StakingKeeper.Undelegate(ctx, valAddr6.Bytes(), valAddr6, sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, math.NewInt(5_000_000))))
	require.NoError(t, err)

	staking.EndBlocker(ctx, input.StakingKeeper)
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// verify that validator is jailed and removed from whitelist
	_, err = input.StakingKeeper.GetValidator(ctx, valAddr6)
	require.Error(t, err)

	// verify we cannot unjail
	require.Error(t, input.SlashingKeeper.Unjail(ctx, valAddr6))

	staking.EndBlocker(ctx, input.StakingKeeper)
}

// Test a new validator entering the validator set
// Ensure that SigningInfo.StartHeight is set correctly
// and that they are not immediately jailed
func TestHandleNewValidator(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	signedBlock, err := input.SlashingKeeper.SignedBlocksWindow(ctx)
	require.NoError(t, err)
	ctx = ctx.WithBlockHeight(signedBlock + 1)

	valAddr, valPubKey := createValidatorWithBalanceAndGetPk(ctx, input, 100_000_000, 10_000_000, 1)
	validator, err := input.StakingKeeper.GetValidator(ctx, valAddr)
	require.NoError(t, err)

	staking.EndBlocker(ctx, input.StakingKeeper)

	require.Equal(
		t, input.BankKeeper.GetAllBalances(ctx, sdk.AccAddress(valAddr)),
		sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(90_000_000))),
	)
	val, err := input.StakingKeeper.Validator(ctx, valAddr)
	require.NoError(t, err)
	require.Equal(t, math.NewInt(10_000_000), val.GetBondedTokens().AmountOf(bondDenom))

	// Now a validator, for two blocks
	err = input.SlashingKeeper.HandleValidatorSignature(ctx, valPubKey.Address(), 10, comet.BlockIDFlagCommit)
	require.NoError(t, err)
	ctx = ctx.WithBlockHeight(signedBlock + 2)
	err = input.SlashingKeeper.HandleValidatorSignature(ctx, valPubKey.Address(), 10, comet.BlockIDFlagAbsent)
	require.NoError(t, err)

	valConsPk, err := validator.ConsPubKey()
	require.NoError(t, err)

	info, err := input.SlashingKeeper.GetValidatorSigningInfo(ctx, sdk.ConsAddress(valConsPk.Address()))
	require.NoError(t, err)
	require.Equal(t, signedBlock+1, info.StartHeight)
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

	p, err := input.SlashingKeeper.GetParams(ctx)
	require.NoError(t, err)
	p.SignedBlocksWindow = 1000
	input.SlashingKeeper.SetParams(ctx, p)

	signedBlock, err := input.SlashingKeeper.SignedBlocksWindow(ctx)
	require.NoError(t, err)
	ctx = ctx.WithBlockHeight(1)

	power := int64(100)
	amt := math.NewInt(100_000_000)
	_, valPubKey := createValidatorWithBalanceAndGetPk(ctx, input, 100_000_000, 100_000_000, 1)

	staking.EndBlocker(ctx, input.StakingKeeper)

	// 1000 first blocks OK
	height := int64(0)
	for ; height < signedBlock; height++ {
		ctx = ctx.WithBlockHeight(height)
		input.SlashingKeeper.HandleValidatorSignature(ctx, valPubKey.Address(), power, comet.BlockIDFlagCommit)
	}

	minSignedPerWindow, err := input.SlashingKeeper.MinSignedPerWindow(ctx)
	require.NoError(t, err)

	// 501 blocks missed
	for ; height < signedBlock+(signedBlock-minSignedPerWindow)+1; height++ {
		ctx = ctx.WithBlockHeight(height)
		input.SlashingKeeper.HandleValidatorSignature(ctx, valPubKey.Address(), power, comet.BlockIDFlagAbsent)
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
	input.SlashingKeeper.HandleValidatorSignature(ctx, valPubKey.Address(), power, comet.BlockIDFlagAbsent)

	// validator should not have been slashed twice
	validator, _ = input.StakingKeeper.GetValidatorByConsAddr(ctx, sdk.GetConsAddress(valPubKey))
	require.Equal(t, resultingTokens, validator.GetTokens().AmountOf(bondDenom))
}

// Test a validator dipping in and out of the validator set
// Ensure that missed blocks are tracked correctly and that
// the start height of the signing info is reset correctly
func TestValidatorDippingInAndOut(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	ctx = ctx.WithBlockHeight(1)

	stakingParams, err := input.StakingKeeper.GetParams(ctx)
	require.NoError(t, err)

	stakingParams.MaxValidators = 1
	input.StakingKeeper.SetParams(ctx, stakingParams)

	slashingParams, err := input.SlashingKeeper.GetParams(ctx)
	require.NoError(t, err)
	slashingParams.SignedBlocksWindow = 1000
	input.SlashingKeeper.SetParams(ctx, slashingParams)

	signedBlock, err := input.SlashingKeeper.SignedBlocksWindow(ctx)
	require.NoError(t, err)

	power := int64(100)

	valAddr1, valPubKey1 := createValidatorWithBalanceAndGetPk(ctx, input, 200_000_000, 100_000_000, 1)
	validator1, err := input.StakingKeeper.GetValidator(ctx, valAddr1)
	require.NoError(t, err)
	ok, err := input.StakingKeeper.IsWhitelist(ctx, validator1)
	require.NoError(t, err)
	require.True(t, ok)

	staking.EndBlocker(ctx, input.StakingKeeper)

	// 100 first blocks OK
	height := int64(0)
	for ; height < int64(100); height++ {
		ctx = ctx.WithBlockHeight(height)
		input.SlashingKeeper.HandleValidatorSignature(ctx, valPubKey1.Address(), power, comet.BlockIDFlagCommit)
	}

	// kick first validator out of validator set
	createValidatorWithBalance(ctx, input, 200_000_000, 101_000_000, 2)
	staking.EndBlocker(ctx, input.StakingKeeper)

	validator1, err = input.StakingKeeper.GetValidator(ctx, valAddr1)
	require.NoError(t, err)
	require.Equal(t, stakingtypes.Unbonding, validator1.GetStatus())

	// 600 more blocks happened
	height = height + 600
	ctx = ctx.WithBlockHeight(height)

	// validator added back in
	valAddr3 := createValidatorWithBalance(ctx, input, 200_000_000, 50_000_000, 3)
	bondCoins := sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(50_000_000)))
	input.StakingKeeper.Delegate(ctx, valAddr3.Bytes(), bondCoins, stakingtypes.Unbonded, validator1, true)
	staking.EndBlocker(ctx, input.StakingKeeper)

	validator1, _ = input.StakingKeeper.GetValidator(ctx, valAddr1)
	require.Equal(t, stakingtypes.Bonded, validator1.GetStatus())

	newPower := power + 50

	// validator misses a block
	input.SlashingKeeper.HandleValidatorSignature(ctx, valPubKey1.Address(), newPower, comet.BlockIDFlagAbsent)
	height++

	// shouldn't be jailed/kicked yet
	validator1, _ = input.StakingKeeper.GetValidator(ctx, valAddr1)
	require.Equal(t, stakingtypes.Bonded, validator1.GetStatus())

	// validator misses an additional 500 more blocks within the SignedBlockWindow (here 1000 blocks).
	latest := signedBlock + height
	// misses 500 blocks + within the signing windows i.e. 700-1700
	// validators misses all 1000 block of a SignedBlockWindows
	for ; height < latest+1; height++ {
		ctx = ctx.WithBlockHeight(height)
		input.SlashingKeeper.HandleValidatorSignature(ctx, valPubKey1.Address(), newPower, comet.BlockIDFlagAbsent)
	}

	// should now be jailed & kicked
	staking.EndBlocker(ctx, input.StakingKeeper)
	validator1, _ = input.StakingKeeper.GetValidator(ctx, valAddr1)
	require.Equal(t, stakingtypes.Unbonding, validator1.GetStatus())
	require.True(t, validator1.IsJailed())

	// array should be cleared
	for offset := int64(0); offset < signedBlock; offset++ {
		missed, err := input.SlashingKeeper.GetMissedBlockBitmapValue(ctx, sdk.ConsAddress(valPubKey1.Address()), offset)
		require.NoError(t, err)
		require.False(t, missed)
	}

	// some blocks pass
	height = int64(5000)
	ctx = ctx.WithBlockHeight(height)

	// // validator rejoins and starts signing again
	input.StakingKeeper.Unjail(ctx, sdk.ConsAddress(valPubKey1.Address()))
	input.SlashingKeeper.HandleValidatorSignature(ctx, valPubKey1.Address(), newPower, comet.BlockIDFlagCommit)
	height++

	// validator should not be kicked since we reset counter/array when it was jailed
	staking.EndBlocker(ctx, input.StakingKeeper)
	validator1, _ = input.StakingKeeper.GetValidator(ctx, valAddr1)
	require.Equal(t, stakingtypes.Bonded, validator1.GetStatus())

	// validator misses 501 blocks after SignedBlockWindow period (1000 blocks)
	latest = signedBlock + height
	for ; height < latest+1; height++ {
		ctx = ctx.WithBlockHeight(height)
		input.SlashingKeeper.HandleValidatorSignature(ctx, valPubKey1.Address(), newPower, comet.BlockIDFlagAbsent)
	}

	// validator should now be jailed & kicked
	staking.EndBlocker(ctx, input.StakingKeeper)
	validator1, _ = input.StakingKeeper.GetValidator(ctx, valAddr1)
	require.Equal(t, stakingtypes.Unbonding, validator1.GetStatus())
}
