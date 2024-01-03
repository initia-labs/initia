package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/initia-labs/initia/x/mstaking/keeper"
	"github.com/initia-labs/initia/x/mstaking/types"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func applyValidatorSetUpdates(t *testing.T, ctx sdk.Context, k keeper.Keeper, expectedUpdatesLen int) []abci.ValidatorUpdate {
	updates, err := k.ApplyAndReturnValidatorSetUpdates(ctx)
	require.NoError(t, err)
	if expectedUpdatesLen >= 0 {
		require.Equal(t, expectedUpdatesLen, len(updates), "%v", updates)
	}
	return updates
}

func Test_SlashUnbondingDelegation(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 1)
	validator, err := input.StakingKeeper.Validators.Get(ctx, valAddr)
	require.NoError(t, err)

	ctx = ctx.WithBlockHeight(90)
	_, _, err = input.StakingKeeper.Undelegate(ctx, valAddr.Bytes(), valAddr, sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, math.NewInt(500_000))))
	require.NoError(t, err)

	consAddr, err := validator.GetConsAddr()
	require.NoError(t, err)

	ctx = ctx.WithBlockHeight(100)

	// 10% slashing
	input.StakingKeeper.Slash(ctx, consAddr, 90, math.LegacyNewDecWithPrec(1, 1))
	ubd, err := input.StakingKeeper.GetUnbondingDelegation(ctx, valAddr.Bytes(), valAddr)
	require.NoError(t, err)
	require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin(bondDenom, 450_000)), ubd.Entries[0].Balance)
	require.Equal(t, sdk.NewInt64Coin(bondDenom, 450_000), input.BankKeeper.GetBalance(ctx, input.StakingKeeper.GetNotBondedPool(ctx).GetAddress(), bondDenom))
}

// tests Jail, Unjail
func Test_Revocation(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 1)
	validator, err := input.StakingKeeper.Validators.Get(ctx, valAddr)
	require.NoError(t, err)

	consAddr, err := validator.GetConsAddr()
	require.NoError(t, err)

	// initial state
	val, err := input.StakingKeeper.Validators.Get(ctx, valAddr)
	require.NoError(t, err)
	require.False(t, val.IsJailed())

	// test jail
	require.NoError(t, input.StakingKeeper.Jail(ctx, consAddr))
	val, err = input.StakingKeeper.Validators.Get(ctx, valAddr)
	require.NoError(t, err)
	require.True(t, val.IsJailed())

	// test unjail
	require.NoError(t, input.StakingKeeper.Unjail(ctx, consAddr))
	val, err = input.StakingKeeper.Validators.Get(ctx, valAddr)
	require.NoError(t, err)
	require.False(t, val.IsJailed())
}

func Test_SlashRedelegation(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr1 := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 1)
	valAddr2 := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 2)
	validator, err := input.StakingKeeper.Validators.Get(ctx, valAddr1)
	require.NoError(t, err)

	ctx = ctx.WithBlockHeight(90)
	_, err = input.StakingKeeper.BeginRedelegation(ctx, valAddr1.Bytes(), valAddr1, valAddr2, sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, math.NewInt(500_000))))
	require.NoError(t, err)

	consAddr, err := validator.GetConsAddr()
	require.NoError(t, err)

	ctx = ctx.WithBlockHeight(100)

	// 10% slashing
	_, err = input.StakingKeeper.Slash(ctx, consAddr, 90, math.LegacyNewDecWithPrec(1, 1))
	require.NoError(t, err)
	delegation, err := input.StakingKeeper.GetDelegation(ctx, valAddr1.Bytes(), valAddr2)
	require.NoError(t, err)
	require.Equal(t, sdk.NewDecCoins(sdk.NewInt64DecCoin(bondDenom, 450_000)), delegation.Shares)
}

// tests Slash at a future height (must panic)
func Test_SlashAtFutureHeight(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 1)
	validator, err := input.StakingKeeper.Validators.Get(ctx, valAddr)
	require.NoError(t, err)

	consAddr, err := validator.GetConsAddr()
	require.NoError(t, err)

	fraction := math.LegacyNewDecWithPrec(5, 1)
	ctx = ctx.WithBlockHeight(1)
	require.Panics(t, func() { input.StakingKeeper.Slash(ctx, consAddr, 2, fraction) })
}

