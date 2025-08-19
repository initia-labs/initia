package keeper_test

import (
	"testing"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	movetypes "github.com/initia-labs/initia/x/move/types"
	"github.com/initia-labs/initia/x/mstaking/keeper"
	"github.com/initia-labs/initia/x/mstaking/types"
	vmtypes "github.com/initia-labs/movevm/types"
)

func Test_grpcQueryValidators(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	// set max validators to 1
	params, err := input.StakingKeeper.GetParams(ctx)
	require.NoError(t, err)
	params.MaxValidators = 2
	err = input.StakingKeeper.SetParams(ctx, params)
	require.NoError(t, err)

	// one validator is in unbonding state
	_ = createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 1)
	_ = createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 2)
	_ = createValidatorWithBalance(ctx, input, 100_000_000, 1_000_000, 3)

	req := types.QueryValidatorsRequest{
		Status: types.BondStatusBonded,
	}

	querier := keeper.Querier{&input.StakingKeeper}
	res, err := querier.Validators(ctx, &req)
	require.NoError(t, err)
	require.Len(t, res.Validators, 2)
}

func Test_grpcQueryValidator(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 1)
	valAddrStr, err := input.StakingKeeper.ValidatorAddressCodec().BytesToString(valAddr)
	require.NoError(t, err)

	req := types.QueryValidatorRequest{
		ValidatorAddr: valAddrStr,
	}

	querier := keeper.Querier{&input.StakingKeeper}
	res, err := querier.Validator(ctx, &req)
	require.NoError(t, err)

	v := res.Validator
	validator, err := input.StakingKeeper.Validators.Get(ctx, valAddr)
	require.NoError(t, err)
	require.Equal(t, validator, v)
}

func Test_grpcQueryValidatorDelegations(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 1)
	valAddrStr, err := input.StakingKeeper.ValidatorAddressCodec().BytesToString(valAddr)
	require.NoError(t, err)

	bondCoins := sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(1_000_000)))
	delAddr := input.Faucet.NewFundedAccount(ctx, bondCoins...)

	validator, err := input.StakingKeeper.Validators.Get(ctx, valAddr)
	require.NoError(t, err)

	shares, err := input.StakingKeeper.Delegate(ctx, delAddr, bondCoins, types.Unbonded, validator, true)
	require.NoError(t, err)
	require.Equal(t, sdk.NewDecCoinsFromCoins(bondCoins...), shares)

	// query validator delegations
	req := types.QueryValidatorDelegationsRequest{
		ValidatorAddr: valAddrStr,
	}

	querier := keeper.Querier{&input.StakingKeeper}
	res, err := querier.ValidatorDelegations(ctx, &req)
	require.NoError(t, err)

	delegations := res.DelegationResponses
	require.Len(t, delegations, 2)

	for _, d := range delegations {
		delAddr, err := input.AccountKeeper.AddressCodec().StringToBytes(d.Delegation.GetDelegatorAddr())
		require.NoError(t, err)
		valAddr, err := input.StakingKeeper.ValidatorAddressCodec().StringToBytes(d.Delegation.GetValidatorAddr())
		require.NoError(t, err)

		delegation, err := input.StakingKeeper.GetDelegation(ctx, delAddr, valAddr)
		require.NoError(t, err)
		require.Equal(t, delegation, d.Delegation)
	}
}

func Test_grpcQueryValidatorUnbondingDelegations(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 1)
	valAddrStr, err := input.StakingKeeper.ValidatorAddressCodec().BytesToString(valAddr)
	require.NoError(t, err)

	bondCoins := sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(1_000_000)))
	delAddr := input.Faucet.NewFundedAccount(ctx, bondCoins...)

	validator, err := input.StakingKeeper.Validators.Get(ctx, valAddr)
	require.NoError(t, err)

	shares, err := input.StakingKeeper.Delegate(ctx, delAddr, bondCoins, types.Unbonded, validator, true)
	require.NoError(t, err)
	require.Equal(t, sdk.NewDecCoinsFromCoins(bondCoins...), shares)

	_, _, err = input.StakingKeeper.Undelegate(ctx, valAddr.Bytes(), valAddr, sdk.NewDecCoins(sdk.NewInt64DecCoin(bondDenom, 500_000)))
	require.NoError(t, err)
	_, _, err = input.StakingKeeper.Undelegate(ctx, delAddr, valAddr, sdk.NewDecCoins(sdk.NewInt64DecCoin(bondDenom, 500_000)))
	require.NoError(t, err)

	// query validator delegations
	req := types.QueryValidatorUnbondingDelegationsRequest{
		ValidatorAddr: valAddrStr,
	}

	querier := keeper.Querier{&input.StakingKeeper}
	res, err := querier.ValidatorUnbondingDelegations(ctx, &req)
	require.NoError(t, err)

	unbondings := res.UnbondingResponses
	require.Len(t, unbondings, 2)

	for _, u := range unbondings {
		delAddr, err := input.AccountKeeper.AddressCodec().StringToBytes(u.DelegatorAddress)
		require.NoError(t, err)
		valAddr, err := input.StakingKeeper.ValidatorAddressCodec().StringToBytes(u.ValidatorAddress)
		require.NoError(t, err)

		unbonding, err := input.StakingKeeper.GetUnbondingDelegation(ctx, delAddr, valAddr)
		require.NoError(t, err)
		require.Equal(t, unbonding, u)
	}
}

