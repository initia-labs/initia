package keeper_test

import (
	"testing"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	customtypes "github.com/initia-labs/initia/x/distribution/types"
	staking "github.com/initia-labs/initia/x/mstaking"
	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"
	"github.com/stretchr/testify/require"
)

func TestCalculateRewardsBasic(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	valAddr1 := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)
	_, found := input.StakingKeeper.GetValidator(ctx, valAddr1)
	require.True(t, found)

	// historical count should be 2 (once for validator init, once for delegation init)
	require.Equal(t, uint64(2), input.DistKeeper.GetValidatorHistoricalReferenceCount(ctx))

	// end block to bond validator and start new block
	staking.EndBlocker(ctx, input.StakingKeeper)
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// fetch validator and delegation
	val := input.StakingKeeper.Validator(ctx, valAddr1)
	del := input.StakingKeeper.Delegation(ctx, sdk.AccAddress(valAddr1), valAddr1)

	// end period
	endingPeriod := input.DistKeeper.IncrementValidatorPeriod(ctx, val)

	// historical count should be 2 still
	require.Equal(t, uint64(2), input.DistKeeper.GetValidatorHistoricalReferenceCount(ctx))

	// calculate delegation rewards
	rewards := input.DistKeeper.CalculateDelegationRewards(ctx, val, del, endingPeriod)

	// rewards should be zero
	require.True(t, rewards.Sum().IsZero())

	// allocate some rewards
	initial := int64(10)
	tokens := sdk.DecCoins{{Denom: bondDenom, Amount: sdk.NewDec(initial)}}
	input.DistKeeper.AllocateTokensToValidatorPool(ctx, val, bondDenom, tokens)

	// end period
	endingPeriod = input.DistKeeper.IncrementValidatorPeriod(ctx, val)

	// calculate delegation rewards
	rewards = input.DistKeeper.CalculateDelegationRewards(ctx, val, del, endingPeriod)

	// rewards should be half the tokens
	require.Equal(t,
		customtypes.DecPools{{Denom: bondDenom, DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: sdk.NewDec(initial / 2)}}}},
		rewards)

	// commission should be the other half
	require.Equal(t,
		customtypes.DecPools{{Denom: bondDenom, DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: sdk.NewDec(initial / 2)}}}},
		input.DistKeeper.GetValidatorAccumulatedCommission(ctx, valAddr1).Commissions)
}

func TestCalculateRewardsAfterSlash(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr1 := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)
	validator1, found := input.StakingKeeper.GetValidator(ctx, valAddr1)
	require.True(t, found)

	// end block to bond validator and start new block
	staking.EndBlocker(ctx, input.StakingKeeper)
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// fetch validator and delegation
	val := input.StakingKeeper.Validator(ctx, valAddr1)
	del := input.StakingKeeper.Delegation(ctx, sdk.AccAddress(valAddr1), valAddr1)

	// end period
	endingPeriod := input.DistKeeper.IncrementValidatorPeriod(ctx, val)
	// calculate delegation rewards
	rewards := input.DistKeeper.CalculateDelegationRewards(ctx, val, del, endingPeriod)
	// rewards should be zero
	require.True(t, rewards.Sum().IsZero())

	pubkey, err := validator1.ConsPubKey()
	require.NoError(t, err)

	// update validator for voting power update
	_, err = input.StakingKeeper.ApplyAndReturnValidatorSetUpdates(ctx)
	require.NoError(t, err)
	power := validator1.ConsensusPower(input.StakingKeeper.PowerReduction(ctx))
	require.Equal(t, int64(1), power)

	// slash the validator by 50%
	input.StakingKeeper.Slash(ctx, pubkey.Address().Bytes(), ctx.BlockHeight(), sdk.NewDecWithPrec(5, 1))

	// retrieve validator
	val = input.StakingKeeper.Validator(ctx, valAddr1)

	// allocate some rewards
	initial := input.StakingKeeper.VotingPowerFromConsensusPower(ctx, 10)
	tokens := sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDecFromInt(initial)}}
	input.DistKeeper.AllocateTokensToValidatorPool(ctx, val, bondDenom, tokens)

	// end period
	endingPeriod = input.DistKeeper.IncrementValidatorPeriod(ctx, val)

	// calculate delegation rewards
	rewards = input.DistKeeper.CalculateDelegationRewards(ctx, val, del, endingPeriod)

	// rewards should be half the tokens
	require.Equal(t, customtypes.DecPools{{Denom: bondDenom, DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDecFromInt(initial.QuoRaw(2))}}}}, rewards)
	// commission should be the other half
	require.Equal(t, customtypes.DecPools{{Denom: bondDenom, DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDecFromInt(initial.QuoRaw(2))}}}}, input.DistKeeper.GetValidatorAccumulatedCommission(ctx, valAddr1).Commissions)
}