// test slash at a negative height
// this just represents pre-genesis and should have the same effect as slashing at height 0
func Test_SlashAtNegativeHeight(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 10_000_000, 1)
	validator, err := input.StakingKeeper.Validators.Get(ctx, valAddr)
	require.NoError(t, err)

	consAddr, err := validator.GetConsAddr()
	require.NoError(t, err)

	fraction := math.LegacyNewDecWithPrec(5, 1)

	bondedPool := input.StakingKeeper.GetBondedPool(ctx)
	oldBondedPoolBalances := input.BankKeeper.GetAllBalances(ctx, bondedPool.GetAddress())

	input.StakingKeeper.Slash(ctx, consAddr, -2, fraction)

	// end block
	applyValidatorSetUpdates(t, ctx, input.StakingKeeper, 1)

	validator, err = input.StakingKeeper.Validators.Get(ctx, valAddr)
	require.NoError(t, err)
	// power decreased (-50%)
	require.Equal(t, int64(5), validator.GetConsensusPower(input.StakingKeeper.PowerReduction(ctx)))

	bondDenoms, err := input.StakingKeeper.BondDenoms(ctx)
	require.NoError(t, err)

	// pool bonded shares decreased
	newBondedPoolBalances := input.BankKeeper.GetAllBalances(ctx, bondedPool.GetAddress())
	diffTokens := oldBondedPoolBalances.Sub(newBondedPoolBalances...).AmountOf(bondDenoms[0])
	require.Equal(t, input.StakingKeeper.VotingPowerFromConsensusPower(ctx, 5).String(), diffTokens.String())
}

// tests Slash at the current height
func Test_SlashValidatorAtCurrentHeight(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 10_000_000, 1)
	validator, err := input.StakingKeeper.Validators.Get(ctx, valAddr)
	require.NoError(t, err)

	consAddr, err := validator.GetConsAddr()
	require.NoError(t, err)

	fraction := math.LegacyNewDecWithPrec(5, 1)

	bondedPool := input.StakingKeeper.GetBondedPool(ctx)
	oldBondedPoolBalances := input.BankKeeper.GetAllBalances(ctx, bondedPool.GetAddress())

	input.StakingKeeper.Slash(ctx, consAddr, ctx.BlockHeight(), fraction)

	// end block
	applyValidatorSetUpdates(t, ctx, input.StakingKeeper, 1)

	// read updated state
	validator, err = input.StakingKeeper.Validators.Get(ctx, valAddr)
	require.NoError(t, err)
	// power decreased
	require.Equal(t, int64(5), validator.GetConsensusPower(input.StakingKeeper.PowerReduction(ctx)))

	bondDenoms, err := input.StakingKeeper.BondDenoms(ctx)
	require.NoError(t, err)

	// pool bonded shares decreased
	newBondedPoolBalances := input.BankKeeper.GetAllBalances(ctx, bondedPool.GetAddress())
	diffTokens := oldBondedPoolBalances.Sub(newBondedPoolBalances...).AmountOf(bondDenoms[0])
	require.Equal(t, input.StakingKeeper.VotingPowerFromConsensusPower(ctx, 5).String(), diffTokens.String())
}

