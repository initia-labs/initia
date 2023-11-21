package types

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"

	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
)

// define constants used for testing
const (
	validPort        = "testportid"
	invalidPort      = "(invalidport1)"
	invalidShortPort = "p"
	// 195 characters
	invalidLongPort = "Lorem ipsum dolor sit amet, consectetur adipiscing elit. Duis eros neque, ultricies vel ligula ac, convallis porttitor elit. Maecenas tincidunt turpis elit, vel faucibus nisl pellentesque sodales"

	validChannel        = "testchannel"
	invalidChannel      = "(invalidchannel1)"
	invalidShortChannel = "invalid"
	invalidLongChannel  = "invalidlongchannelinvalidlongchannelinvalidlongchannelinvalidlongchannel"
)

var (
	addr1     = sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()
	addr2     = sdk.AccAddress("testaddr2").String()
	emptyAddr string

	validClassId      = "0x123::nft_store::Extension"
	validTokenIds     = []string{"1", "2", "3"}
	ibcClassId        = "ibc/F54C67869D6548E0078EA5AD443B858272B04939E6AD9108E970D04887694437"
	invalidIBCClassId = "ibc/F54C67869D6548E007"
	emptyClassId      = ""
	emptyTokenIds     = []string{}
	emptyTokenIds2    = []string{"", "", ""}

	timeoutHeight = clienttypes.NewHeight(0, 10)
)

// TestMsgTransferRoute tests Route for MsgTransfer
func TestMsgTransferRoute(t *testing.T) {
	msg := NewMsgTransfer(validPort, validChannel, validClassId, validTokenIds, addr1, addr2, timeoutHeight, 0, "")

	require.Equal(t, RouterKey, msg.Route())
}

// TestMsgTransferType tests Type for MsgTransfer
func TestMsgTransferType(t *testing.T) {
	msg := NewMsgTransfer(validPort, validChannel, validClassId, validTokenIds, addr1, addr2, timeoutHeight, 0, "")

	require.Equal(t, "nft_transfer", msg.Type())
}

func TestMsgTransferGetSignBytes(t *testing.T) {
	msg := NewMsgTransfer(validPort, validChannel, validClassId, validTokenIds, addr1, addr2, timeoutHeight, 0, "")
	expected := fmt.Sprintf(`{"type":"nft-transfer/MsgTransfer","value":{"class_id":"%s","receiver":"%s","sender":"%s","source_channel":"testchannel","source_port":"testportid","timeout_height":{"revision_height":"10"},"token_ids":["1","2","3"]}}`, validClassId, addr2, addr1)
	require.NotPanics(t, func() {
		res := msg.GetSignBytes()
		require.Equal(t, expected, string(res))
	})
}

// TestMsgTransferValidation tests ValidateBasic for MsgTransfer
func TestMsgTransferValidation(t *testing.T) {
	testCases := []struct {
		name    string
		msg     *MsgTransfer
		expPass bool
	}{
		{"valid msg with base denom", NewMsgTransfer(validPort, validChannel, validClassId, validTokenIds, addr1, addr2, timeoutHeight, 0, ""), true},
		{"valid msg with trace hash", NewMsgTransfer(validPort, validChannel, ibcClassId, validTokenIds, addr1, addr2, timeoutHeight, 0, ""), true},
		{"invalid ibc denom", NewMsgTransfer(validPort, validChannel, invalidIBCClassId, validTokenIds, addr1, addr2, timeoutHeight, 0, ""), false},
		{"too short port id", NewMsgTransfer(invalidShortPort, validChannel, validClassId, validTokenIds, addr1, addr2, timeoutHeight, 0, ""), false},
		{"too long port id", NewMsgTransfer(invalidLongPort, validChannel, validClassId, validTokenIds, addr1, addr2, timeoutHeight, 0, ""), false},
		{"port id contains non-alpha", NewMsgTransfer(invalidPort, validChannel, validClassId, validTokenIds, addr1, addr2, timeoutHeight, 0, ""), false},
		{"too short channel id", NewMsgTransfer(validPort, invalidShortChannel, validClassId, validTokenIds, addr1, addr2, timeoutHeight, 0, ""), false},
		{"too long channel id", NewMsgTransfer(validPort, invalidLongChannel, validClassId, validTokenIds, addr1, addr2, timeoutHeight, 0, ""), false},
		{"channel id contains non-alpha", NewMsgTransfer(validPort, invalidChannel, validClassId, validTokenIds, addr1, addr2, timeoutHeight, 0, ""), false},
		{"empty class id", NewMsgTransfer(validPort, validChannel, emptyClassId, validTokenIds, addr1, addr2, timeoutHeight, 0, ""), false},
		{"empty token ids", NewMsgTransfer(validPort, validChannel, validClassId, emptyTokenIds, addr1, addr2, timeoutHeight, 0, ""), false},
		{"empty token ids 2", NewMsgTransfer(validPort, validChannel, validClassId, emptyTokenIds2, addr1, addr2, timeoutHeight, 0, ""), false},
		{"missing sender address", NewMsgTransfer(validPort, validChannel, validClassId, validTokenIds, emptyAddr, addr2, timeoutHeight, 0, ""), false},
		{"missing recipient address", NewMsgTransfer(validPort, validChannel, validClassId, validTokenIds, addr1, "", timeoutHeight, 0, ""), false},
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

// TestMsgTransferGetSigners tests GetSigners for MsgTransfer
func TestMsgTransferGetSigners(t *testing.T) {
	addr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())

	msg := NewMsgTransfer(validPort, validChannel, validClassId, validTokenIds, addr.String(), addr2, timeoutHeight, 0, "")
	res := msg.GetSigners()

	require.Equal(t, []sdk.AccAddress{addr}, res)
}
