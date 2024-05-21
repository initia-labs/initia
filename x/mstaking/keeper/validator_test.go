package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	"github.com/initia-labs/initia/x/mstaking/types"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func Test_GetValidator(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	_, err := input.StakingKeeper.Validators.Get(ctx, valAddrs[0])
	require.ErrorIs(t, err, collections.ErrNotFound)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)
	validator, err := input.StakingKeeper.Validators.Get(ctx, valAddr)
	require.NoError(t, err)
	require.Equal(t, sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(1_000_000))), validator.Tokens)
	require.Equal(t, sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, math.NewInt(1_000_000))), validator.DelegatorShares)
}

func Test_GetValidatorByConsAddr(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	_, err := input.StakingKeeper.GetValidatorByConsAddr(ctx, valPubKeys[0].Address().Bytes())
	require.ErrorIs(t, err, collections.ErrNotFound)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)
	validator, err := input.StakingKeeper.Validators.Get(ctx, valAddr)
	require.NoError(t, err)

	consPubKey, err := validator.ConsPubKey()
	require.NoError(t, err)

	validator, err = input.StakingKeeper.GetValidatorByConsAddr(ctx, consPubKey.Address().Bytes())
	require.NoError(t, err)
	require.Equal(t, sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(1_000_000))), validator.Tokens)
	require.Equal(t, sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, math.NewInt(1_000_000))), validator.DelegatorShares)
}

func Test_UpdateValidatorCommission(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)
	validator, err := input.StakingKeeper.Validators.Get(ctx, valAddr)
	require.NoError(t, err)

	updateTime := time.Now().UTC()
	validator.Commission = types.NewCommissionWithTime(
		math.LegacyNewDecWithPrec(5, 2),  // rate 5%
		math.LegacyNewDecWithPrec(20, 2), // max rate 20%
		math.LegacyNewDecWithPrec(5, 2),  // max change 5%
		updateTime,
	)

	// time not passed
	ctx = ctx.WithBlockTime(updateTime)
	_, err = input.StakingKeeper.UpdateValidatorCommission(ctx, validator, math.LegacyNewDecWithPrec(10, 2))
	require.Error(t, err)

	// after 24 hours
	updateTime = updateTime.Add(time.Hour * 24)
	ctx = ctx.WithBlockTime(updateTime)

	// invalid rate
	_, err = input.StakingKeeper.UpdateValidatorCommission(ctx, validator, math.LegacyNewDecWithPrec(5, 1))
	require.Error(t, err)

	// valid rate
	commission, err := input.StakingKeeper.UpdateValidatorCommission(ctx, validator, math.LegacyNewDecWithPrec(10, 2))
	require.NoError(t, err)

	validator.Commission.Rate = math.LegacyNewDecWithPrec(10, 2)
	validator.Commission.UpdateTime = updateTime
	require.Equal(t, validator.Commission, commission)
}

func Test_RemoveValidator(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)
	_, _, err := input.StakingKeeper.Undelegate(ctx, valAddr.Bytes(), valAddr, sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, math.NewInt(1_000_000))))
	require.NoError(t, err)

	// insert validator to no longer bonded group
	_, err = input.StakingKeeper.BlockValidatorUpdates(ctx)
	require.NoError(t, err)

	// update time to fully unbond a validator
	unbondingTime, err := input.StakingKeeper.UnbondingTime(ctx)
	require.NoError(t, err)

	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(unbondingTime))
	_, err = input.StakingKeeper.BlockValidatorUpdates(ctx)
	require.NoError(t, err)

	_, err = input.StakingKeeper.Validators.Get(ctx, valAddr)
	require.ErrorIs(t, err, collections.ErrNotFound)
}

func Test_RemoveValidatorWithUndelegate(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)
	_, _, err := input.StakingKeeper.Undelegate(ctx, valAddr.Bytes(), valAddr, sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, math.NewInt(500_000))))
	require.NoError(t, err)

	unbondingTime, err := input.StakingKeeper.UnbondingTime(ctx)
	require.NoError(t, err)

	// insert validator to no longer bonded group
	_, err = input.StakingKeeper.BlockValidatorUpdates(ctx)
	require.NoError(t, err)

	// update time to fully unbond a validator
	ctx = ctx.WithBlockTime(ctx.BlockTime().Add(unbondingTime))
	_, err = input.StakingKeeper.BlockValidatorUpdates(ctx)
	require.NoError(t, err)

	_, _, err = input.StakingKeeper.Undelegate(ctx, valAddr.Bytes(), valAddr, sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, math.NewInt(500_000))))
	require.NoError(t, err)

	_, err = input.StakingKeeper.Validators.Get(ctx, valAddr)
	require.ErrorIs(t, err, collections.ErrNotFound)
}

