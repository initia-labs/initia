package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/x/mstaking/keeper"
	"github.com/initia-labs/initia/x/mstaking/types"
)

func Test_grpcQueryValidators(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	_ = createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 1)
	_ = createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 2)

	req := types.QueryValidatorsRequest{
		Status: types.BondStatusBonded,
	}

	querier := keeper.Querier{&input.StakingKeeper}
	res, err := querier.Validators(sdk.WrapSDKContext(ctx), &req)
	require.NoError(t, err)
	require.Len(t, res.Validators, 2)
}

func Test_grpcQueryValidator(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 1)

	req := types.QueryValidatorRequest{
		ValidatorAddr: valAddr.String(),
	}

	querier := keeper.Querier{&input.StakingKeeper}
	res, err := querier.Validator(sdk.WrapSDKContext(ctx), &req)
	require.NoError(t, err)

	v := res.Validator
	validator, found := input.StakingKeeper.GetValidator(ctx, v.GetOperator())
	require.True(t, found)
	require.Equal(t, validator, v)
}

func Test_grpcQueryValidatorDelegations(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 1)

	bondCoins := sdk.NewCoins(sdk.NewCoin(bondDenom, sdk.NewInt(1_000_000)))
	delAddr := input.Faucet.NewFundedAccount(ctx, bondCoins...)

	validator, found := input.StakingKeeper.GetValidator(ctx, valAddr)
	require.True(t, found)

	shares, err := input.StakingKeeper.Delegate(ctx, delAddr, bondCoins, types.Unbonded, validator, true)
	require.NoError(t, err)
	require.Equal(t, sdk.NewDecCoinsFromCoins(bondCoins...), shares)

	// query validator delegations
	req := types.QueryValidatorDelegationsRequest{
		ValidatorAddr: valAddr.String(),
	}

	querier := keeper.Querier{&input.StakingKeeper}
	res, err := querier.ValidatorDelegations(sdk.WrapSDKContext(ctx), &req)
	require.NoError(t, err)

	delegations := res.DelegationResponses
	require.Len(t, delegations, 2)

	for _, d := range delegations {
		delegation, found := input.StakingKeeper.GetDelegation(ctx, d.Delegation.GetDelegatorAddr(), d.Delegation.GetValidatorAddr())
		require.True(t, found)
		require.Equal(t, delegation, d.Delegation)
	}
}

func Test_grpcQueryValidatorUnbondingDelegations(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 1)

	bondCoins := sdk.NewCoins(sdk.NewCoin(bondDenom, sdk.NewInt(1_000_000)))
	delAddr := input.Faucet.NewFundedAccount(ctx, bondCoins...)

	validator, found := input.StakingKeeper.GetValidator(ctx, valAddr)
	require.True(t, found)

	shares, err := input.StakingKeeper.Delegate(ctx, delAddr, bondCoins, types.Unbonded, validator, true)
	require.NoError(t, err)
	require.Equal(t, sdk.NewDecCoinsFromCoins(bondCoins...), shares)

	input.StakingKeeper.Undelegate(ctx, valAddr.Bytes(), valAddr, sdk.NewDecCoins(sdk.NewInt64DecCoin(bondDenom, 500_000)))
	input.StakingKeeper.Undelegate(ctx, delAddr, valAddr, sdk.NewDecCoins(sdk.NewInt64DecCoin(bondDenom, 500_000)))

	// query validator delegations
	req := types.QueryValidatorUnbondingDelegationsRequest{
		ValidatorAddr: valAddr.String(),
	}

	querier := keeper.Querier{&input.StakingKeeper}
	res, err := querier.ValidatorUnbondingDelegations(sdk.WrapSDKContext(ctx), &req)
	require.NoError(t, err)

	unbondings := res.UnbondingResponses
	require.Len(t, unbondings, 2)

	for _, u := range unbondings {
		delAddr, err := sdk.AccAddressFromBech32(u.DelegatorAddress)
		require.NoError(t, err)

		valAddr, err := sdk.ValAddressFromBech32(u.ValidatorAddress)
		require.NoError(t, err)

		unbonding, found := input.StakingKeeper.GetUnbondingDelegation(ctx, delAddr, valAddr)
		require.True(t, found)
		require.Equal(t, unbonding, u)
	}
}

