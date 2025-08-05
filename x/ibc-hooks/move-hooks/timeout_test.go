package move_hooks_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	nfttransfertypes "github.com/initia-labs/initia/x/ibc/nft-transfer/types"

	movetypes "github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"
)

func Test_OnTimeoutPacket(t *testing.T) {
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

	err = input.IBCHooksMiddleware.OnTimeoutPacket(ctx, transfertypes.V1, channeltypes.Packet{
		Data: dataBz,
	}, addr)
	require.NoError(t, err)
}

func Test_onTimeoutPacket_memo(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	_, _, addr := keyPubAddr()

	data := transfertypes.FungibleTokenPacketData{
		Denom:    "foo",
		Amount:   "10000",
		Sender:   addr.String(),
		Receiver: "0x1::Counter::increase",
		Memo: `{
			"move": {
				"async_callback": {
					"id": 99,
					"module_address": "0x1",
					"module_name": "Counter"
				}
			}
		}`,
	}

	dataBz, err := json.Marshal(&data)
	require.NoError(t, err)

	// hook should not be called to due to acl
	err = input.IBCHooksMiddleware.OnTimeoutPacket(ctx, transfertypes.V1, channeltypes.Packet{
		Data: dataBz,
	}, addr)
	require.NoError(t, err)

	// check the contract state
	queryRes, _, err := input.MoveKeeper.ExecuteViewFunction(
		ctx,
		vmtypes.StdAddress,
		"Counter",
		"get",
		[]vmtypes.TypeTag{},
		[][]byte{},
	)
	require.NoError(t, err)
	require.Equal(t, "\"0\"", queryRes.Ret)

	// set acl
	require.NoError(t, input.IBCHooksKeeper.SetAllowed(ctx, movetypes.ConvertVMAddressToSDKAddress(vmtypes.StdAddress), true))

	// success
	err = input.IBCHooksMiddleware.OnTimeoutPacket(ctx, transfertypes.V1, channeltypes.Packet{
		Data: dataBz,
	}, addr)
	require.NoError(t, err)

	// check the contract state; increased by 99
	queryRes, _, err = input.MoveKeeper.ExecuteViewFunction(
		ctx,
		vmtypes.StdAddress,
		"Counter",
		"get",
		[]vmtypes.TypeTag{},
		[][]byte{},
	)
	require.NoError(t, err)
	require.Equal(t, "\"99\"", queryRes.Ret)
}

func Test_OnTimeoutPacket_ICS721(t *testing.T) {
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

	err = input.IBCHooksMiddleware.OnTimeoutPacket(ctx, nfttransfertypes.V1, channeltypes.Packet{
		Data: dataBz,
	}, addr)
	require.NoError(t, err)
}

func Test_onTimeoutPacket_memo_ICS721(t *testing.T) {
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
				"async_callback": {
					"id": 99,
					"module_address": "0x1",
					"module_name": "Counter"
				}
			}
		}`,
	}

	dataBz, err := json.Marshal(&data)
	require.NoError(t, err)

	// hook should not be called to due to acl
	err = input.IBCHooksMiddleware.OnTimeoutPacket(ctx, nfttransfertypes.V1, channeltypes.Packet{
		Data: dataBz,
	}, addr)
	require.NoError(t, err)

	// check the contract state
	queryRes, _, err := input.MoveKeeper.ExecuteViewFunction(
		ctx,
		vmtypes.StdAddress,
		"Counter",
		"get",
		[]vmtypes.TypeTag{},
		[][]byte{},
	)
	require.NoError(t, err)
	require.Equal(t, "\"0\"", queryRes.Ret)

	// set acl
	require.NoError(t, input.IBCHooksKeeper.SetAllowed(ctx, movetypes.ConvertVMAddressToSDKAddress(vmtypes.StdAddress), true))

	// success
	err = input.IBCHooksMiddleware.OnTimeoutPacket(ctx, nfttransfertypes.V1, channeltypes.Packet{
		Data: dataBz,
	}, addr)
	require.NoError(t, err)

	// check the contract state; increased by 99 if ack is success
	queryRes, _, err = input.MoveKeeper.ExecuteViewFunction(
		ctx,
		vmtypes.StdAddress,
		"Counter",
		"get",
		[]vmtypes.TypeTag{},
		[][]byte{},
	)
	require.NoError(t, err)
	require.Equal(t, "\"99\"", queryRes.Ret)
}