func TestCalculateRewardsAfterManySlashes(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	valAddr1 := createValidatorWithBalance(ctx, input, 100_000_000, 100_000_000, 1)
	_, found := input.StakingKeeper.GetValidator(ctx, valAddr1)
	require.True(t, found)

	validator1, found := input.StakingKeeper.GetValidator(ctx, valAddr1)
	require.True(t, found)

	pubkey, err := validator1.ConsPubKey()
	require.NoError(t, err)
	valConsAddr1 := pubkey.Address().Bytes()

	// end block to bond validator
	staking.EndBlocker(ctx, input.StakingKeeper)

	// next block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// fetch validator and delegation
	val := input.StakingKeeper.Validator(ctx, valAddr1)
	del := input.StakingKeeper.Delegation(ctx, sdk.AccAddress(valAddr1), valAddr1)

	// end period
	endingPeriod := input.DistKeeper.IncrementValidatorPeriod(ctx, val)

	// calculate delegation rewards
	rewards := input.DistKeeper.CalculateDelegationRewards(ctx, val, del, endingPeriod)

	// rewards should be zero
	require.True(t, rewards.Sum().IsZero())

	// start out block height
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 3)

	// slash the validator by 50%
	input.StakingKeeper.Slash(ctx, valConsAddr1, ctx.BlockHeight(), sdk.NewDecWithPrec(5, 1))

	// fetch the validator again
	val = input.StakingKeeper.Validator(ctx, valAddr1)

	// increase block height
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 3)

	// allocate some rewards
	initial := input.StakingKeeper.VotingPowerFromConsensusPower(ctx, 10)
	tokens := sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDecFromInt(initial)}}
	input.DistKeeper.AllocateTokensToValidatorPool(ctx, val, bondDenom, tokens)

	// slash the validator by 50% again
	input.StakingKeeper.Slash(ctx, valConsAddr1, ctx.BlockHeight(), sdk.NewDecWithPrec(5, 1))

	// fetch the validator again
	val = input.StakingKeeper.Validator(ctx, valAddr1)

	// increase block height
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 3)

	// allocate some more rewards
	input.DistKeeper.AllocateTokensToValidatorPool(ctx, val, bondDenom, tokens)

	// end period
	endingPeriod = input.DistKeeper.IncrementValidatorPeriod(ctx, val)

	// calculate delegation rewards
	rewards = input.DistKeeper.CalculateDelegationRewards(ctx, val, del, endingPeriod)

	// rewards should be half the tokens
	require.Equal(t,
		customtypes.DecPools{{Denom: bondDenom, DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDecFromInt(initial)}}}},
		rewards)

	// commission should be the other half
	require.Equal(t,
		customtypes.DecPools{{Denom: bondDenom, DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: math.LegacyNewDecFromInt(initial)}}}},
		input.DistKeeper.GetValidatorAccumulatedCommission(ctx, valAddr1).Commissions)
}

