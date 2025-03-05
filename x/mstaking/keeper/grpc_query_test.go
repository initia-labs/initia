package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/v1/x/mstaking/keeper"
	"github.com/initia-labs/initia/v1/x/mstaking/types"
)

func Test_grpcQueryValidators(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	_ = createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 1)
	_ = createValidatorWithBalance(ctx, input, 100_000_000, 2_000_000, 2)

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

	// 1_000_000 (validator creation) + 2_000_000 (extra bond to valAddr2)
	require.Equal(t, res.Balance, sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(3_000_000))))
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