func Test_grpcQueryDelegatorDelegations(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr1 := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 1)
	valAddr2 := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 2)

	bondCoins := sdk.NewCoins(sdk.NewCoin(bondDenom, sdk.NewInt(1_000_000)))

	validator, found := input.StakingKeeper.GetValidator(ctx, valAddr2)
	require.True(t, found)

	shares, err := input.StakingKeeper.Delegate(ctx, valAddr1.Bytes(), bondCoins, types.Unbonded, validator, true)
	require.NoError(t, err)
	require.Equal(t, sdk.NewDecCoinsFromCoins(bondCoins...), shares)

	// query delegator delegations
	req := types.QueryDelegatorDelegationsRequest{
		DelegatorAddr: sdk.AccAddress(valAddr1.Bytes()).String(),
	}

	querier := keeper.Querier{&input.StakingKeeper}
	res, err := querier.DelegatorDelegations(sdk.WrapSDKContext(ctx), &req)
	require.NoError(t, err)

	delegations := res.DelegationResponses
	require.Len(t, delegations, 2)

	for _, d := range delegations {
		delegation, found := input.StakingKeeper.GetDelegation(ctx, d.Delegation.GetDelegatorAddr(), d.Delegation.GetValidatorAddr())
		require.True(t, found)
		require.Equal(t, delegation, d.Delegation)
	}
}

func Test_grpcQueryDelegatorUnbondingDelegations(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr1 := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 1)
	valAddr2 := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 2)

	bondCoins := sdk.NewCoins(sdk.NewCoin(bondDenom, sdk.NewInt(1_000_000)))

	validator, found := input.StakingKeeper.GetValidator(ctx, valAddr2)
	require.True(t, found)

	shares, err := input.StakingKeeper.Delegate(ctx, valAddr1.Bytes(), bondCoins, types.Unbonded, validator, true)
	require.NoError(t, err)
	require.Equal(t, sdk.NewDecCoinsFromCoins(bondCoins...), shares)

	input.StakingKeeper.Undelegate(ctx, valAddr1.Bytes(), valAddr1, sdk.NewDecCoins(sdk.NewInt64DecCoin(bondDenom, 500_000)))
	input.StakingKeeper.Undelegate(ctx, valAddr1.Bytes(), valAddr2, sdk.NewDecCoins(sdk.NewInt64DecCoin(bondDenom, 500_000)))

	// query delegator undelegations
	req := types.QueryDelegatorUnbondingDelegationsRequest{
		DelegatorAddr: sdk.AccAddress(valAddr1.Bytes()).String(),
	}

	querier := keeper.Querier{&input.StakingKeeper}
	res, err := querier.DelegatorUnbondingDelegations(sdk.WrapSDKContext(ctx), &req)
	require.NoError(t, err)

	unbondings := res.UnbondingResponses
	require.Len(t, unbondings, 2)

	for _, u := range unbondings {
		delAddr, err := sdk.AccAddressFromBech32(u.DelegatorAddress)
		require.NoError(t, err)

		valAddr, err := sdk.ValAddressFromBech32(u.ValidatorAddress)
		require.NoError(t, err)

		unbonding, found := input.StakingKeeper.GetUnbondingDelegation(ctx, delAddr, valAddr)
		require.True(t, found)
		require.Equal(t, unbonding, u)
	}
}

func Test_grpcQueryDelegatorValidators(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr1 := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 1)
	valAddr2 := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 2)

	bondCoins := sdk.NewCoins(sdk.NewCoin(bondDenom, sdk.NewInt(1_000_000)))

	validator, found := input.StakingKeeper.GetValidator(ctx, valAddr2)
	require.True(t, found)

	shares, err := input.StakingKeeper.Delegate(ctx, valAddr1.Bytes(), bondCoins, types.Unbonded, validator, true)
	require.NoError(t, err)
	require.Equal(t, sdk.NewDecCoinsFromCoins(bondCoins...), shares)

	// query delegator validators
	req := types.QueryDelegatorValidatorsRequest{
		DelegatorAddr: sdk.AccAddress(valAddr1.Bytes()).String(),
	}

	querier := keeper.Querier{&input.StakingKeeper}
	res, err := querier.DelegatorValidators(sdk.WrapSDKContext(ctx), &req)
	require.NoError(t, err)

	validators := res.Validators
	require.Len(t, validators, 2)

	for _, v := range validators {
		validator, found := input.StakingKeeper.GetValidator(ctx, v.GetOperator())
		require.True(t, found)
		require.Equal(t, validator, v)
	}
}

func Test_grpcQueryDelegatorValidator(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 1)

	// query delegator validator
	req := types.QueryDelegatorValidatorRequest{
		DelegatorAddr: sdk.AccAddress(valAddr.Bytes()).String(),
		ValidatorAddr: valAddr.String(),
	}

	querier := keeper.Querier{&input.StakingKeeper}
	res, err := querier.DelegatorValidator(sdk.WrapSDKContext(ctx), &req)
	require.NoError(t, err)

	v := res.Validator
	validator, found := input.StakingKeeper.GetValidator(ctx, v.GetOperator())
	require.True(t, found)
	require.Equal(t, validator, v)
}

func Test_grpcQueryDelegation(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 1)

	// query delegator validator
	req := types.QueryDelegationRequest{
		DelegatorAddr: sdk.AccAddress(valAddr.Bytes()).String(),
		ValidatorAddr: valAddr.String(),
	}

	querier := keeper.Querier{&input.StakingKeeper}
	res, err := querier.Delegation(sdk.WrapSDKContext(ctx), &req)
	require.NoError(t, err)

	d := res.DelegationResponse

	delegation, found := input.StakingKeeper.GetDelegation(ctx, d.Delegation.GetDelegatorAddr(), d.Delegation.GetValidatorAddr())
	require.True(t, found)
	require.Equal(t, delegation, d.Delegation)
}