func Test_grpcQueryDelegatorDelegations(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr1 := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 1)
	delAddrStr1, err := input.AccountKeeper.AddressCodec().BytesToString(valAddr1)
	require.NoError(t, err)

	valAddr2 := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 2)
	bondCoins := sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(1_000_000)))

	validator, err := input.StakingKeeper.Validators.Get(ctx, valAddr2)
	require.NoError(t, err)

	shares, err := input.StakingKeeper.Delegate(ctx, valAddr1.Bytes(), bondCoins, types.Unbonded, validator, true)
	require.NoError(t, err)
	require.Equal(t, sdk.NewDecCoinsFromCoins(bondCoins...), shares)

	// query delegator delegations
	req := types.QueryDelegatorDelegationsRequest{
		DelegatorAddr: delAddrStr1,
	}

	querier := keeper.Querier{&input.StakingKeeper}
	res, err := querier.DelegatorDelegations(ctx, &req)
	require.NoError(t, err)

	delegations := res.DelegationResponses
	require.Len(t, delegations, 2)

	for _, d := range delegations {
		delAddr, err := input.AccountKeeper.AddressCodec().StringToBytes(d.Delegation.GetDelegatorAddr())
		require.NoError(t, err)
		valAddr, err := input.StakingKeeper.ValidatorAddressCodec().StringToBytes(d.Delegation.GetValidatorAddr())
		require.NoError(t, err)

		delegation, err := input.StakingKeeper.GetDelegation(ctx, delAddr, valAddr)
		require.NoError(t, err)
		require.Equal(t, delegation, d.Delegation)
	}

	// query delegator delegations with invalid status
	req = types.QueryDelegatorDelegationsRequest{
		DelegatorAddr: delAddrStr1,
		Status:        "invalid",
	}
	_, err = querier.DelegatorDelegations(ctx, &req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid validator status invalid")

	// set a validator status to unbonding
	validator.Status = types.Unbonding
	err = input.StakingKeeper.Validators.Set(ctx, valAddr2.Bytes(), validator)
	require.NoError(t, err)

	// query delegator delegations with valid status
	req = types.QueryDelegatorDelegationsRequest{
		DelegatorAddr: delAddrStr1,
		Status:        types.BondStatusUnbonding,
	}
	res, err = querier.DelegatorDelegations(ctx, &req)
	require.NoError(t, err)
	require.Len(t, res.DelegationResponses, 1)
}

func Test_grpcQueryDelegatorTotalDelegationBalance(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr1 := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 1)
	delAddrStr1, err := input.AccountKeeper.AddressCodec().BytesToString(valAddr1)
	require.NoError(t, err)

	valAddr2 := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 2)
	bondCoins := sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(1_000_000)))

	validator, err := input.StakingKeeper.Validators.Get(ctx, valAddr2)
	require.NoError(t, err)

	shares, err := input.StakingKeeper.Delegate(ctx, valAddr1.Bytes(), bondCoins, types.Unbonded, validator, true)
	require.NoError(t, err)
	require.Equal(t, sdk.NewDecCoinsFromCoins(bondCoins...), shares)

	// query delegator delegations
	req := types.QueryDelegatorTotalDelegationBalanceRequest{
		DelegatorAddr: delAddrStr1,
	}

	querier := keeper.Querier{&input.StakingKeeper}
	res, err := querier.DelegatorTotalDelegationBalance(ctx, &req)
	require.NoError(t, err)

	// 2_000_000 (validator creation) + 1_000_000 (extra bond to valAddr2)
	require.Equal(t, res.Balance, sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(3_000_000))))

	// invalid status
	req = types.QueryDelegatorTotalDelegationBalanceRequest{
		DelegatorAddr: delAddrStr1,
		Status:        "invalid",
	}
	_, err = querier.DelegatorTotalDelegationBalance(ctx, &req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid validator status invalid")

	// set a validator status to unbonding
	validator.Status = types.Unbonding
	err = input.StakingKeeper.Validators.Set(ctx, valAddr2.Bytes(), validator)
	require.NoError(t, err)

	// query delegator delegations with valid status
	req = types.QueryDelegatorTotalDelegationBalanceRequest{
		DelegatorAddr: delAddrStr1,
		Status:        types.BondStatusUnbonding,
	}
	res, err = querier.DelegatorTotalDelegationBalance(ctx, &req)
	require.NoError(t, err)
	require.Equal(t, res.Balance, sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(1_000_000))))
}