func TestCalculateRewardsMultiDelegator(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	// self-delegation
	valAddr1 := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)
	validator, found := input.StakingKeeper.GetValidator(ctx, valAddr1)
	require.True(t, found)
	del1, found := input.StakingKeeper.GetDelegation(ctx, sdk.AccAddress(valAddr1), valAddr1)
	require.True(t, found)

	// fetch validator and delegation
	val := input.StakingKeeper.Validator(ctx, valAddr1)

	// allocate some rewards
	initial := int64(1000)
	tokens := sdk.DecCoins{{Denom: bondDenom, Amount: sdk.NewDec(initial)}}
	input.DistKeeper.AllocateTokensToValidatorPool(ctx, val, bondDenom, tokens)

	// delegate to validator
	bondCoins := sdk.NewCoins(sdk.NewCoin(bondDenom, sdk.NewInt(1_000_000)))
	delAddr := input.Faucet.NewFundedAccount(ctx, bondCoins...)
	require.True(t, found)
	shares, err := input.StakingKeeper.Delegate(ctx, delAddr, bondCoins, stakingtypes.Unbonded, validator, true)
	require.NoError(t, err)
	require.Equal(t, sdk.NewDecCoinsFromCoins(bondCoins...), shares)
	del2, found := input.StakingKeeper.GetDelegation(ctx, delAddr, valAddr1)
	require.True(t, found)

	// fetch validator and delegation
	val = input.StakingKeeper.Validator(ctx, valAddr1)

	// end block
	staking.EndBlocker(ctx, input.StakingKeeper)
	// next block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// allocate some more rewards
	input.DistKeeper.AllocateTokensToValidatorPool(ctx, val, bondDenom, tokens)

	// end period
	endingPeriod := input.DistKeeper.IncrementValidatorPeriod(ctx, val)

	// calculate delegation rewards for del1
	rewards := input.DistKeeper.CalculateDelegationRewards(ctx, val, del1, endingPeriod)

	require.Equal(t,
		customtypes.DecPools{{Denom: bondDenom, DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: sdk.NewDec(initial * 3 / 4)}}}},
		rewards)

	// calculate delegation rewards for del2
	rewards = input.DistKeeper.CalculateDelegationRewards(ctx, val, del2, endingPeriod)

	require.Equal(t,
		customtypes.DecPools{{Denom: bondDenom, DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: sdk.NewDec(initial * 1 / 4)}}}},
		rewards)

	// commission should be equal to initial (50% twice)
	require.Equal(t,
		customtypes.DecPools{{Denom: bondDenom, DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: sdk.NewDec(initial)}}}},
		input.DistKeeper.GetValidatorAccumulatedCommission(ctx, valAddr1).Commissions)
}

