package keeper_test

import (
	"testing"
	"time"

	"github.com/initia-labs/initia/x/mstaking/types"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func Test_GetValidator(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	_, found := input.StakingKeeper.GetValidator(ctx, valAddrs[0])
	require.False(t, found)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)
	validator, found := input.StakingKeeper.GetValidator(ctx, valAddr)
	require.True(t, found)
	require.Equal(t, sdk.NewCoins(sdk.NewCoin(bondDenom, sdk.NewInt(1_000_000))), validator.Tokens)
	require.Equal(t, sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, sdk.NewInt(1_000_000))), validator.DelegatorShares)
}

func Test_GetValidatorByConsAddr(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	_, found := input.StakingKeeper.GetValidatorByConsAddr(ctx, valPubKeys[0].Address().Bytes())
	require.False(t, found)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)
	validator, found := input.StakingKeeper.GetValidator(ctx, valAddr)
	require.True(t, found)

	consPubKey, err := validator.ConsPubKey()
	require.NoError(t, err)

	validator, found = input.StakingKeeper.GetValidatorByConsAddr(ctx, consPubKey.Address().Bytes())
	require.True(t, found)
	require.Equal(t, sdk.NewCoins(sdk.NewCoin(bondDenom, sdk.NewInt(1_000_000))), validator.Tokens)
	require.Equal(t, sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, sdk.NewInt(1_000_000))), validator.DelegatorShares)
}

func Test_UpdateValidatorCommission(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)
	validator, found := input.StakingKeeper.GetValidator(ctx, valAddr)
	require.True(t, found)

	updateTime := time.Now().UTC()
	validator.Commission = types.NewCommissionWithTime(
		sdk.NewDecWithPrec(5, 2),  // rate 5%
		sdk.NewDecWithPrec(20, 2), // max rate 20%
		sdk.NewDecWithPrec(5, 2),  // max change 5%
		updateTime,
	)

	// time not passed
	ctx = ctx.WithBlockTime(updateTime)
	_, err := input.StakingKeeper.UpdateValidatorCommission(ctx, validator, sdk.NewDecWithPrec(10, 2))
	require.Error(t, err)

	// after 24 hours
	updateTime = updateTime.Add(time.Hour * 24)
	ctx = ctx.WithBlockTime(updateTime)

	// invalid rate
	_, err = input.StakingKeeper.UpdateValidatorCommission(ctx, validator, sdk.NewDecWithPrec(5, 1))
	require.Error(t, err)

	// valid rate
	commission, err := input.StakingKeeper.UpdateValidatorCommission(ctx, validator, sdk.NewDecWithPrec(10, 2))
	require.NoError(t, err)

	validator.Commission.Rate = sdk.NewDecWithPrec(10, 2)
	validator.Commission.UpdateTime = updateTime
	require.Equal(t, validator.Commission, commission)
}

func Test_RemoveValidator(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)
	_, err := input.StakingKeeper.Undelegate(ctx, valAddr.Bytes(), valAddr, sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, sdk.NewInt(1_000_000))))
	require.NoError(t, err)

	// insert validator to no longer bonded group
	_ = input.StakingKeeper.BlockValidatorUpdates(ctx)

	// update time to fully unbond a validator
	unbondingTime := input.StakingKeeper.UnbondingTime(ctx)
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(unbondingTime))
	_ = input.StakingKeeper.BlockValidatorUpdates(ctx)

	_, found := input.StakingKeeper.GetValidator(ctx, valAddr)
	require.False(t, found)
}

func Test_RemoveValidatorWithUndelegate(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)
	_, err := input.StakingKeeper.Undelegate(ctx, valAddr.Bytes(), valAddr, sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, sdk.NewInt(500_000))))
	require.NoError(t, err)

	unbondingTime := input.StakingKeeper.UnbondingTime(ctx)

	// insert validator to no longer bonded group
	_ = input.StakingKeeper.BlockValidatorUpdates(ctx)

	// update time to fully unbond a validator
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(unbondingTime))
	_ = input.StakingKeeper.BlockValidatorUpdates(ctx)

	_, err = input.StakingKeeper.Undelegate(ctx, valAddr.Bytes(), valAddr, sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, sdk.NewInt(500_000))))
	require.NoError(t, err)

	_, found := input.StakingKeeper.GetValidator(ctx, valAddr)
	require.False(t, found)
}

