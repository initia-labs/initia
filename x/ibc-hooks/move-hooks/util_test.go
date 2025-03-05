package move_hooks

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"

	nfttransfertypes "github.com/initia-labs/initia/v1/x/ibc/nft-transfer/types"
	movetypes "github.com/initia-labs/initia/v1/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"

	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"

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
	isMoveRouted, hookData, err := validateAndParseMemo(memo)
	require.True(t, isMoveRouted)
	require.NoError(t, err)
	require.Equal(t, HookData{
		Message: &movetypes.MsgExecute{
			ModuleAddress: "0x1",
			ModuleName:    "dex",
			FunctionName:  "swap",
			TypeArgs:      []string{"0x1::native_uinit::Coin", "0x1::native_uusdc::Coin"},
			Args:          [][]byte{argBz},
		},
		AsyncCallback: nil,
	}, hookData)
	require.NoError(t, validateReceiver(hookData.Message, "0x1::dex::swap"))

	// invalid receiver
	require.NoError(t, err)
	require.Error(t, validateReceiver(hookData.Message, "0x2::dex::swap"))

	isMoveRouted, _, err = validateAndParseMemo("hihi")
	require.False(t, isMoveRouted)
	require.NoError(t, err)
}

func Test_validateAndParseMemo_with_callback(t *testing.T) {

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
	isMoveRouted, hookData, err := validateAndParseMemo(memo)
	require.True(t, isMoveRouted)
	require.NoError(t, err)
	require.Equal(t, HookData{
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
	require.NoError(t, validateReceiver(hookData.Message, "0x1::dex::swap"))
}
