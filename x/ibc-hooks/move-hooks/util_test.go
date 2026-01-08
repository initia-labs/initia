package move_hooks

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"

	nfttransfertypes "github.com/initia-labs/initia/x/ibc/nft-transfer/types"
	movetypes "github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"

	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"

	"github.com/stretchr/testify/require"
)

func Test_isIcs20Packet(t *testing.T) {
	transferMsg := transfertypes.NewFungibleTokenPacketData("denom", "1000000", "0x1", "0x2", "memo")
	bz, err := json.Marshal(transferMsg)
	require.NoError(t, err)

	ok, _transferMsg := isIcs20Packet(bz)
	require.True(t, ok)
	require.Equal(t, transferMsg, _transferMsg)

	nftTransferMsg := nfttransfertypes.NewNonFungibleTokenPacketData("class_id", "uri", "data", []string{"1", "2", "3"}, []string{"uri1", "uri2", "uri3"}, []string{"data1", "data2", "data3"}, "sender", "receiver", "memo")
	bz, err = json.Marshal(nftTransferMsg)
	require.NoError(t, err)

	ok, _ = isIcs20Packet(bz)
	require.False(t, ok)
}

func Test_isIcs721Packet(t *testing.T) {
	nftTransferMsg := nfttransfertypes.NewNonFungibleTokenPacketData("class_id", "uri", "data", []string{"1", "2", "3"}, []string{"uri1", "uri2", "uri3"}, []string{"data1", "data2", "data3"}, "sender", "receiver", "memo")
	ok, _nftTransferMsg := isIcs721Packet(nftTransferMsg.GetBytes())
	require.True(t, ok)
	require.Equal(t, nftTransferMsg, _nftTransferMsg)

	// invalid
	transferMsg := transfertypes.NewFungibleTokenPacketData("denom", "1000000", "0x1", "0x2", "memo")
	ok, _ = isIcs721Packet(transferMsg.GetBytes())
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
	hookData, isMoveRouted, err := parseHookData(memo)
	require.NoError(t, err)
	require.True(t, isMoveRouted)
	require.NotNil(t, hookData)
	require.Equal(t, &HookData{
		Message: &movetypes.MsgExecute{
			ModuleAddress: "0x1",
			ModuleName:    "dex",
			FunctionName:  "swap",
			TypeArgs:      []string{"0x1::native_uinit::Coin", "0x1::native_uusdc::Coin"},
			Args:          [][]byte{argBz},
		},
		AsyncCallback: nil,
	}, hookData)
	functionIdentifier := fmt.Sprintf("%s::%s::%s", hookData.Message.ModuleAddress, hookData.Message.ModuleName, hookData.Message.FunctionName)
	require.NoError(t, validateReceiver(functionIdentifier, "0x1::dex::swap", ac))

	// invalid receiver
	require.NoError(t, err)
	require.Error(t, validateReceiver(functionIdentifier, "0x2::dex::swap", ac))

	hookData, isMoveRouted, err = parseHookData("hihi")
	require.NoError(t, err)
	require.False(t, isMoveRouted)
	require.Nil(t, hookData)
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
	hookData, isMoveRouted, err := parseHookData(memo)
	require.NoError(t, err)
	require.True(t, isMoveRouted)
	require.NotNil(t, hookData)
	require.Equal(t, &HookData{
		Message: &movetypes.MsgExecute{
			ModuleAddress: "0x1",
			ModuleName:    "dex",
			FunctionName:  "swap",
			TypeArgs:      []string{"0x1::native_uinit::Coin", "0x1::native_uusdc::Coin"},
			Args:          [][]byte{argBz},
		},
		AsyncCallback: &AsyncCallback{
			Id:            1,
			ModuleAddress: "0x1",
			ModuleName:    "dex",
		},
	}, hookData)
	functionIdentifier := fmt.Sprintf("%s::%s::%s", hookData.Message.ModuleAddress, hookData.Message.ModuleName, hookData.Message.FunctionName)
	require.NoError(t, validateReceiver(functionIdentifier, "0x1::dex::swap", ac))
}

func Test_validateReceiver(t *testing.T) {
	ac := authcodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())

	hookData := HookData{
		Message: &movetypes.MsgExecute{
			ModuleAddress: "0x1",
			ModuleName:    "dex",
			FunctionName:  "swap",
			TypeArgs:      []string{},
			Args:          [][]byte{},
		},
	}

	functionIdentifier := fmt.Sprintf("%s::%s::%s", hookData.Message.ModuleAddress, hookData.Message.ModuleName, hookData.Message.FunctionName)

	require.NoError(t, validateReceiver(functionIdentifier, "cosmos14ve5y0rgh6aaa45k0g99ctj4la0hw3prr6h7e57mzqx86eg63r6s9yz06a", ac))
	require.NoError(t, validateReceiver(functionIdentifier, "0x1::dex::swap", ac))
}