func Test_grpcQueryUnbondingDelegation(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 1)

	input.StakingKeeper.Undelegate(ctx, valAddr.Bytes(), valAddr, sdk.NewDecCoins(sdk.NewInt64DecCoin(bondDenom, 500_000)))

	// query validator delegations
	req := types.QueryUnbondingDelegationRequest{
		DelegatorAddr: sdk.AccAddress(valAddr.Bytes()).String(),
		ValidatorAddr: valAddr.String(),
	}

	querier := keeper.Querier{&input.StakingKeeper}
	res, err := querier.UnbondingDelegation(sdk.WrapSDKContext(ctx), &req)
	require.NoError(t, err)

	u := res.Unbond

	delAddr, err := sdk.AccAddressFromBech32(u.DelegatorAddress)
	require.NoError(t, err)

	valAddr, err = sdk.ValAddressFromBech32(u.ValidatorAddress)
	require.NoError(t, err)

	unbonding, found := input.StakingKeeper.GetUnbondingDelegation(ctx, delAddr, valAddr)
	require.True(t, found)
	require.Equal(t, unbonding, u)
}

func Test_grpcQueryRedelegations(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	valAddr1 := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 1)
	valAddr2 := createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 2)

	bondCoins := sdk.NewCoins(sdk.NewCoin(bondDenom, sdk.NewInt(1_000_000)))
	delAddr := input.Faucet.NewFundedAccount(ctx, bondCoins...)

	validator, found := input.StakingKeeper.GetValidator(ctx, valAddr1)
	require.True(t, found)

	shares, err := input.StakingKeeper.Delegate(ctx, delAddr, bondCoins, types.Unbonded, validator, true)
	require.NoError(t, err)
	require.Equal(t, sdk.NewDecCoinsFromCoins(bondCoins...), shares)

	input.StakingKeeper.BeginRedelegation(ctx, valAddr1.Bytes(), valAddr1, valAddr2, sdk.NewDecCoins(sdk.NewInt64DecCoin(bondDenom, 500_000)))
	input.StakingKeeper.BeginRedelegation(ctx, delAddr, valAddr1, valAddr2, sdk.NewDecCoins(sdk.NewInt64DecCoin(bondDenom, 500_000)))

	// query validator delegations
	req := types.QueryRedelegationsRequest{
		DelegatorAddr:    delAddr.String(),
		SrcValidatorAddr: valAddr1.String(),
		DstValidatorAddr: valAddr2.String(),
	}

	querier := keeper.Querier{&input.StakingKeeper}
	res, err := querier.Redelegations(sdk.WrapSDKContext(ctx), &req)
	require.NoError(t, err)

	redels := res.RedelegationResponses
	require.Len(t, redels, 1)

	// query with src validators
	req = types.QueryRedelegationsRequest{
		SrcValidatorAddr: valAddr1.String(),
	}

	res, err = querier.Redelegations(sdk.WrapSDKContext(ctx), &req)
	require.NoError(t, err)

	redels = res.RedelegationResponses
	require.Len(t, redels, 2)

	// query with delegator addr & src validators
	req = types.QueryRedelegationsRequest{
		DelegatorAddr:    delAddr.String(),
		SrcValidatorAddr: valAddr1.String(),
	}

	res, err = querier.Redelegations(sdk.WrapSDKContext(ctx), &req)
	require.NoError(t, err)

	redels = res.RedelegationResponses
	require.Len(t, redels, 1)

	// query with delegator addr & dst validators
	req = types.QueryRedelegationsRequest{
		DelegatorAddr:    delAddr.String(),
		DstValidatorAddr: valAddr2.String(),
	}

	res, err = querier.Redelegations(sdk.WrapSDKContext(ctx), &req)
	require.NoError(t, err)

	redels = res.RedelegationResponses
	require.Len(t, redels, 1)
}

func Test_grpcPool(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	_ = createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 1)

	querier := keeper.Querier{&input.StakingKeeper}
	res, err := querier.Pool(sdk.WrapSDKContext(ctx), &types.QueryPoolRequest{})
	require.NoError(t, err)

	require.Equal(t, res.Pool, types.Pool{
		NotBondedTokens: sdk.NewCoins(),
		BondedTokens:    sdk.NewCoins(sdk.NewCoin(bondDenom, sdk.NewInt(2_000_000))),
	})
}

func Test_grpcParams(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	querier := keeper.Querier{&input.StakingKeeper}
	params, err := querier.Params(sdk.WrapSDKContext(ctx), &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, input.StakingKeeper.GetParams(ctx), params.Params)
}