func Test_grpcQueryDelegatorUnbondingDelegations(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr1 := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 1)
	delAddrStr1, err := input.AccountKeeper.AddressCodec().BytesToString(valAddr1)
	require.NoError(t, err)

	valAddr2 := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 2)

	bondCoins := sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(1_000_000)))

	validator, err := input.StakingKeeper.Validators.Get(ctx, valAddr2)
	require.NoError(t, err)

	shares, err := input.StakingKeeper.Delegate(ctx, valAddr1.Bytes(), bondCoins, types.Unbonded, validator, true)
	require.NoError(t, err)
	require.Equal(t, sdk.NewDecCoinsFromCoins(bondCoins...), shares)

	_, _, err = input.StakingKeeper.Undelegate(ctx, valAddr1.Bytes(), valAddr1, sdk.NewDecCoins(sdk.NewInt64DecCoin(bondDenom, 500_000)))
	require.NoError(t, err)
	_, _, err = input.StakingKeeper.Undelegate(ctx, valAddr1.Bytes(), valAddr2, sdk.NewDecCoins(sdk.NewInt64DecCoin(bondDenom, 500_000)))
	require.NoError(t, err)

	// query delegator undelegations
	req := types.QueryDelegatorUnbondingDelegationsRequest{
		DelegatorAddr: delAddrStr1,
	}

	querier := keeper.Querier{&input.StakingKeeper}
	res, err := querier.DelegatorUnbondingDelegations(ctx, &req)
	require.NoError(t, err)

	unbondings := res.UnbondingResponses
	require.Len(t, unbondings, 2)

	for _, u := range unbondings {
		delAddr, err := input.AccountKeeper.AddressCodec().StringToBytes(u.DelegatorAddress)
		require.NoError(t, err)
		valAddr, err := input.StakingKeeper.ValidatorAddressCodec().StringToBytes(u.ValidatorAddress)
		require.NoError(t, err)

		unbonding, err := input.StakingKeeper.GetUnbondingDelegation(ctx, delAddr, valAddr)
		require.NoError(t, err)
		require.Equal(t, unbonding, u)
	}
}

func Test_grpcQueryDelegatorValidators(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr1 := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 1)
	delAddrStr1, err := input.AccountKeeper.AddressCodec().BytesToString(valAddr1)
	require.NoError(t, err)

	valAddr2 := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 2)

	bondCoins := sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(1_000_000)))

	validator, err := input.StakingKeeper.Validators.Get(ctx, valAddr2)
	require.NoError(t, err)

	shares, err := input.StakingKeeper.Delegate(ctx, valAddr1.Bytes(), bondCoins, types.Unbonded, validator, true)
	require.NoError(t, err)
	require.Equal(t, sdk.NewDecCoinsFromCoins(bondCoins...), shares)

	// query delegator validators
	req := types.QueryDelegatorValidatorsRequest{
		DelegatorAddr: delAddrStr1,
	}

	querier := keeper.Querier{&input.StakingKeeper}
	res, err := querier.DelegatorValidators(ctx, &req)
	require.NoError(t, err)

	validators := res.Validators
	require.Len(t, validators, 2)

	for _, v := range validators {
		valAddr, err := input.StakingKeeper.ValidatorAddressCodec().StringToBytes(v.GetOperator())
		require.NoError(t, err)

		validator, err := input.StakingKeeper.Validators.Get(ctx, valAddr)
		require.NoError(t, err)
		require.Equal(t, validator, v)
	}
}

