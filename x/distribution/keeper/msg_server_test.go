package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/initia-labs/initia/x/distribution/keeper"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"
)

func TestMsgSetWithdrawAddress(t *testing.T) {
	ctx, testKps := createDefaultTestInput(t)
	msgServer := keeper.NewMsgServerImpl(testKps.DistKeeper)
	_, _, addr0Str := keyPubAddr()
	_, _, addr1Str := keyPubAddr()
	cases := []struct {
		name   string
		msg    *types.MsgSetWithdrawAddress
		errMsg string
	}{
		{
			name: "success",
			msg: &types.MsgSetWithdrawAddress{
				DelegatorAddress: addr0Str.String(),
				WithdrawAddress:  addr1Str.String(),
			},
			errMsg: "",
		},
		{
			name: "invalid delegator address",
			msg: &types.MsgSetWithdrawAddress{
				DelegatorAddress: "invalid",
				WithdrawAddress:  addr1Str.String(),
			},
			errMsg: "decoding bech32 failed: invalid bech32 string length 7",
		},
		{
			name: "invalid withdraw address",
			msg: &types.MsgSetWithdrawAddress{
				DelegatorAddress: addr0Str.String(),
				WithdrawAddress:  "invalid",
			},
			errMsg: "decoding bech32 failed: invalid bech32 string length 7",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := msgServer.SetWithdrawAddress(ctx, tc.msg)
			if tc.errMsg == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
			}
		})
	}
}

func TestMsgWithdrawDelegatorReward(t *testing.T) {
	ctx, testKps := createDefaultTestInput(t)
	msgServer := keeper.NewMsgServerImpl(testKps.DistKeeper)
	_, _, addr0Str := keyPubAddr()
	valAddr1Str := createValidatorWithBalance(ctx, testKps, 100_000_000, 1_000_000, 1)
	cases := []struct {
		name   string
		preRun func()
		msg    *types.MsgWithdrawDelegatorReward
		errMsg string
	}{
		{
			name: "invalid delegator address",
			msg: &types.MsgWithdrawDelegatorReward{
				DelegatorAddress: "invalid",
				ValidatorAddress: valAddr1Str.String(),
			},
			errMsg: "invalid delegator address",
		},
		{
			name: "invalid validator address",
			msg: &types.MsgWithdrawDelegatorReward{
				DelegatorAddress: addr0Str.String(),
				ValidatorAddress: "invalid",
			},
			errMsg: "invalid validator address",
		},
		{
			name: "no validator",
			msg: &types.MsgWithdrawDelegatorReward{
				DelegatorAddress: addr0Str.String(),
				ValidatorAddress: valAddr1Str.String(),
			},
			errMsg: "no validator distribution info",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.preRun != nil {
				tc.preRun()
			}
			_, err := msgServer.WithdrawDelegatorReward(ctx, tc.msg)
			if tc.errMsg == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}

}

func TestMsgWithdrawValidatorCommission(t *testing.T) {
	ctx, testKps := createDefaultTestInput(t)
	msgServer := keeper.NewMsgServerImpl(testKps.DistKeeper)
	valAddr1Str := createValidatorWithBalance(ctx, testKps, 100_000_000, 1_000_000, 1)

	cases := []struct {
		name   string
		preRun func()
		msg    *types.MsgWithdrawValidatorCommission
		errMsg string
	}{
		{
			name: "invalid validator address",
			msg: &types.MsgWithdrawValidatorCommission{
				ValidatorAddress: "invalid",
			},
			errMsg: "invalid validator address",
		},
		{
			name: "no validator commission to withdraw",
			msg: &types.MsgWithdrawValidatorCommission{
				ValidatorAddress: valAddr1Str.String(),
			},
			errMsg: "no validator commission to withdraw",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.preRun != nil {
				tc.preRun()
			}
			_, err := msgServer.WithdrawValidatorCommission(ctx, tc.msg)
			if tc.errMsg == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}

}

func TestMsgFundCommunityPool(t *testing.T) {
	ctx, testKps := createDefaultTestInput(t)
	msgServer := keeper.NewMsgServerImpl(testKps.DistKeeper)
	_, _, addr0Str := keyPubAddr()
	testKps.BankKeeper.MintCoins(ctx, authtypes.Minter, sdk.NewCoins(sdk.NewCoin("uinit", math.NewInt(1000))))
	testKps.BankKeeper.SendCoinsFromModuleToAccount(ctx, authtypes.Minter, sdk.AccAddress(addr0Str), sdk.NewCoins(sdk.NewCoin("uinit", math.NewInt(1000))))
	cases := []struct {
		name   string
		msg    *types.MsgFundCommunityPool
		errMsg string
	}{
		{
			name: "invalid depositor address",
			msg: &types.MsgFundCommunityPool{
				Depositor: "invalid",
				Amount:    sdk.NewCoins(sdk.NewCoin("uinit", math.NewInt(100))),
			},
			errMsg: "invalid depositor address",
		},
		{
			name: "success",
			msg: &types.MsgFundCommunityPool{
				Depositor: addr0Str.String(),
				Amount:    sdk.NewCoins(sdk.NewCoin("uinit", math.NewInt(1000))),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := msgServer.FundCommunityPool(ctx, tc.msg)
			if tc.errMsg == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