func Test_GetAllValidators(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr1 := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)
	valAddr2 := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 2)
	valAddr3 := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 3)

	valAddrStr1, err := input.StakingKeeper.ValidatorAddressCodec().BytesToString(valAddr1)
	require.NoError(t, err)
	valAddrStr2, err := input.StakingKeeper.ValidatorAddressCodec().BytesToString(valAddr2)
	require.NoError(t, err)
	valAddrStr3, err := input.StakingKeeper.ValidatorAddressCodec().BytesToString(valAddr3)
	require.NoError(t, err)

	validators, err := input.StakingKeeper.GetAllValidators(ctx)
	require.NoError(t, err)

	resAddrs := []string{}
	for _, validator := range validators {
		resAddrs = append(resAddrs, validator.OperatorAddress)
	}

	require.Contains(t, resAddrs, valAddrStr1)
	require.Contains(t, resAddrs, valAddrStr2)
	require.Contains(t, resAddrs, valAddrStr3)
}

func Test_GetValidators(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr1 := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)
	valAddr2 := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 2)
	valAddr3 := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 3)

	valAddrStr1, err := input.StakingKeeper.ValidatorAddressCodec().BytesToString(valAddr1)
	require.NoError(t, err)
	valAddrStr2, err := input.StakingKeeper.ValidatorAddressCodec().BytesToString(valAddr2)
	require.NoError(t, err)
	valAddrStr3, err := input.StakingKeeper.ValidatorAddressCodec().BytesToString(valAddr3)
	require.NoError(t, err)

	validators, err := input.StakingKeeper.GetValidators(ctx, 3)
	require.NoError(t, err)

	resAddrs := []string{}
	for _, validator := range validators {
		resAddrs = append(resAddrs, validator.OperatorAddress)
	}

	require.Contains(t, resAddrs, valAddrStr1)
	require.Contains(t, resAddrs, valAddrStr2)
	require.Contains(t, resAddrs, valAddrStr3)

	validators, err = input.StakingKeeper.GetValidators(ctx, 2)
	require.NoError(t, err)
	require.Len(t, validators, 2)
}

func Test_GetBondedValidatorsByPower(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr1 := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)
	valAddr2 := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 2)
	valAddr3 := createValidatorWithBalance(ctx, input, 100_000_000, 3_000_000, 3)

	valAddrStr1, err := input.StakingKeeper.ValidatorAddressCodec().BytesToString(valAddr1)
	require.NoError(t, err)
	valAddrStr2, err := input.StakingKeeper.ValidatorAddressCodec().BytesToString(valAddr2)
	require.NoError(t, err)
	valAddrStr3, err := input.StakingKeeper.ValidatorAddressCodec().BytesToString(valAddr3)
	require.NoError(t, err)

	validators, err := input.StakingKeeper.GetBondedValidatorsByPower(ctx)
	require.NoError(t, err)

	resAddrs := []string{}
	for _, validator := range validators {
		resAddrs = append(resAddrs, validator.OperatorAddress)
	}

	require.Equal(t, []string{
		valAddrStr3,
		valAddrStr2,
		valAddrStr1,
	}, resAddrs)

	pubkey, err := validators[0].ConsPubKey()
	require.NoError(t, err)

	// jail validator 3
	input.StakingKeeper.Jail(ctx, pubkey.Address().Bytes())

	validators, err = input.StakingKeeper.GetBondedValidatorsByPower(ctx)
	require.NoError(t, err)

	resAddrs = []string{}
	for _, validator := range validators {
		resAddrs = append(resAddrs, validator.OperatorAddress)
	}

	require.Equal(t, []string{
		valAddrStr2,
		valAddrStr1,
	}, resAddrs)
}

func Test_LastValidatorPower(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr1 := createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 1)
	valAddr2 := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 2)

	power1, err := input.StakingKeeper.GetLastValidatorConsPower(ctx, valAddr1)
	require.NoError(t, err)
	power2, err := input.StakingKeeper.GetLastValidatorConsPower(ctx, valAddr2)
	require.NoError(t, err)
	validator1, err := input.StakingKeeper.Validators.Get(ctx, valAddr1)
	require.NoError(t, err)
	validator2, err := input.StakingKeeper.Validators.Get(ctx, valAddr2)
	require.NoError(t, err)

	require.Equal(t, validator1.ConsensusPower(input.StakingKeeper.PowerReduction(ctx)), power1)
	require.Equal(t, validator2.ConsensusPower(input.StakingKeeper.PowerReduction(ctx)), power2)

	require.NoError(t, input.StakingKeeper.IterateLastValidatorConsPowers(ctx, func(valAddr sdk.ValAddress, power int64) (bool, error) {
		if valAddr.Equals(valAddr1) {
			require.Equal(t, validator1.ConsensusPower(input.StakingKeeper.PowerReduction(ctx)), power)
		} else {
			require.Equal(t, validator2.ConsensusPower(input.StakingKeeper.PowerReduction(ctx)), power)
		}
		return false, nil
	}))

	resValidators, err := input.StakingKeeper.GetLastValidators(ctx)
	require.NoError(t, err)

	for _, resVal := range resValidators {
		if resVal.OperatorAddress == validator1.OperatorAddress {
			require.Equal(t, validator1, resVal)
		} else {
			require.Equal(t, validator2, resVal)
		}
	}
}