func Test_grpcQueryDelegatorValidator(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 1)

	delAddrStr, err := input.AccountKeeper.AddressCodec().BytesToString(valAddr)
	require.NoError(t, err)
	valAddrStr, err := input.StakingKeeper.ValidatorAddressCodec().BytesToString(valAddr)
	require.NoError(t, err)

	// query delegator validator
	req := types.QueryDelegatorValidatorRequest{
		DelegatorAddr: delAddrStr,
		ValidatorAddr: valAddrStr,
	}

	querier := keeper.Querier{&input.StakingKeeper}
	res, err := querier.DelegatorValidator(ctx, &req)
	require.NoError(t, err)

	v := res.Validator
	validator, err := input.StakingKeeper.Validators.Get(ctx, valAddr)
	require.NoError(t, err)
	require.Equal(t, validator, v)
}

func Test_grpcQueryDelegation(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 1)

	delAddrStr, err := input.AccountKeeper.AddressCodec().BytesToString(valAddr)
	require.NoError(t, err)
	valAddrStr, err := input.StakingKeeper.ValidatorAddressCodec().BytesToString(valAddr)
	require.NoError(t, err)

	// query delegator validator
	req := types.QueryDelegationRequest{
		DelegatorAddr: delAddrStr,
		ValidatorAddr: valAddrStr,
	}

	querier := keeper.Querier{&input.StakingKeeper}
	res, err := querier.Delegation(ctx, &req)
	require.NoError(t, err)

	d := res.DelegationResponse

	delegation, err := input.StakingKeeper.GetDelegation(ctx, sdk.AccAddress(valAddr), valAddr)
	require.NoError(t, err)
	require.Equal(t, delegation, d.Delegation)
}

func Test_grpcQueryUnbondingDelegation(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 1)

	delAddrStr, err := input.AccountKeeper.AddressCodec().BytesToString(valAddr)
	require.NoError(t, err)
	valAddrStr, err := input.StakingKeeper.ValidatorAddressCodec().BytesToString(valAddr)
	require.NoError(t, err)

	_, _, err = input.StakingKeeper.Undelegate(ctx, valAddr.Bytes(), valAddr, sdk.NewDecCoins(sdk.NewInt64DecCoin(bondDenom, 500_000)))
	require.NoError(t, err)

	// query validator delegations
	req := types.QueryUnbondingDelegationRequest{
		DelegatorAddr: delAddrStr,
		ValidatorAddr: valAddrStr,
	}

	querier := keeper.Querier{&input.StakingKeeper}
	res, err := querier.UnbondingDelegation(ctx, &req)
	require.NoError(t, err)

	u := res.Unbond

	delAddr, err := input.AccountKeeper.AddressCodec().StringToBytes(u.DelegatorAddress)
	require.NoError(t, err)

	valAddr, err = input.StakingKeeper.ValidatorAddressCodec().StringToBytes(u.ValidatorAddress)
	require.NoError(t, err)

	unbonding, err := input.StakingKeeper.GetUnbondingDelegation(ctx, delAddr, valAddr)
	require.NoError(t, err)
	require.Equal(t, unbonding, u)
}

func Test_grpcQueryRedelegations(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr1 := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 1)
	valAddrStr1, err := input.StakingKeeper.ValidatorAddressCodec().BytesToString(valAddr1)
	require.NoError(t, err)

	valAddr2 := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 2)
	valAddrStr2, err := input.StakingKeeper.ValidatorAddressCodec().BytesToString(valAddr2)
	require.NoError(t, err)

	bondCoins := sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(1_000_000)))
	delAddr := input.Faucet.NewFundedAccount(ctx, bondCoins...)
	delAddrStr, err := input.AccountKeeper.AddressCodec().BytesToString(delAddr)
	require.NoError(t, err)

	validator, err := input.StakingKeeper.Validators.Get(ctx, valAddr1)
	require.NoError(t, err)

	shares, err := input.StakingKeeper.Delegate(ctx, delAddr, bondCoins, types.Unbonded, validator, true)
	require.NoError(t, err)
	require.Equal(t, sdk.NewDecCoinsFromCoins(bondCoins...), shares)

	input.StakingKeeper.BeginRedelegation(ctx, valAddr1.Bytes(), valAddr1, valAddr2, sdk.NewDecCoins(sdk.NewInt64DecCoin(bondDenom, 500_000)))
	input.StakingKeeper.BeginRedelegation(ctx, delAddr, valAddr1, valAddr2, sdk.NewDecCoins(sdk.NewInt64DecCoin(bondDenom, 500_000)))

	// query validator delegations
	req := types.QueryRedelegationsRequest{
		DelegatorAddr:    delAddrStr,
		SrcValidatorAddr: valAddrStr1,
		DstValidatorAddr: valAddrStr2,
	}

	querier := keeper.Querier{&input.StakingKeeper}
	res, err := querier.Redelegations(ctx, &req)
	require.NoError(t, err)

	redels := res.RedelegationResponses
	require.Len(t, redels, 1)

	// query with src validators
	req = types.QueryRedelegationsRequest{
		SrcValidatorAddr: valAddrStr1,
	}

	res, err = querier.Redelegations(ctx, &req)
	require.NoError(t, err)

	redels = res.RedelegationResponses
	require.Len(t, redels, 2)

	// query with src validators
	req = types.QueryRedelegationsRequest{
		SrcValidatorAddr: valAddrStr1,
	}

	res, err = querier.Redelegations(ctx, &req)
	require.NoError(t, err)

	redels = res.RedelegationResponses
	require.Len(t, redels, 2)

	// query with dst validators
	req = types.QueryRedelegationsRequest{
		DstValidatorAddr: valAddrStr2,
	}

	res, err = querier.Redelegations(ctx, &req)
	require.NoError(t, err)

	redels = res.RedelegationResponses
	require.Len(t, redels, 2)

	// query with delegator address
	req = types.QueryRedelegationsRequest{
		DelegatorAddr: delAddrStr,
	}

	res2, err := querier.RedelegationsOfDelegator(ctx, &req)
	require.NoError(t, err)

	redels = res2.RedelegationResponses
	require.Len(t, redels, 1)
}

