package move_hooks_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	nfttransfertypes "github.com/initia-labs/initia/x/ibc/nft-transfer/types"

	ibchookstypes "github.com/initia-labs/initia/x/ibc-hooks/types"
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
		Memo: `{
			"move": {
				"message": null
			}
		}`,
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
		SourcePort:         "transfer",
		SourceChannel:      "channel-0",
		DestinationPort:    "transfer",
		DestinationChannel: "channel-0",
		Data:               dataBz,
	}, addr)
	require.True(t, ack.Success())

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
	require.Equal(t, "\"1\"", queryRes.Ret)
}

func Test_TransferFunds_Option_None(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	_, _, addr := keyPubAddr()
	coin := sdk.NewCoin("uinit", sdkmath.NewInt(10000))
	input.Faucet.Fund(ctx, addr, coin)

	// if we call 0x1::hook_sender::send_funds without storing transfer funds, it should return an error
	err := input.MoveKeeper.ExecuteEntryFunctionJSON(
		ctx,
		vmtypes.StdAddress,
		vmtypes.StdAddress,
		"hook_sender",
		"send_funds",
		[]vmtypes.TypeTag{},
		[]string{
			fmt.Sprintf(`"0x%x"`, addr),
		},
	)
	require.ErrorContains(t, err, "code=1000")
}

func Test_onReceiveIcs20Packet_memo_and_transfer_funds(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	_, _, addr := keyPubAddr()
	_, _, addr2 := keyPubAddr()

	addr2VmAddress := movetypes.ConvertSDKAddressToVMAddress(addr2)
	addr2VmAddressBz, err := addr2VmAddress.BcsSerialize()
	require.NoError(t, err)

	data := transfertypes.FungibleTokenPacketData{
		Denom:    bondDenom,
		Amount:   "10000",
		Sender:   addr.String(),
		Receiver: "0x1::hook_sender::send_funds",
		Memo: fmt.Sprintf(`{
			"move": {
				"message": {
					"module_address": "0x1",
					"module_name": "hook_sender",
					"function_name": "send_funds",
					"args": ["%s"]
				}
			}
		}`, base64.StdEncoding.EncodeToString(addr2VmAddressBz)),
	}

	dataBz, err := json.Marshal(&data)
	require.NoError(t, err)

	// set acl
	require.NoError(t, input.IBCHooksKeeper.SetAllowed(ctx, movetypes.ConvertVMAddressToSDKAddress(vmtypes.StdAddress), true))

	packet := channeltypes.Packet{
		SourcePort:         "transfer",
		SourceChannel:      "channel-0",
		DestinationPort:    "transfer",
		DestinationChannel: "channel-0",
		Data:               dataBz,
	}

	denom := ibchookstypes.GetReceivedTokenDenom(packet, data)

	beforeBalance, err := input.MoveKeeper.MoveBankKeeper().GetBalance(ctx, addr2, denom)
	require.NoError(t, err)
	require.Zero(t, beforeBalance.Int64())

	// success
	ack := input.TransferStack.OnRecvPacket(ctx, packet, addr)
	require.True(t, ack.Success())

	afterBalance, err := input.MoveKeeper.MoveBankKeeper().GetBalance(ctx, addr2, denom)
	require.NoError(t, err)
	require.Equal(t, int64(10000), afterBalance.Int64())
}

func Test_onReceiveIcs20Packet_memo_with_hashed_receiver(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	_, _, addr := keyPubAddr()

	data := transfertypes.FungibleTokenPacketData{
		Denom:    "foo",
		Amount:   "10000",
		Sender:   addr.String(),
		Receiver: "cosmos1w53w03gsuvwazjx7jkq530q2l4e496m00hcx2rkj43gvl4vx9zrs65nfw5",
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
		SourcePort:         "transfer",
		SourceChannel:      "channel-0",
		DestinationPort:    "transfer",
		DestinationChannel: "channel-0",
		Data:               dataBz,
	}, addr)
	require.True(t, ack.Success())

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
	queryRes, _, err := input.MoveKeeper.ExecuteViewFunction(
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
