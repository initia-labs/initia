package types_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	icatypes "github.com/cosmos/ibc-go/v8/modules/apps/27-interchain-accounts/types"
	ibctesting "github.com/initia-labs/initia/x/ibc/testing"

	"github.com/cosmos/cosmos-sdk/codec/address"
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
		Amount:      sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, math.NewInt(100))),
	}
)

// TestMsgRegisterAccountValidate tests Validate for MsgRegisterAccount
func TestMsgRegisterAccountValidate(t *testing.T) {
	testCases := []struct {
		name    string
		msg     *types.MsgRegisterAccount
		expPass bool
	}{
		{"success", types.NewMsgRegisterAccount(TestOwnerAddress, ibctesting.FirstConnectionID, TestVersion), true},
		{"owner address is empty", types.NewMsgRegisterAccount("", ibctesting.FirstConnectionID, TestVersion), false},
		{"owner address is invalid", types.NewMsgRegisterAccount("invalid_address", ibctesting.FirstConnectionID, TestVersion), false},
	}

	ac := address.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	for i, tc := range testCases {
		err := tc.msg.Validate(ac)
		if tc.expPass {
			require.NoError(t, err, "valid test case %d failed: %s", i, tc.name)
		} else {
			require.Error(t, err, "invalid test case %d passed: %s", i, tc.name)
		}
	}
}

// TestMsgSubmitTxValidate tests Validate for MsgSubmitTx
func TestMsgSubmitTxValidate(t *testing.T) {
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

	ac := address.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	for i, tc := range testCases {
		msg, _ = types.NewMsgSubmitTx(TestMessage, ibctesting.FirstConnectionID, TestOwnerAddress)

		tc.malleate()

		err := msg.Validate(ac)
		if tc.expPass {
			require.NoError(t, err, "valid test case %d failed: %s", i, tc.name)
		} else {
			require.Error(t, err, "invalid test case %d passed: %s", i, tc.name)
		}
	}
}