// tests Slash at a previous height with an unbonding delegation
func Test_SlashWithUnbondingDelegation(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 10_000_000, 1)
	validator, err := input.StakingKeeper.Validators.Get(ctx, valAddr)
	require.NoError(t, err)

	consAddr, err := validator.GetConsAddr()
	require.NoError(t, err)
	require.Equal(t, int64(10), validator.GetConsensusPower(input.StakingKeeper.PowerReduction(ctx)))
	fraction := math.LegacyNewDecWithPrec(5, 1)

	ctx = ctx.WithBlockHeight(10)
	_, _, err = input.StakingKeeper.Undelegate(ctx, valAddr.Bytes(), valAddr, sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, math.NewInt(2_000_000))))
	require.NoError(t, err)

	// slash validator for the first time (#1)
	ctx = ctx.WithBlockHeight(12)
	bondedPool := input.StakingKeeper.GetBondedPool(ctx)
	oldBondedPoolBalances := input.BankKeeper.GetAllBalances(ctx, bondedPool.GetAddress())
	_, err = input.StakingKeeper.GetValidatorByConsAddr(ctx, consAddr)
	require.NoError(t, err)
	input.StakingKeeper.Slash(ctx, consAddr, 10, fraction)

	// end block
	applyValidatorSetUpdates(t, ctx, input.StakingKeeper, 1)

	// read updating unbonding delegation
	ubd, err := input.StakingKeeper.GetUnbondingDelegation(ctx, valAddr.Bytes(), valAddr)
	require.NoError(t, err)
	require.Len(t, ubd.Entries, 1)

	// balance decreased (-50%, 2 -> 1)
	require.Equal(t, input.StakingKeeper.VotingPowerFromConsensusPower(ctx, 1), ubd.Entries[0].Balance.AmountOf(bondDenom))

	// bonded tokens burned (-50%, 8 -> 4)
	newBondedPoolBalances := input.BankKeeper.GetAllBalances(ctx, bondedPool.GetAddress())
	diffTokens := oldBondedPoolBalances.Sub(newBondedPoolBalances...).AmountOf(bondDenom)
	require.Equal(t, input.StakingKeeper.VotingPowerFromConsensusPower(ctx, 4), diffTokens)

	// read updated validator
	validator, err = input.StakingKeeper.GetValidatorByConsAddr(ctx, consAddr)
	require.NoError(t, err)

	// power decreased (-50%, 8 -> 4)
	require.Equal(t, int64(4), validator.GetConsensusPower(input.StakingKeeper.PowerReduction(ctx)))

	// slash validator again (#2)
	ctx = ctx.WithBlockHeight(13)
	input.StakingKeeper.Slash(ctx, consAddr, 10, fraction)

	ubd, err = input.StakingKeeper.GetUnbondingDelegation(ctx, valAddr.Bytes(), valAddr)
	require.NoError(t, err)
	require.Len(t, ubd.Entries, 1)

	// balance decreased again (-50%, 1 -> 0)
	require.Equal(t, math.NewInt(0), ubd.Entries[0].Balance.AmountOf(bondDenom))

	// bonded tokens burned again
	newBondedPoolBalances = input.BankKeeper.GetAllBalances(ctx, bondedPool.GetAddress())
	diffTokens = oldBondedPoolBalances.Sub(newBondedPoolBalances...).AmountOf(bondDenom)
	// oldBondPool = 8, newBondPool = 4 -> 2, diff = 4 -> 6
	require.Equal(t, input.StakingKeeper.VotingPowerFromConsensusPower(ctx, 6), diffTokens)

	// read updated validator
	validator, err = input.StakingKeeper.GetValidatorByConsAddr(ctx, consAddr)
	require.NoError(t, err)

	// end block
	applyValidatorSetUpdates(t, ctx, input.StakingKeeper, 1)

	// power decreased by -50% again
	require.Equal(t, int64(4), validator.GetConsensusPower(input.StakingKeeper.PowerReduction(ctx)))

	// slash validator again (#3)
	ctx = ctx.WithBlockHeight(14)
	input.StakingKeeper.Slash(ctx, consAddr, 10, fraction)

	ubd, err = input.StakingKeeper.GetUnbondingDelegation(ctx, valAddr.Bytes(), valAddr)
	require.NoError(t, err)
	require.Len(t, ubd.Entries, 1)

	// balance unchanged (0 -> 0)
	require.Equal(t, math.NewInt(0), ubd.Entries[0].Balance.AmountOf(bondDenom))

	bondDenoms, err := input.StakingKeeper.BondDenoms(ctx)
	require.NoError(t, err)

	// bonded tokens burned again (2 -> 1)
	newBondedPoolBalances = input.BankKeeper.GetAllBalances(ctx, bondedPool.GetAddress())
	diffTokens = oldBondedPoolBalances.Sub(newBondedPoolBalances...).AmountOf(bondDenoms[0])

	// oldBondPool = 8, newBondPool = 4 -> 2 -> 1, diff = 4 -> 6 -> 8
	require.Equal(t, input.StakingKeeper.VotingPowerFromConsensusPower(ctx, 7), diffTokens)

	// read updated validator
	validator, err = input.StakingKeeper.GetValidatorByConsAddr(ctx, consAddr)
	require.NoError(t, err)

	// power decreased by -50% again
	require.Equal(t, int64(2), validator.GetConsensusPower(input.StakingKeeper.PowerReduction(ctx)))

	// slash validator again (#4)
	ctx = ctx.WithBlockHeight(15)
	input.StakingKeeper.Slash(ctx, consAddr, 10, fraction)

	ubd, err = input.StakingKeeper.GetUnbondingDelegation(ctx, valAddr.Bytes(), valAddr)
	require.NoError(t, err)
	require.Len(t, ubd.Entries, 1)

	// balance unchanged
	require.Equal(t, math.NewInt(0), ubd.Entries[0].Balance.AmountOf(bondDenom))

	// apply TM updates
	applyValidatorSetUpdates(t, ctx, input.StakingKeeper, -1)

	// read updated validator
	// power decreased by 1 again, validator is out of stake
	// validator should be in unbonding period
	validator, _ = input.StakingKeeper.GetValidatorByConsAddr(ctx, consAddr)
	require.Equal(t, validator.GetStatus(), types.Unbonding)
}

