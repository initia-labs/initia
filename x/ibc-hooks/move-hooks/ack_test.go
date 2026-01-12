package move_hooks_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	nfttransfertypes "github.com/initia-labs/initia/x/ibc/nft-transfer/types"

	movehooks "github.com/initia-labs/initia/x/ibc-hooks/move-hooks"
	movetypes "github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"
)

func Test_OnAckPacket(t *testing.T) {
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

	ackBz, err := json.Marshal(channeltypes.NewResultAcknowledgement([]byte{byte(1)}))
	require.NoError(t, err)

	err = input.IBCHooksMiddleware.OnAcknowledgementPacket(ctx, channeltypes.Packet{
		Data: dataBz,
	}, ackBz, addr)
	require.NoError(t, err)
}

func Test_onAckPacket_memo(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	_, _, addr := keyPubAddr()
	sourcePort := "transfer"
	sourceChannel := "channel-0"
	sequence := uint64(1)

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

	successAckBz := channeltypes.NewResultAcknowledgement([]byte{byte(1)}).Acknowledgement()
	failedAckBz := channeltypes.NewErrorAcknowledgement(errors.New("failed")).Acknowledgement()
	callbackBz, err := json.Marshal(movehooks.AsyncCallback{
		Id:            99,
		ModuleAddress: "0x1",
		ModuleName:    "Counter",
	})
	require.NoError(t, err)
	require.NoError(t, input.IBCHooksKeeper.SetAsyncCallback(ctx, sourcePort, sourceChannel, sequence, callbackBz))

	// hook should not be called to due to acl
	err = input.IBCHooksMiddleware.OnAcknowledgementPacket(ctx, channeltypes.Packet{
		Data:          dataBz,
		SourcePort:    sourcePort,
		SourceChannel: sourceChannel,
		Sequence:      sequence,
	}, successAckBz, addr)
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
	require.NoError(t, input.IBCHooksKeeper.SetAsyncCallback(ctx, sourcePort, sourceChannel, sequence, callbackBz))

	// success with success ack
	err = input.IBCHooksMiddleware.OnAcknowledgementPacket(ctx, channeltypes.Packet{
		Data:          dataBz,
		SourcePort:    sourcePort,
		SourceChannel: sourceChannel,
		Sequence:      sequence,
	}, successAckBz, addr)
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

	// success with failed ack
	require.NoError(t, input.IBCHooksKeeper.SetAsyncCallback(ctx, sourcePort, sourceChannel, sequence, callbackBz))
	err = input.IBCHooksMiddleware.OnAcknowledgementPacket(ctx, channeltypes.Packet{
		Data:          dataBz,
		SourcePort:    sourcePort,
		SourceChannel: sourceChannel,
		Sequence:      sequence,
	}, failedAckBz, addr)
	require.NoError(t, err)

	queryRes, _, err = input.MoveKeeper.ExecuteViewFunction(
		ctx,
		vmtypes.StdAddress,
		"Counter",
		"get",
		[]vmtypes.TypeTag{},
		[][]byte{},
	)
	require.NoError(t, err)
	require.Equal(t, "\"100\"", queryRes.Ret)
}

func Test_OnAckPacket_ICS721(t *testing.T) {
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

	ackBz, err := json.Marshal(channeltypes.NewResultAcknowledgement([]byte{byte(1)}))
	require.NoError(t, err)

	err = input.IBCHooksMiddleware.OnAcknowledgementPacket(ctx, channeltypes.Packet{
		Data: dataBz,
	}, ackBz, addr)
	require.NoError(t, err)
}

func Test_onAckPacket_memo_ICS721(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	_, _, addr := keyPubAddr()
	sourcePort := "nft-transfer"
	sourceChannel := "channel-1"
	sequence := uint64(1)

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

	successAckBz := channeltypes.NewResultAcknowledgement([]byte{byte(1)}).Acknowledgement()
	failedAckBz := channeltypes.NewErrorAcknowledgement(errors.New("failed")).Acknowledgement()
	callbackBz, err := json.Marshal(movehooks.AsyncCallback{
		Id:            99,
		ModuleAddress: "0x1",
		ModuleName:    "Counter",
	})
	require.NoError(t, err)
	require.NoError(t, input.IBCHooksKeeper.SetAsyncCallback(ctx, sourcePort, sourceChannel, sequence, callbackBz))

	// hook should not be called to due to acl
	err = input.IBCHooksMiddleware.OnAcknowledgementPacket(ctx, channeltypes.Packet{
		Data:          dataBz,
		SourcePort:    sourcePort,
		SourceChannel: sourceChannel,
		Sequence:      sequence,
	}, successAckBz, addr)
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
	require.NoError(t, input.IBCHooksKeeper.SetAsyncCallback(ctx, sourcePort, sourceChannel, sequence, callbackBz))

	// success with success ack
	err = input.IBCHooksMiddleware.OnAcknowledgementPacket(ctx, channeltypes.Packet{
		Data:          dataBz,
		SourcePort:    sourcePort,
		SourceChannel: sourceChannel,
		Sequence:      sequence,
	}, successAckBz, addr)
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

	// success with failed ack
	require.NoError(t, input.IBCHooksKeeper.SetAsyncCallback(ctx, sourcePort, sourceChannel, sequence, callbackBz))
	err = input.IBCHooksMiddleware.OnAcknowledgementPacket(ctx, channeltypes.Packet{
		Data:          dataBz,
		SourcePort:    sourcePort,
		SourceChannel: sourceChannel,
		Sequence:      sequence,
	}, failedAckBz, addr)
	require.NoError(t, err)

	queryRes, _, err = input.MoveKeeper.ExecuteViewFunction(
		ctx,
		vmtypes.StdAddress,
		"Counter",
		"get",
		[]vmtypes.TypeTag{},
		[][]byte{},
	)
	require.NoError(t, err)
	require.Equal(t, "\"100\"", queryRes.Ret)
}
