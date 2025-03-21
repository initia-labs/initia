package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	initiaapp "github.com/initia-labs/initia/app"
	"github.com/stretchr/testify/require"

	"github.com/initia-labs/initia/x/mstaking/keeper"
	"github.com/initia-labs/initia/x/mstaking/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func Test_UpdateParams(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	params, err := input.StakingKeeper.Params.Get(ctx)
	require.NoError(t, err)

	params.MaxValidators = 10
	ms := keeper.NewMsgServerImpl(input.StakingKeeper)
	_, err = ms.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: input.StakingKeeper.GetAuthority(),
		Params:    params,
	})
	require.NoError(t, err)

	paramsAfter, err := input.StakingKeeper.Params.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, params, paramsAfter)
}

func TestMsgDelegate(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	ms := keeper.NewMsgServerImpl(input.StakingKeeper)
	_, _, delegator := keyPubAddr()
	input.Faucet.Fund(ctx, delegator, sdk.NewCoin(initiaapp.BondDenom, math.NewInt(1000)))
	validator := createValidatorWithBalance(ctx, input, 1000, 1000, 1)

	cases := []struct {
		name   string
		input  *types.MsgDelegate
		expErr bool
		errMsg string
	}{
		{
			name: "invalid validator",
			input: &types.MsgDelegate{
				DelegatorAddress: delegator.String(),
				ValidatorAddress: "invalid",
				Amount:           sdk.NewCoins(sdk.NewCoin(initiaapp.BondDenom, math.NewInt(1000))),
			},
			expErr: true,
			errMsg: "decoding bech32 failed: invalid bech32 string length 7",
		},
		{
			name: "invalid delegator",
			input: &types.MsgDelegate{
				DelegatorAddress: "invalid",
				ValidatorAddress: validator.String(),
				Amount:           sdk.NewCoins(sdk.NewCoin(initiaapp.BondDenom, math.NewInt(1000))),
			},
			expErr: true,
			errMsg: "decoding bech32 failed: invalid bech32 string length 7",
		},
		{
			name: "success",
			input: &types.MsgDelegate{
				DelegatorAddress: delegator.String(),
				ValidatorAddress: validator.String(),
				Amount:           sdk.NewCoins(sdk.NewCoin(initiaapp.BondDenom, math.NewInt(1000))),
			},
			expErr: false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := ms.Delegate(ctx, tc.input)
			if tc.errMsg == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
			}
		})
	}
}

func TestBeginRedelegate(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	ms := keeper.NewMsgServerImpl(input.StakingKeeper)
	_, _, delegator := keyPubAddr()
	input.Faucet.Fund(ctx, delegator, sdk.NewCoin(initiaapp.BondDenom, math.NewInt(1000)))
	srcVali := createValidatorWithBalance(ctx, input, 1000, 1000, 1)
	// set src validator status to active
	ms.Delegate(ctx, &types.MsgDelegate{
		DelegatorAddress: delegator.String(),
		ValidatorAddress: srcVali.String(),
		Amount:           sdk.NewCoins(sdk.NewCoin(initiaapp.BondDenom, math.NewInt(1000))),
	})
	dstVali := createValidatorWithBalance(ctx, input, 1000, 1000, 2)
	cases := []struct {
		name   string
		input  *types.MsgBeginRedelegate
		expErr bool
		errMsg string
	}{
		{
			name: "invalid validator",
			input: &types.MsgBeginRedelegate{
				DelegatorAddress:    delegator.String(),
				ValidatorSrcAddress: srcVali.String(),
				ValidatorDstAddress: "invalid",
				Amount:              sdk.NewCoins(sdk.NewCoin(initiaapp.BondDenom, math.NewInt(1000))),
			},
			expErr: true,
			errMsg: "decoding bech32 failed: invalid bech32 string length 7",
		},
		{
			name: "invalid delegator",
			input: &types.MsgBeginRedelegate{
				DelegatorAddress:    "invalid",
				ValidatorSrcAddress: srcVali.String(),
				ValidatorDstAddress: "invalid",
				Amount:              sdk.NewCoins(sdk.NewCoin(initiaapp.BondDenom, math.NewInt(1000))),
			},
			expErr: true,
			errMsg: "decoding bech32 failed: invalid bech32 string length 7",
		},
		{
			name: "success",
			input: &types.MsgBeginRedelegate{
				DelegatorAddress:    delegator.String(),
				ValidatorSrcAddress: srcVali.String(),
				ValidatorDstAddress: dstVali.String(),
				Amount:              sdk.NewCoins(sdk.NewCoin(initiaapp.BondDenom, math.NewInt(1000))),
			},
			expErr: false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := ms.BeginRedelegate(ctx, tc.input)
			if tc.errMsg == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
			}
		})
	}
}