func TestWithdrawDelegationRewardsBasic(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	// create validator with 50% commission
	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)
	_, found := input.StakingKeeper.GetValidator(ctx, valAddr)
	require.True(t, found)

	balancePower := int64(100)
	balanceTokens := input.StakingKeeper.VotingPowerFromConsensusPower(ctx, balancePower)

	// set module account coins
	distrAcc := input.DistKeeper.GetDistributionAccount(ctx)
	amount := sdk.NewCoins(sdk.NewCoin(bondDenom, balanceTokens))
	err := input.BankKeeper.MintCoins(ctx, authtypes.Minter, amount)
	require.NoError(t, err)
	err = input.BankKeeper.SendCoinsFromModuleToModule(ctx, authtypes.Minter, distrAcc.GetName(), amount)
	require.NoError(t, err)

	power := int64(1)
	valTokens := input.StakingKeeper.VotingPowerFromConsensusPower(ctx, power)

	// assert correct initial balance
	expTokens := balanceTokens.Sub(valTokens)
	require.Equal(t,
		sdk.Coins{sdk.NewCoin(bondDenom, expTokens)},
		input.BankKeeper.GetAllBalances(ctx, sdk.AccAddress(valAddr)),
	)

	// end block to bond validator
	staking.EndBlocker(ctx, input.StakingKeeper)
	// next block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// fetch validator and delegation
	val := input.StakingKeeper.Validator(ctx, valAddr)

	// allocate some rewards
	initial := input.StakingKeeper.VotingPowerFromConsensusPower(ctx, 1)
	tokens := sdk.DecCoins{sdk.NewDecCoin(bondDenom, initial)}

	input.DistKeeper.AllocateTokensToValidatorPool(ctx, val, bondDenom, tokens)

	// historical count should be 2 (initial + latest for delegation)
	require.Equal(t, uint64(2), input.DistKeeper.GetValidatorHistoricalReferenceCount(ctx))

	// withdraw rewards
	_, err = input.DistKeeper.WithdrawDelegationRewards(ctx, sdk.AccAddress(valAddr), valAddr)
	require.Nil(t, err)

	// historical count should still be 2 (added one record, cleared one)
	require.Equal(t, uint64(2), input.DistKeeper.GetValidatorHistoricalReferenceCount(ctx))

	// assert correct balance
	exp := balanceTokens.Sub(valTokens).Add(initial.QuoRaw(2))
	require.Equal(t,
		sdk.Coins{sdk.NewCoin(bondDenom, exp)},
		input.BankKeeper.GetAllBalances(ctx, sdk.AccAddress(valAddr)),
	)

	// withdraw commission
	_, err = input.DistKeeper.WithdrawValidatorCommission(ctx, valAddr)
	require.Nil(t, err)
}

func TestWithdrawDelegationZeroRewards(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	// create validator with 50% commission
	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)
	_, found := input.StakingKeeper.GetValidator(ctx, valAddr)
	require.True(t, found)

	balancePower := int64(1000)
	balanceTokens := input.StakingKeeper.VotingPowerFromConsensusPower(ctx, balancePower)

	// set module account coins
	distrAcc := input.DistKeeper.GetDistributionAccount(ctx)
	amount := sdk.NewCoins(sdk.NewCoin(bondDenom, balanceTokens))
	err := input.BankKeeper.MintCoins(ctx, authtypes.Minter, amount)
	require.NoError(t, err)
	err = input.BankKeeper.SendCoinsFromModuleToModule(ctx, authtypes.Minter, distrAcc.GetName(), amount)
	require.NoError(t, err)
	input.AccountKeeper.SetModuleAccount(ctx, distrAcc)

	// withdraw rewards -- should be 0
	pool, err := input.DistKeeper.WithdrawDelegationRewards(ctx, sdk.AccAddress(valAddr), valAddr)
	require.NoError(t, err)
	require.True(t, pool.Sum().IsZero(), "expected withdraw rewards to be zero")
	require.True(t, pool.Sum().IsValid(), "expected returned coins to be valid")
}

