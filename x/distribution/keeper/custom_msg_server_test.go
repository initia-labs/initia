package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/initia-labs/initia/x/distribution/keeper"
	customTypes "github.com/initia-labs/initia/x/distribution/types"
	"github.com/stretchr/testify/require"
)

func TestDepositValidatorRewardsPool(t *testing.T) {
	ctx, keepers := createDefaultTestInput(t)
	customMsgServer := keeper.NewCustomMsgServerImpl(keepers.DistKeeper)
	_, _, depositor := keyPubAddr()
	keepers.Faucet.Mint(ctx, depositor, types.NewCoin("uinit", math.NewInt(20)))
	val := createValidatorWithBalance(ctx, keepers, 100, 100, 1)

	cases := []struct {
		name   string
		msg    *customTypes.MsgDepositValidatorRewardsPool
		resp   *customTypes.MsgDepositValidatorRewardsPoolResponse
		errMsg string
	}{
		{
			name: "success",
			msg: &customTypes.MsgDepositValidatorRewardsPool{
				Depositor:        depositor.String(),
				ValidatorAddress: val.String(),
				Denom:            "uinit",
				Amount:           types.NewCoins(types.NewCoin("uinit", math.NewInt(10))),
			},

			resp:   &customTypes.MsgDepositValidatorRewardsPoolResponse{},
			errMsg: "",
		},
		{
			name: "invalid depositor address",
			msg: &customTypes.MsgDepositValidatorRewardsPool{
				Depositor:        "invalid",
				ValidatorAddress: val.String(),
				Denom:            "uinit",
				Amount:           types.NewCoins(types.NewCoin("uinit", math.NewInt(10))),
			},
			resp:   &customTypes.MsgDepositValidatorRewardsPoolResponse{},
			errMsg: "decoding bech32 failed: invalid bech32 string length 7",
		},
		{
			name: "invalid validaor address",
			msg: &customTypes.MsgDepositValidatorRewardsPool{
				Depositor:        depositor.String(),
				ValidatorAddress: "invalid",
				Denom:            "uinit",
				Amount:           types.NewCoins(types.NewCoin("uinit", math.NewInt(10))),
			},
			resp:   &customTypes.MsgDepositValidatorRewardsPoolResponse{},
			errMsg: "decoding bech32 failed: invalid bech32 string length 7",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			out, err := customMsgServer.DepositValidatorRewardsPool(ctx, tc.msg)
			if tc.errMsg == "" {
				require.NoError(t, err)
				require.Equal(t, tc.resp, out)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
			}
		})
	}

}
