package move_hooks_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"

	move_hooks "github.com/initia-labs/initia/x/ibc-hooks/move-hooks"
	nfttransfertypes "github.com/initia-labs/initia/x/ibc/nft-transfer/types"
	movetypes "github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"

	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	"github.com/stretchr/testify/require"
)

func Test_isIcs20Packet(t *testing.T) {
	transferMsg := transfertypes.NewFungibleTokenPacketData("denom", "1000000", "0x1", "0x2", "memo")
	bz, err := json.Marshal(transferMsg)
	require.NoError(t, err)

	ok, internalRep := move_hooks.IsIcs20Packet(bz, transfertypes.V1, "")
	require.True(t, ok)
	// Check that the internal representation matches the original data
	require.Equal(t, "denom", internalRep.Token.Denom.Base)
	require.Equal(t, "1000000", internalRep.Token.Amount)
	require.Equal(t, "0x1", internalRep.Sender)
	require.Equal(t, "0x2", internalRep.Receiver)
	require.Equal(t, "memo", internalRep.Memo)

	nftTransferMsg := nfttransfertypes.NewNonFungibleTokenPacketData("class_id", "uri", "data", []string{"1", "2", "3"}, []string{"uri1", "uri2", "uri3"}, []string{"data1", "data2", "data3"}, "sender", "receiver", "memo")
	bz, err = json.Marshal(nftTransferMsg)
	require.NoError(t, err)

	ok, _ = move_hooks.IsIcs20Packet(bz, transfertypes.V1, "")
	require.False(t, ok)
}

func Test_isIcs721Packet(t *testing.T) {
	nftTransferMsg := nfttransfertypes.NewNonFungibleTokenPacketData("class_id", "uri", "data", []string{"1", "2", "3"}, []string{"uri1", "uri2", "uri3"}, []string{"data1", "data2", "data3"}, "sender", "receiver", "memo")
	ok, _nftTransferMsg := move_hooks.IsIcs721Packet(nftTransferMsg.GetBytes(), nfttransfertypes.V1, "")
	require.True(t, ok)
	require.Equal(t, nftTransferMsg, _nftTransferMsg)

	// invalid
	transferMsg := transfertypes.NewFungibleTokenPacketData("denom", "1000000", "0x1", "0x2", "memo")
	ok, _ = move_hooks.IsIcs721Packet(transferMsg.GetBytes(), nfttransfertypes.V1, "")
	require.False(t, ok)
}

func Test_validateAndParseMemo_without_callback(t *testing.T) {
	ac := authcodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())

	argBz, err := vmtypes.SerializeUint64(100)
	require.NoError(t, err)

	memo := fmt.Sprintf(
		`{
			"move" : {
				"message": {
					"module_address": "0x1",
					"module_name": "dex",
					"function_name": "swap",
					"type_args": ["0x1::native_uinit::Coin", "0x1::native_uusdc::Coin"],
					"args": ["%s"]
				}
			}
		}`, base64.StdEncoding.EncodeToString(argBz))
	isMoveRouted, hookData, err := move_hooks.ValidateAndParseMemo(memo)
	require.True(t, isMoveRouted)
	require.NoError(t, err)
	require.Equal(t, move_hooks.HookData{
		Message: &movetypes.MsgExecute{
			ModuleAddress: "0x1",
			ModuleName:    "dex",
			FunctionName:  "swap",
			TypeArgs:      []string{"0x1::native_uinit::Coin", "0x1::native_uusdc::Coin"},
			Args:          [][]byte{argBz},
		},
		AsyncCallback: nil,
	}, hookData)
	require.NoError(t, move_hooks.ValidateReceiver(hookData.Message, "0x1::dex::swap", ac))

	// invalid receiver
	require.NoError(t, err)
	require.Error(t, move_hooks.ValidateReceiver(hookData.Message, "0x2::dex::swap", ac))

	isMoveRouted, _, err = move_hooks.ValidateAndParseMemo("hihi")
	require.False(t, isMoveRouted)
	require.NoError(t, err)
}

func Test_validateAndParseMemo_with_callback(t *testing.T) {
	ac := authcodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())

	argBz, err := vmtypes.SerializeUint64(100)
	require.NoError(t, err)

	memo := fmt.Sprintf(
		`{
			"move" : {
				"message": {
					"module_address": "0x1",
					"module_name": "dex",
					"function_name": "swap",
					"type_args": ["0x1::native_uinit::Coin", "0x1::native_uusdc::Coin"],
					"args": ["%s"]
				},
				"async_callback": {
					"id": 1,
					"module_address": "0x1",
					"module_name": "dex"
				}
			}			
		}`, base64.StdEncoding.EncodeToString(argBz))
	isMoveRouted, hookData, err := move_hooks.ValidateAndParseMemo(memo)
	require.True(t, isMoveRouted)
	require.NoError(t, err)
	require.Equal(t, move_hooks.HookData{
		Message: &movetypes.MsgExecute{
			ModuleAddress: "0x1",
			ModuleName:    "dex",
			FunctionName:  "swap",
			TypeArgs:      []string{"0x1::native_uinit::Coin", "0x1::native_uusdc::Coin"},
			Args:          [][]byte{argBz},
		},
		AsyncCallback: &move_hooks.AsyncCallback{
			Id:            1,
			ModuleAddress: "0x1",
			ModuleName:    "dex",
		},
	}, hookData)
	require.NoError(t, move_hooks.ValidateReceiver(hookData.Message, "0x1::dex::swap", ac))
}

func Test_ValidateReceiver(t *testing.T) {
	ac := authcodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())

	hookData := move_hooks.HookData{
		Message: &movetypes.MsgExecute{
			ModuleAddress: "0x1",
			ModuleName:    "dex",
			FunctionName:  "swap",
			TypeArgs:      []string{},
			Args:          [][]byte{},
		},
	}

	require.NoError(t, move_hooks.ValidateReceiver(hookData.Message, "cosmos14ve5y0rgh6aaa45k0g99ctj4la0hw3prr6h7e57mzqx86eg63r6s9yz06a", ac))
	require.NoError(t, move_hooks.ValidateReceiver(hookData.Message, "0x1::dex::swap", ac))
}
