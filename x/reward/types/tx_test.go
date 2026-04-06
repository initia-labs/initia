package types

import (
	"testing"

	"github.com/cometbft/cometbft/crypto/ed25519"
	"github.com/stretchr/testify/require"

	coreaddress "cosmossdk.io/core/address"
	sdkmath "cosmossdk.io/math"

	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func TestNewMsgUpdateParams(t *testing.T) {
	params := DefaultParams()

	msg := NewMsgUpdateParams("authority", params)
	require.Equal(t, &MsgUpdateParams{
		Authority: "authority",
		Params:    params,
	}, msg)
}

func TestMsgUpdateParamsValidate(t *testing.T) {
	ac := addresscodec.NewBech32Codec("init")
	validAuthority := randomAuthority(t, ac)

	testCases := []struct {
		name      string
		msg       MsgUpdateParams
		err       error
		errSubstr string
	}{
		{
			name: "valid",
			msg: MsgUpdateParams{
				Authority: validAuthority,
				Params:    DefaultParams(),
			},
		},
		{
			name: "invalid authority",
			msg: MsgUpdateParams{
				Authority: "invalid-authority",
				Params:    DefaultParams(),
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "invalid params",
			msg: MsgUpdateParams{
				Authority: validAuthority,
				Params: func() Params {
					p := DefaultParams()
					p.DilutionPeriod = 0
					return p
				}(),
			},
			errSubstr: "invalid dilution period",
		},
	}

	for _, tc := range testCases {

		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.Validate(ac)
			if tc.err != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tc.err)
				return
			}
			if tc.errSubstr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tc.errSubstr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestNewMsgFundCommunityPool(t *testing.T) {
	amount := sdk.NewCoins(sdk.NewInt64Coin("uinit", 100))

	msg := NewMsgFundCommunityPool("authority", amount)
	require.Equal(t, &MsgFundCommunityPool{
		Authority: "authority",
		Amount:    amount,
	}, msg)
}

func TestMsgFundCommunityPoolValidate(t *testing.T) {
	ac := addresscodec.NewBech32Codec("init")
	validAuthority := randomAuthority(t, ac)

	testCases := []struct {
		name      string
		msg       MsgFundCommunityPool
		err       error
		errSubstr string
	}{
		{
			name: "valid",
			msg: MsgFundCommunityPool{
				Authority: validAuthority,
				Amount:    sdk.NewCoins(sdk.NewInt64Coin("uinit", 10)),
			},
		},
		{
			name: "invalid authority",
			msg: MsgFundCommunityPool{
				Authority: "invalid-authority",
				Amount:    sdk.NewCoins(sdk.NewInt64Coin("uinit", 10)),
			},
			err: sdkerrors.ErrInvalidAddress,
		},
		{
			name: "invalid amount",
			msg: MsgFundCommunityPool{
				Authority: validAuthority,
				Amount: sdk.Coins{
					{
						Denom:  "uinit",
						Amount: sdkmath.ZeroInt(),
					},
				},
			},
			err: sdkerrors.ErrInvalidCoins,
		},
	}

	for _, tc := range testCases {

		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.Validate(ac)
			if tc.err != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, tc.err)
				return
			}
			if tc.errSubstr != "" {
				require.Error(t, err)
				require.ErrorContains(t, err, tc.errSubstr)
				return
			}
			require.NoError(t, err)
		})
	}
}

func randomAuthority(t *testing.T, ac coreaddress.Codec) string {
	t.Helper()

	addr, err := ac.BytesToString(ed25519.GenPrivKey().PubKey().Address())
	require.NoError(t, err)

	return addr
}
