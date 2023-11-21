package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	icatypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
	ibctesting "github.com/cosmos/ibc-go/v7/testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/initia-labs/initia/x/intertx/types"
)

var (
	// TestOwnerAddress defines a reusable bech32 address for testing purposes
	TestOwnerAddress = "cosmos17dtl0mjt3t77kpuhg2edqzjpszulwhgzuj9ljs"

	// TestVersion defines a reusable interchainaccounts version string for testing purposes
	TestVersion = string(icatypes.ModuleCdc.MustMarshalJSON(&icatypes.Metadata{
		Version:                icatypes.Version,
		ControllerConnectionId: ibctesting.FirstConnectionID,
		HostConnectionId:       ibctesting.FirstConnectionID,
		Encoding:               icatypes.EncodingProtobuf,
		TxType:                 icatypes.TxTypeSDKMultiMsg,
	}))

	TestMessage = &banktypes.MsgSend{
		FromAddress: "cosmos17dtl0mjt3t77kpuhg2edqzjpszulwhgzuj9ljs",
		ToAddress:   "cosmos17dtl0mjt3t77kpuhg2edqzjpszulwhgzuj9ljs",
		Amount:      sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(100))),
	}
)

// TestMsgRegisterAccountValidateBasic tests ValidateBasic for MsgRegisterAccount
func TestMsgRegisterAccountValidateBasic(t *testing.T) {
	testCases := []struct {
		name    string
		msg     *types.MsgRegisterAccount
		expPass bool
	}{
		{"success", types.NewMsgRegisterAccount(TestOwnerAddress, ibctesting.FirstConnectionID, TestVersion), true},
		{"owner address is empty", types.NewMsgRegisterAccount("", ibctesting.FirstConnectionID, TestVersion), false},
		{"owner address is invalid", types.NewMsgRegisterAccount("invalid_address", ibctesting.FirstConnectionID, TestVersion), false},
	}

	for i, tc := range testCases {
		err := tc.msg.ValidateBasic()
		if tc.expPass {
			require.NoError(t, err, "valid test case %d failed: %s", i, tc.name)
		} else {
			require.Error(t, err, "invalid test case %d passed: %s", i, tc.name)
		}
	}
}

// TestMsgRegisterAccountGetSigners tests GetSigners for MsgRegisterAccount
func TestMsgRegisterAccountGetSigners(t *testing.T) {
	expSigner, err := sdk.AccAddressFromBech32(TestOwnerAddress)
	require.NoError(t, err)

	msg := types.NewMsgRegisterAccount(TestOwnerAddress, ibctesting.FirstConnectionID, TestVersion)

	require.Equal(t, []sdk.AccAddress{expSigner}, msg.GetSigners())
}

// TestMsgSubmitTxValidateBasic tests ValidateBasic for MsgSubmitTx
func TestMsgSubmitTxValidateBasic(t *testing.T) {
	var msg *types.MsgSubmitTx

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
	}{
		{
			"success",
			func() {},
			true,
		},
		{
			"owner address is invalid",
			func() {
				msg.Owner = "invalid_address"
			},
			false,
		},
	}

	for i, tc := range testCases {
		msg, _ = types.NewMsgSubmitTx(TestMessage, ibctesting.FirstConnectionID, TestOwnerAddress)

		tc.malleate()

		err := msg.ValidateBasic()
		if tc.expPass {
			require.NoError(t, err, "valid test case %d failed: %s", i, tc.name)
		} else {
			require.Error(t, err, "invalid test case %d passed: %s", i, tc.name)
		}
	}
}

// TestMsgSubmitTxGetSigners tests GetSigners for MsgSubmitTx
func TestMsgSubmitTxGetSigners(t *testing.T) {
	expSigner, err := sdk.AccAddressFromBech32(TestOwnerAddress)
	require.NoError(t, err)

	msg, err := types.NewMsgSubmitTx(TestMessage, ibctesting.FirstConnectionID, TestOwnerAddress)
	require.NoError(t, err)

	require.Equal(t, []sdk.AccAddress{expSigner}, msg.GetSigners())
}