// tests Slash at a previous height with a redelegation
func Test_SlashWithRedelegation(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr1 := createValidatorWithBalance(ctx, input, 100_000_000, 10_000_000, 1)
	valAddr2 := createValidatorWithBalance(ctx, input, 100_000_000, 10_000_000, 2)
	validator, err := input.StakingKeeper.Validators.Get(ctx, valAddr1)
	require.NoError(t, err)

	consAddr, err := validator.GetConsAddr()
	require.NoError(t, err)
	fraction := math.LegacyNewDecWithPrec(5, 1)

	ctx = ctx.WithBlockHeight(11)
	// set a redelegation
	rdTokens := sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, math.NewInt(6_000_000)))
	_, err = input.StakingKeeper.BeginRedelegation(ctx, valAddr1.Bytes(), valAddr1, valAddr2, rdTokens)
	require.NoError(t, err)

	// update bonded tokens
	bondedPool := input.StakingKeeper.GetBondedPool(ctx)
	notBondedPool := input.StakingKeeper.GetNotBondedPool(ctx)

	oldBonded := input.BankKeeper.GetBalance(ctx, bondedPool.GetAddress(), bondDenom).Amount
	oldNotBonded := input.BankKeeper.GetBalance(ctx, notBondedPool.GetAddress(), bondDenom).Amount

	// slash validator
	ctx = ctx.WithBlockHeight(12)
	require.NotPanics(t, func() { input.StakingKeeper.Slash(ctx, consAddr, 10, fraction) })
	burnAmount := math.LegacyNewDecFromInt(input.StakingKeeper.VotingPowerFromConsensusPower(ctx, 10)).Mul(fraction).TruncateInt()

	bondedPool = input.StakingKeeper.GetBondedPool(ctx)
	notBondedPool = input.StakingKeeper.GetNotBondedPool(ctx)

	// burn bonded tokens from only from delegations
	bondedPoolBalance := input.BankKeeper.GetBalance(ctx, bondedPool.GetAddress(), bondDenom).Amount
	require.True(math.IntEq(t, oldBonded.Sub(burnAmount), bondedPoolBalance))

	notBondedPoolBalance := input.BankKeeper.GetBalance(ctx, notBondedPool.GetAddress(), bondDenom).Amount
	require.True(math.IntEq(t, oldNotBonded, notBondedPoolBalance))

	// read updating redelegation
	rd, err := input.StakingKeeper.GetRedelegation(ctx, valAddr1.Bytes(), valAddr1, valAddr2)
	require.NoError(t, err)
	require.Len(t, rd.Entries, 1)

	// end block
	applyValidatorSetUpdates(t, ctx, input.StakingKeeper, 2)

	// read updated validator
	validator, err = input.StakingKeeper.GetValidatorByConsAddr(ctx, consAddr)
	require.NoError(t, err)
	// 6 redelegation, slash -50%
	require.Equal(t, int64(2), validator.GetConsensusPower(input.StakingKeeper.PowerReduction(ctx)))

}
