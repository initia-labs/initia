package move_hooks_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	nfttransfertypes "github.com/initia-labs/initia/x/ibc/nft-transfer/types"

	movetypes "github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"
)

func Test_OnReceivePacketWithoutMemo(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	_, _, addr := keyPubAddr()
	_, _, addr2 := keyPubAddr()

	data := transfertypes.FungibleTokenPacketData{
		Denom:    "foo",
		Amount:   "10000",
		Sender:   addr.String(),
		Receiver: addr2.String(),
		Memo:     "",
	}

	dataBz, err := json.Marshal(&data)
	require.NoError(t, err)

	ack := input.IBCHooksMiddleware.OnRecvPacket(ctx, channeltypes.Packet{
		Data: dataBz,
	}, addr)

	require.True(t, ack.Success())
}

func Test_onReceiveIcs20Packet_memo(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	_, _, addr := keyPubAddr()

	data := transfertypes.FungibleTokenPacketData{
		Denom:    "foo",
		Amount:   "10000",
		Sender:   addr.String(),
		Receiver: "0x1::Counter::increase",
		Memo: `{
			"move": {
				"message": {
					"module_address": "0x1",
					"module_name": "Counter",
					"function_name": "increase"
				}
			}
		}`,
	}

	dataBz, err := json.Marshal(&data)
	require.NoError(t, err)

	// failed to due to acl
	ack := input.IBCHooksMiddleware.OnRecvPacket(ctx, channeltypes.Packet{
		Data: dataBz,
	}, addr)
	require.False(t, ack.Success())

	// set acl
	require.NoError(t, input.IBCHooksKeeper.SetAllowed(ctx, movetypes.ConvertVMAddressToSDKAddress(vmtypes.StdAddress), true))

	// success
	ack = input.IBCHooksMiddleware.OnRecvPacket(ctx, channeltypes.Packet{
		Data: dataBz,
	}, addr)
	require.True(t, ack.Success())

	// check the contract state
	queryRes, err := input.MoveKeeper.ExecuteViewFunction(
		ctx,
		vmtypes.StdAddress,
		"Counter",
		"get",
		[]vmtypes.TypeTag{},
		[][]byte{},
	)
	require.NoError(t, err)
	require.Equal(t, "\"1\"", queryRes.Ret)
}

func Test_OnReceivePacket_ICS721(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	_, _, addr := keyPubAddr()
	_, _, addr2 := keyPubAddr()

	data := nfttransfertypes.NonFungibleTokenPacketData{
		ClassId:   "classId",
		ClassUri:  "classUri",
		ClassData: "classData",
		TokenIds:  []string{"tokenId"},
		TokenUris: []string{"tokenUri"},
		TokenData: []string{"tokenData"},
		Sender:    addr.String(),
		Receiver:  addr2.String(),
		Memo:      "",
	}

	dataBz, err := json.Marshal(&data)
	require.NoError(t, err)

	ack := input.IBCHooksMiddleware.OnRecvPacket(ctx, channeltypes.Packet{
		Data: dataBz,
	}, addr)

	require.True(t, ack.Success())
}

func Test_onReceivePacket_memo_ICS721(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	_, _, addr := keyPubAddr()

	data := nfttransfertypes.NonFungibleTokenPacketData{
		ClassId:   "classId",
		ClassUri:  "classUri",
		ClassData: "classData",
		TokenIds:  []string{"tokenId"},
		TokenUris: []string{"tokenUri"},
		TokenData: []string{"tokenData"},
		Sender:    addr.String(),
		Receiver:  "0x1::Counter::increase",
		Memo: `{
			"move": {
				"message": {
					"module_address": "0x1",
					"module_name": "Counter",
					"function_name": "increase"
				}
			}
		}`,
	}

	dataBz, err := json.Marshal(&data)
	require.NoError(t, err)

	// failed to due to acl
	ack := input.IBCHooksMiddleware.OnRecvPacket(ctx, channeltypes.Packet{
		Data: dataBz,
	}, addr)
	require.False(t, ack.Success())

	// set acl
	require.NoError(t, input.IBCHooksKeeper.SetAllowed(ctx, movetypes.ConvertVMAddressToSDKAddress(vmtypes.StdAddress), true))

	// success
	ack = input.IBCHooksMiddleware.OnRecvPacket(ctx, channeltypes.Packet{
		Data: dataBz,
	}, addr)
	require.True(t, ack.Success())

	// check the contract state
	queryRes, err := input.MoveKeeper.ExecuteViewFunction(
		ctx,
		vmtypes.StdAddress,
		"Counter",
		"get",
		[]vmtypes.TypeTag{},
		[][]byte{},
	)
	require.NoError(t, err)
	require.Equal(t, "\"1\"", queryRes.Ret)
}

func Test_onReceivePacket_memo_ICS721_Wasm(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	_, _, addr := keyPubAddr()

	data := nfttransfertypes.NonFungibleTokenPacketDataWasm{
		ClassId:   "classId",
		ClassUri:  "classUri",
		ClassData: "classData",
		TokenIds:  []string{"tokenId"},
		TokenUris: []string{"tokenUri"},
		TokenData: []string{"tokenData"},
		Sender:    addr.String(),
		Receiver:  "0x1::Counter::increase",
		Memo: `{
			"move": {
				"message": {
					"module_address": "0x1",
					"module_name": "Counter",
					"function_name": "increase"
				}
			}
		}`,
	}

	dataBz, err := json.Marshal(&data)
	require.NoError(t, err)

	// failed to due to acl
	ack := input.IBCHooksMiddleware.OnRecvPacket(ctx, channeltypes.Packet{
		SourcePort: "wasm.contract_address",
		Data:       dataBz,
	}, addr)
	require.False(t, ack.Success())

	// set acl
	require.NoError(t, input.IBCHooksKeeper.SetAllowed(ctx, movetypes.ConvertVMAddressToSDKAddress(vmtypes.StdAddress), true))

	// success
	ack = input.IBCHooksMiddleware.OnRecvPacket(ctx, channeltypes.Packet{
		SourcePort: "wasm.contract_address",
		Data:       dataBz,
	}, addr)
	require.True(t, ack.Success())

	// check the contract state
	queryRes, err := input.MoveKeeper.ExecuteViewFunction(
		ctx,
		vmtypes.StdAddress,
		"Counter",
		"get",
		[]vmtypes.TypeTag{},
		[][]byte{},
	)
	require.NoError(t, err)
	require.Equal(t, "\"1\"", queryRes.Ret)
}