func TestUndelegate(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	ms := keeper.NewMsgServerImpl(input.StakingKeeper)
	_, _, delegator := keyPubAddr()
	input.Faucet.Fund(ctx, delegator, sdk.NewCoin(initiaapp.BondDenom, math.NewInt(1000)))
	validator := createValidatorWithBalance(ctx, input, 1000, 1000, 1)
	// set validator status to active
	ms.Delegate(ctx, &types.MsgDelegate{
		DelegatorAddress: delegator.String(),
		ValidatorAddress: validator.String(),
		Amount:           sdk.NewCoins(sdk.NewCoin(initiaapp.BondDenom, math.NewInt(1000))),
	})
	cases := []struct {
		name   string
		input  *types.MsgUndelegate
		expErr bool
		errMsg string
	}{
		{
			name: "invalid validator",
			input: &types.MsgUndelegate{
				DelegatorAddress: delegator.String(),
				ValidatorAddress: "invalid",
				Amount:           sdk.NewCoins(sdk.NewCoin(initiaapp.BondDenom, math.NewInt(1000))),
			},
			expErr: true,
			errMsg: "decoding bech32 failed: invalid bech32 string length 7",
		},
		{
			name: "invalid delegator",
			input: &types.MsgUndelegate{
				DelegatorAddress: "invalid",
				ValidatorAddress: validator.String(),
				Amount:           sdk.NewCoins(sdk.NewCoin(initiaapp.BondDenom, math.NewInt(1000))),
			},
			expErr: true,
			errMsg: "decoding bech32 failed: invalid bech32 string length 7",
		},
		{
			name: "success",
			input: &types.MsgUndelegate{
				DelegatorAddress: delegator.String(),
				ValidatorAddress: validator.String(),
				Amount:           sdk.NewCoins(sdk.NewCoin(initiaapp.BondDenom, math.NewInt(1000))),
			},
			expErr: false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := ms.Undelegate(ctx, tc.input)
			if tc.errMsg == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
			}
		})
	}
}

func TestCancelUnbondingDelegation(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	ms := keeper.NewMsgServerImpl(input.StakingKeeper)
	_, _, delegator := keyPubAddr()
	validator := createValidatorWithBalance(ctx, input, 1000, 1000, 1)

	// set validator status to active
	ms.Delegate(ctx, &types.MsgDelegate{
		DelegatorAddress: delegator.String(),
		ValidatorAddress: validator.String(),
		Amount:           sdk.NewCoins(sdk.NewCoin(initiaapp.BondDenom, math.NewInt(1000))),
	})

	// create unbonding delegation
	ubd := types.NewUnbondingDelegation(delegator.String(), validator.String(), 10, ctx.BlockTime().Add(time.Minute*10), sdk.NewCoins(sdk.NewCoin(initiaapp.BondDenom, math.NewInt(1000))), 0)
	require.NoError(t, input.StakingKeeper.SetUnbondingDelegation(ctx, ubd))
	resUnbond, err := input.StakingKeeper.GetUnbondingDelegation(ctx, delegator, validator)
	require.NoError(t, err)
	require.Equal(t, ubd, resUnbond)

	cases := []struct {
		name   string
		input  *types.MsgCancelUnbondingDelegation
		output *types.MsgCancelUnbondingDelegationResponse
		expErr bool
		errMsg string
	}{
		{
			name: "invalid validator",
			input: &types.MsgCancelUnbondingDelegation{
				DelegatorAddress: delegator.String(),
				ValidatorAddress: "invalid",
				Amount:           sdk.NewCoins(sdk.NewCoin(initiaapp.BondDenom, math.NewInt(1000))),
				CreationHeight:   10,
			},

			expErr: true,
			errMsg: "decoding bech32 failed: invalid bech32 string length 7",
		},
		{
			name: "invalid delegator",
			input: &types.MsgCancelUnbondingDelegation{
				DelegatorAddress: "invalid",
				ValidatorAddress: validator.String(),
				Amount:           sdk.NewCoins(sdk.NewCoin(initiaapp.BondDenom, math.NewInt(1000))),
				CreationHeight:   10,
			},

			expErr: true,
			errMsg: "decoding bech32 failed: invalid bech32 string length 7",
		},
		{
			name: "amount is greater than balance",
			input: &types.MsgCancelUnbondingDelegation{
				DelegatorAddress: delegator.String(),
				ValidatorAddress: validator.String(),
				Amount:           sdk.NewCoins(sdk.NewCoin(initiaapp.BondDenom, math.NewInt(1001))),
				CreationHeight:   10,
			},
			expErr: true,
			errMsg: "amount is greater than the unbonding delegation entry balance",
		},
		{
			name: "entry not found at height",
			input: &types.MsgCancelUnbondingDelegation{
				DelegatorAddress: delegator.String(),
				ValidatorAddress: validator.String(),
				Amount:           sdk.NewCoins(sdk.NewCoin(initiaapp.BondDenom, math.NewInt(1000))),
				CreationHeight:   11,
			},
			expErr: true,
			errMsg: "unbonding delegation entry is not found at block height",
		},
		{
			name: "success",
			input: &types.MsgCancelUnbondingDelegation{
				DelegatorAddress: delegator.String(),
				ValidatorAddress: validator.String(),
				Amount:           sdk.NewCoins(sdk.NewCoin(initiaapp.BondDenom, math.NewInt(1000))),
				CreationHeight:   10,
			},
			expErr: false,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			_, err := ms.CancelUnbondingDelegation(ctx, tc.input)
			if tc.errMsg == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
			}
		})
	}
}