func Test_grpcPool(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	_ = createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 1)

	querier := keeper.Querier{&input.StakingKeeper}
	res, err := querier.Pool(ctx, &types.QueryPoolRequest{})
	require.NoError(t, err)

	require.Equal(t, res.Pool, types.Pool{
		NotBondedTokens:    sdk.NewCoins(),
		BondedTokens:       sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(2_000_000))),
		VotingPowerWeights: sdk.NewDecCoins(sdk.NewDecCoin(bondDenom, math.OneInt())),
	})
}

func Test_grpcParams(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	querier := keeper.Querier{&input.StakingKeeper}
	params, err := querier.Params(ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)

	_params, err := input.StakingKeeper.GetParams(ctx)
	require.NoError(t, err)
	require.Equal(t, _params, params.Params)
}

func Test_grpcMigration(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	// Publish the migration module
	migrationModule := ReadMoveFile("dex_migration")
	err := input.MoveKeeper.PublishModuleBundle(ctx, movetypes.ConvertSDKAddressToVMAddress(movetypes.TestAddr), vmtypes.NewModuleBundle(vmtypes.Module{Code: migrationModule}), movetypes.UpgradePolicy_COMPATIBLE)
	require.NoError(t, err)

	// Create DEX pools for both LP denominations
	baseDenom := bondDenom
	metadataLPOld := createDexPool(t, ctx, input, sdk.NewInt64Coin(baseDenom, 1_000_000_000), sdk.NewInt64Coin("uusdc", 2_500_000_000), math.LegacyNewDecWithPrec(8, 1), math.LegacyNewDecWithPrec(2, 1), true)
	metadataLPNew := createDexPool(t, ctx, input, sdk.NewInt64Coin(baseDenom, 100_000_000), sdk.NewInt64Coin("uusdc2", 250_000_000), math.LegacyNewDecWithPrec(8, 1), math.LegacyNewDecWithPrec(2, 1), true)

	// Get LP denominations from metadata
	lpDenomOld, err := movetypes.DenomFromMetadataAddress(ctx, input.MoveKeeper.MoveBankKeeper(), metadataLPOld)
	require.NoError(t, err)
	lpDenomNew, err := movetypes.DenomFromMetadataAddress(ctx, input.MoveKeeper.MoveBankKeeper(), metadataLPNew)
	require.NoError(t, err)

	// Register a migration
	err = input.StakingKeeper.RegisterMigration(ctx, lpDenomOld, lpDenomNew, "0x2", "dex_migration")
	require.NoError(t, err)

	// Query the migration
	req := types.QueryMigrationRequest{
		DenomLpFrom: lpDenomOld,
		DenomLpTo:   lpDenomNew,
	}

	querier := keeper.Querier{&input.StakingKeeper}
	res, err := querier.Migration(ctx, &req)
	require.NoError(t, err)

	// Verify the migration response
	require.NotNil(t, res.Migration)

	// Get the expected migration from the keeper
	expectedMigration, err := input.StakingKeeper.Migrations.Get(ctx, collections.Join(lpDenomOld, lpDenomNew))
	require.NoError(t, err)

	require.Equal(t, expectedMigration, res.Migration)
}