func TestCalculateRewardsAfterManySlashesInSameBlock(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 10_000_000, 1)
	validator, found := input.StakingKeeper.GetValidator(ctx, valAddr)
	require.True(t, found)

	// end block to bond validator
	staking.EndBlocker(ctx, input.StakingKeeper)
	// next block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// fetch validator and delegation
	val := input.StakingKeeper.Validator(ctx, valAddr)
	del := input.StakingKeeper.Delegation(ctx, sdk.AccAddress(valAddr), valAddr)

	// end period
	endingPeriod := input.DistKeeper.IncrementValidatorPeriod(ctx, val)
	// calculate delegation rewards
	rewards := input.DistKeeper.CalculateDelegationRewards(ctx, val, del, endingPeriod)

	// rewards should be zero
	require.True(t, rewards.Sum().IsZero())
	// start out block height
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 3)

	// allocate some rewards
	initial := math.LegacyNewDecFromInt(input.StakingKeeper.VotingPowerFromConsensusPower(ctx, 1))
	tokens := sdk.DecCoins{{Denom: bondDenom, Amount: initial}}
	input.DistKeeper.AllocateTokensToValidatorPool(ctx, val, bondDenom, tokens)

	pubkey, err := validator.ConsPubKey()
	require.NoError(t, err)
	valConsAddr := pubkey.Address().Bytes()

	// slash the validator by 50%
	input.StakingKeeper.Slash(ctx, valConsAddr, ctx.BlockHeight(), sdk.NewDecWithPrec(5, 1))

	// slash the validator by 50% again
	input.StakingKeeper.Slash(ctx, valConsAddr, ctx.BlockHeight(), sdk.NewDecWithPrec(5, 1))

	// fetch the validator again
	val = input.StakingKeeper.Validator(ctx, valAddr)
	// increase block height
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 3)

	// allocate some more rewards
	input.DistKeeper.AllocateTokensToValidatorPool(ctx, val, bondDenom, tokens)

	// end period
	endingPeriod = input.DistKeeper.IncrementValidatorPeriod(ctx, val)

	// calculate delegation rewards
	rewards = input.DistKeeper.CalculateDelegationRewards(ctx, val, del, endingPeriod)

	// rewards should be half the tokens
	require.Equal(t,
		customtypes.DecPools{{Denom: bondDenom, DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: initial}}}},
		rewards)

	// commission should be the other half
	require.Equal(t,
		customtypes.DecPools{{Denom: bondDenom, DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: initial}}}},
		input.DistKeeper.GetValidatorAccumulatedCommission(ctx, valAddr).Commissions)
}

func TestCalculateRewardsMultiDelegatorMultiSlash(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	// self delegation
	valAddr1 := createValidatorWithBalance(ctx, input, 100_000_000, 10_000_000, 1)
	_, found := input.StakingKeeper.GetValidator(ctx, valAddr1)
	require.True(t, found)

	valAddr2 := createValidatorWithBalance(ctx, input, 100_000_000, 1, 2)
	_, found = input.StakingKeeper.GetValidator(ctx, valAddr2)
	require.True(t, found)

	validator1, found := input.StakingKeeper.GetValidator(ctx, valAddr1)
	require.True(t, found)

	pubkey, err := validator1.ConsPubKey()
	require.NoError(t, err)
	valConsAddr1 := pubkey.Address().Bytes()

	// end block to bond validator
	staking.EndBlocker(ctx, input.StakingKeeper)

	// next block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// fetch validator and delegation
	val := input.StakingKeeper.Validator(ctx, valAddr1)
	del1 := input.StakingKeeper.Delegation(ctx, sdk.AccAddress(valAddr1), valAddr1)

	// allocate some rewards
	initial := math.LegacyNewDecFromInt(input.StakingKeeper.VotingPowerFromConsensusPower(ctx, 10))
	tokens := sdk.DecCoins{{Denom: bondDenom, Amount: initial}}
	input.DistKeeper.AllocateTokensToValidatorPool(ctx, val, bondDenom, tokens)

	// slash the validator
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 3)
	input.StakingKeeper.Slash(ctx, valConsAddr1, ctx.BlockHeight(), sdk.NewDecWithPrec(5, 1))
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 3)

	// second delegation
	bondCoins := sdk.NewCoins(sdk.NewCoin(bondDenom, sdk.NewInt(10_000_000)))
	require.True(t, found)
	shares, err := input.StakingKeeper.Delegate(ctx, sdk.AccAddress(valAddr2), bondCoins, stakingtypes.Unbonded, validator1, true)
	require.NoError(t, err)
	require.Equal(t, sdk.NewDecCoinsFromCoins(bondCoins...), shares)
	del2, found := input.StakingKeeper.GetDelegation(ctx, sdk.AccAddress(valAddr2), valAddr1)
	require.True(t, found)

	// end block
	staking.EndBlocker(ctx, input.StakingKeeper)

	// next block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// allocate some more rewards
	input.DistKeeper.AllocateTokensToValidatorPool(ctx, val, bondDenom, tokens)

	// slash the validator again
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 3)
	input.StakingKeeper.Slash(ctx, valConsAddr1, ctx.BlockHeight(), sdk.NewDecWithPrec(5, 1))
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 3)

	// fetch updated validator
	val = input.StakingKeeper.Validator(ctx, valAddr1)

	// end period
	endingPeriod := input.DistKeeper.IncrementValidatorPeriod(ctx, val)

	// calculate delegation rewards for del1
	rewards := input.DistKeeper.CalculateDelegationRewards(ctx, val, del1, endingPeriod)

	// rewards for del1 should be 5/8 initial (half initial first period, 1/8 initial second period)
	require.Equal(t,
		customtypes.DecPools{{Denom: bondDenom, DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: initial.QuoInt64(2).Add(initial.QuoInt64(8))}}}},
		rewards)

	// calculate delegation rewards for del2
	rewards = input.DistKeeper.CalculateDelegationRewards(ctx, val, del2, endingPeriod)

	// rewards for del2 should be initial / 8
	require.Equal(t,
		customtypes.DecPools{{Denom: bondDenom, DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: initial.QuoInt64(4)}}}},
		rewards)

	// commission should be equal to initial (twice 50% commission, unaffected by slashing)
	require.Equal(t,
		customtypes.DecPools{{Denom: bondDenom, DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: initial}}}},
		input.DistKeeper.GetValidatorAccumulatedCommission(ctx, valAddr1).Commissions)
}