func Test_GetAllValidators(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr1 := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)
	valAddr2 := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 2)
	valAddr3 := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 3)

	validators := input.StakingKeeper.GetAllValidators(ctx)
	resAddrs := []string{}
	for _, validator := range validators {
		resAddrs = append(resAddrs, validator.OperatorAddress)
	}

	require.Contains(t, resAddrs, valAddr1.String())
	require.Contains(t, resAddrs, valAddr2.String())
	require.Contains(t, resAddrs, valAddr3.String())
}

func Test_GetValidators(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr1 := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)
	valAddr2 := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 2)
	valAddr3 := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 3)

	validators := input.StakingKeeper.GetValidators(ctx, 3)
	resAddrs := []string{}
	for _, validator := range validators {
		resAddrs = append(resAddrs, validator.OperatorAddress)
	}

	require.Contains(t, resAddrs, valAddr1.String())
	require.Contains(t, resAddrs, valAddr2.String())
	require.Contains(t, resAddrs, valAddr3.String())

	validators = input.StakingKeeper.GetValidators(ctx, 2)
	require.Len(t, validators, 2)
}

func Test_GetBondedValidatorsByPower(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr1 := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)
	valAddr2 := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 2)
	valAddr3 := createValidatorWithBalance(ctx, input, 100_000_000, 3_000_000, 3)

	validators := input.StakingKeeper.GetBondedValidatorsByPower(ctx)
	resAddrs := []string{}
	for _, validator := range validators {
		resAddrs = append(resAddrs, validator.OperatorAddress)
	}

	require.Equal(t, []string{
		valAddr3.String(),
		valAddr2.String(),
		valAddr1.String(),
	}, resAddrs)

	pubkey, err := validators[0].ConsPubKey()
	require.NoError(t, err)

	// jail validator 3
	input.StakingKeeper.Jail(ctx, pubkey.Address().Bytes())

	validators = input.StakingKeeper.GetBondedValidatorsByPower(ctx)
	resAddrs = []string{}
	for _, validator := range validators {
		resAddrs = append(resAddrs, validator.OperatorAddress)
	}

	require.Equal(t, []string{
		valAddr2.String(),
		valAddr1.String(),
	}, resAddrs)
}

func Test_LastValidatorPower(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr1 := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)
	valAddr2 := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 2)

	power1 := input.StakingKeeper.GetLastValidatorPower(ctx, valAddr1)
	power2 := input.StakingKeeper.GetLastValidatorPower(ctx, valAddr2)
	validator1, found := input.StakingKeeper.GetValidator(ctx, valAddr1)
	require.True(t, found)
	validator2, found := input.StakingKeeper.GetValidator(ctx, valAddr2)
	require.True(t, found)

	require.Equal(t, validator1.ConsensusPower(input.StakingKeeper.PowerReduction(ctx)), power1)
	require.Equal(t, validator2.ConsensusPower(input.StakingKeeper.PowerReduction(ctx)), power2)

	input.StakingKeeper.IterateLastValidatorPowers(ctx, func(valAddr sdk.ValAddress, power int64) bool {
		if valAddr.Equals(valAddr1) {
			require.Equal(t, validator1.ConsensusPower(input.StakingKeeper.PowerReduction(ctx)), power)
		} else {
			require.Equal(t, validator2.ConsensusPower(input.StakingKeeper.PowerReduction(ctx)), power)
		}
		return false
	})

	resValidators := input.StakingKeeper.GetLastValidators(ctx)
	for _, resVal := range resValidators {
		if resVal.OperatorAddress == validator1.OperatorAddress {
			require.Equal(t, validator1, resVal)
		} else {
			require.Equal(t, validator2, resVal)
		}
	}
}