func TestCalculateRewardsMultiDelegatorMultWithdraw(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	// self delegation
	valAddr1 := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)
	_, found := input.StakingKeeper.GetValidator(ctx, valAddr1)
	require.True(t, found)

	valAddr2 := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 2)
	_, found = input.StakingKeeper.GetValidator(ctx, valAddr2)
	require.True(t, found)

	validator1, found := input.StakingKeeper.GetValidator(ctx, valAddr1)
	require.True(t, found)

	// set module account coins
	balancePower := int64(100)
	balanceTokens := input.StakingKeeper.VotingPowerFromConsensusPower(ctx, balancePower)
	distrAcc := input.DistKeeper.GetDistributionAccount(ctx)
	amount := sdk.NewCoins(sdk.NewCoin(bondDenom, balanceTokens))
	err := input.BankKeeper.MintCoins(ctx, authtypes.Minter, amount)
	require.NoError(t, err)
	err = input.BankKeeper.SendCoinsFromModuleToModule(ctx, authtypes.Minter, distrAcc.GetName(), amount)
	require.NoError(t, err)

	// fetch validator and delegation
	val := input.StakingKeeper.Validator(ctx, valAddr1)
	del1 := input.StakingKeeper.Delegation(ctx, sdk.AccAddress(valAddr1), valAddr1)

	// end block
	staking.EndBlocker(ctx, input.StakingKeeper)
	// next block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// allocate some rewards (1)
	initial := int64(20)
	tokens := sdk.DecCoins{{Denom: bondDenom, Amount: sdk.NewDec(initial)}}
	input.DistKeeper.AllocateTokensToValidatorPool(ctx, val, bondDenom, tokens)

	// second delegation
	bondCoins := sdk.NewCoins(sdk.NewCoin(bondDenom, sdk.NewInt(1_000_000)))
	require.True(t, found)
	shares, err := input.StakingKeeper.Delegate(ctx, sdk.AccAddress(valAddr2), bondCoins, stakingtypes.Unbonded, validator1, true)
	require.NoError(t, err)
	require.Equal(t, sdk.NewDecCoinsFromCoins(bondCoins...), shares)

	// fetch updated validator
	del2, found := input.StakingKeeper.GetDelegation(ctx, sdk.AccAddress(valAddr2), valAddr1)
	require.True(t, found)

	// end block
	staking.EndBlocker(ctx, input.StakingKeeper)
	// next block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// allocate some more rewards (2)
	input.DistKeeper.AllocateTokensToValidatorPool(ctx, val, bondDenom, tokens)

	// first delegator withdraws
	_, err = input.DistKeeper.WithdrawDelegationRewards(ctx, sdk.AccAddress(valAddr1), valAddr1)
	require.NoError(t, err)

	// second delegator withdraws
	_, err = input.DistKeeper.WithdrawDelegationRewards(ctx, sdk.AccAddress(valAddr2), valAddr1)
	require.NoError(t, err)

	// validator withdraws commission (1)
	_, err = input.DistKeeper.WithdrawValidatorCommission(ctx, valAddr1)
	require.NoError(t, err)

	// end period
	endingPeriod := input.DistKeeper.IncrementValidatorPeriod(ctx, val)

	// calculate delegation rewards for del1
	rewards := input.DistKeeper.CalculateDelegationRewards(ctx, val, del1, endingPeriod)

	// rewards for del1 should be zero
	require.True(t, rewards.Sum().IsZero())

	// calculate delegation rewards for del2
	rewards = input.DistKeeper.CalculateDelegationRewards(ctx, val, del2, endingPeriod)

	// rewards for del2 should be zero
	require.True(t, rewards.Sum().IsZero())

	// commission should be zero
	require.True(t, input.DistKeeper.GetValidatorAccumulatedCommission(ctx, valAddr1).Commissions.Sum().IsZero())

	// next block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// allocate some more rewards (3)
	input.DistKeeper.AllocateTokensToValidatorPool(ctx, val, bondDenom, tokens)

	// first delegator withdraws again
	_, err = input.DistKeeper.WithdrawDelegationRewards(ctx, sdk.AccAddress(valAddr1), valAddr1)
	require.NoError(t, err)

	// end period
	endingPeriod = input.DistKeeper.IncrementValidatorPeriod(ctx, val)

	// calculate delegation rewards for del2
	rewards = input.DistKeeper.CalculateDelegationRewards(ctx, val, del2, endingPeriod)

	// rewards for del2 should be 1/4 initial
	require.Equal(t,
		customtypes.DecPools{{Denom: bondDenom, DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: sdk.NewDec(initial / 4)}}}},
		rewards)

	// commission should be half initial
	require.Equal(t,
		customtypes.DecPools{{Denom: bondDenom, DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: sdk.NewDec(initial / 2)}}}},
		input.DistKeeper.GetValidatorAccumulatedCommission(ctx, valAddr1).Commissions)

	// next block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// allocate some more rewards (4)
	input.DistKeeper.AllocateTokensToValidatorPool(ctx, val, bondDenom, tokens)

	// withdraw commission (2)
	_, err = input.DistKeeper.WithdrawValidatorCommission(ctx, valAddr1)
	require.NoError(t, err)

	// next block
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	// fetch validator again
	val = input.StakingKeeper.Validator(ctx, valAddr1)

	// end period
	endingPeriod = input.DistKeeper.IncrementValidatorPeriod(ctx, val)

	// calculate delegation rewards for del1
	rewards = input.DistKeeper.CalculateDelegationRewards(ctx, val, del1, endingPeriod)

	// rewards for del1 should be 1/4 initial
	require.Equal(t,
		customtypes.DecPools{{Denom: bondDenom, DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: sdk.NewDec(initial / 4)}}}},
		rewards)

	// calculate delegation rewards for del2
	rewards = input.DistKeeper.CalculateDelegationRewards(ctx, val, del2, endingPeriod)

	// rewards for del2 should be 1/2 initial
	require.Equal(t,
		customtypes.DecPools{{Denom: bondDenom, DecCoins: sdk.DecCoins{{Denom: bondDenom, Amount: sdk.NewDec(initial / 2)}}}},
		rewards)

	// commission should be zero
	require.True(t, input.DistKeeper.GetValidatorAccumulatedCommission(ctx, valAddr1).Commissions.Sum().IsZero())
}
