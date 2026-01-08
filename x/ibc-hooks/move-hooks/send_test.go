package move_hooks_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"cosmossdk.io/collections"

	"github.com/stretchr/testify/require"

	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	transfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	nfttransfertypes "github.com/initia-labs/initia/x/ibc/nft-transfer/types"

	movehooks "github.com/initia-labs/initia/x/ibc-hooks/move-hooks"
	vmtypes "github.com/initia-labs/movevm/types"
)

func Test_SendPacket_without_async_callback(t *testing.T) {
	ctx, keepers := createDefaultTestInput(t)
	ics4 := keepers.IBCHooksMiddleware.ICS4Middleware
	keepers.MockIBCMiddleware.setSequence(7)
	ics4.ICS4Wrapper = keepers.MockIBCMiddleware

	argBz, err := vmtypes.SerializeUint64(100)
	require.NoError(t, err)
	memo := fmt.Sprintf(
		`{"move":{"message":{"module_address":"0x1","module_name":"dex","function_name":"swap","type_args":["0x1::native_uinit::Coin"],"args":["%s"]}}}`,
		base64.StdEncoding.EncodeToString(argBz),
	)

	data := transfertypes.FungibleTokenPacketData{
		Denom:    "foo",
		Amount:   "10000",
		Sender:   "sender",
		Receiver: "receiver",
		Memo:     memo,
	}

	seq, err := ics4.SendPacket(ctx, &capabilitytypes.Capability{}, "transfer", "channel-0", clienttypes.Height{}, 0, data.GetBytes())
	require.NoError(t, err)
	require.Equal(t, uint64(7), seq)

	var sent transfertypes.FungibleTokenPacketData
	require.NoError(t, json.Unmarshal(keepers.MockIBCMiddleware.lastData, &sent))
	require.Equal(t, memo, sent.Memo)

	_, err = keepers.IBCHooksKeeper.GetAsyncCallback(ctx, "transfer", "channel-0", seq)
	require.ErrorIs(t, err, collections.ErrNotFound)
}

func Test_SendPacket_with_async_callback(t *testing.T) {
	ctx, keepers := createDefaultTestInput(t)
	ics4 := keepers.IBCHooksMiddleware.ICS4Middleware
	keepers.MockIBCMiddleware.setSequence(9)
	ics4.ICS4Wrapper = keepers.MockIBCMiddleware

	argBz, err := vmtypes.SerializeUint64(42)
	require.NoError(t, err)
	memo := fmt.Sprintf(
		`{"move":{"message":{"module_address":"0x1","module_name":"dex","function_name":"swap","type_args":["0x1::native_uinit::Coin"],"args":["%s"]},"async_callback":{"id":7,"module_address":"0x1","module_name":"Counter"}}}`,
		base64.StdEncoding.EncodeToString(argBz),
	)

	data := transfertypes.FungibleTokenPacketData{
		Denom:    "foo",
		Amount:   "10000",
		Sender:   "sender",
		Receiver: "receiver",
		Memo:     memo,
	}

	seq, err := ics4.SendPacket(ctx, &capabilitytypes.Capability{}, "transfer", "channel-1", clienttypes.Height{}, 0, data.GetBytes())
	require.NoError(t, err)
	require.Equal(t, uint64(9), seq)

	var sent transfertypes.FungibleTokenPacketData
	require.NoError(t, json.Unmarshal(keepers.MockIBCMiddleware.lastData, &sent))
	require.False(t, strings.Contains(sent.Memo, "async_callback"))
	require.True(t, strings.Contains(sent.Memo, "\"message\""))

	callbackBz, err := keepers.IBCHooksKeeper.GetAsyncCallback(ctx, "transfer", "channel-1", seq)
	require.NoError(t, err)

	var callback movehooks.AsyncCallback
	require.NoError(t, json.Unmarshal(callbackBz, &callback))
	require.Equal(t, movehooks.AsyncCallback{
		Id:            7,
		ModuleAddress: "0x1",
		ModuleName:    "Counter",
	}, callback)
}

func Test_SendPacket_ICS721_with_async_callback(t *testing.T) {
	ctx, keepers := createDefaultTestInput(t)
	ics4 := keepers.IBCHooksMiddleware.ICS4Middleware
	keepers.MockIBCMiddleware.setSequence(11)
	ics4.ICS4Wrapper = keepers.MockIBCMiddleware

	memo := `{"move":{"message":{"module_address":"0x1","module_name":"dex","function_name":"swap","type_args":[],"args":[]},"async_callback":{"id":7,"module_address":"0x1","module_name":"Counter"}}}`
	data := nfttransfertypes.NonFungibleTokenPacketData{
		ClassId:   "classId",
		ClassUri:  "classUri",
		ClassData: "classData",
		TokenIds:  []string{"tokenId"},
		TokenUris: []string{"tokenUri"},
		TokenData: []string{"tokenData"},
		Sender:    "sender",
		Receiver:  "receiver",
		Memo:      memo,
	}

	seq, err := ics4.SendPacket(ctx, &capabilitytypes.Capability{}, "nft-transfer", "channel-2", clienttypes.Height{}, 0, data.GetBytes())
	require.NoError(t, err)
	require.Equal(t, uint64(11), seq)

	var sent nfttransfertypes.NonFungibleTokenPacketData
	require.NoError(t, json.Unmarshal(keepers.MockIBCMiddleware.lastData, &sent))
	require.False(t, strings.Contains(sent.Memo, "async_callback"))
	require.True(t, strings.Contains(sent.Memo, "\"message\""))

	callbackBz, err := keepers.IBCHooksKeeper.GetAsyncCallback(ctx, "nft-transfer", "channel-2", seq)
	require.NoError(t, err)

	var callback movehooks.AsyncCallback
	require.NoError(t, json.Unmarshal(callbackBz, &callback))
	require.Equal(t, movehooks.AsyncCallback{
		Id:            7,
		ModuleAddress: "0x1",
		ModuleName:    "Counter",
	}, callback)
}

func Test_SendPacket_not_routed(t *testing.T) {
	ctx, keepers := createDefaultTestInput(t)
	ics4 := keepers.IBCHooksMiddleware.ICS4Middleware
	keepers.MockIBCMiddleware.setSequence(5)
	ics4.ICS4Wrapper = keepers.MockIBCMiddleware

	data := transfertypes.FungibleTokenPacketData{
		Denom:    "foo",
		Amount:   "10000",
		Sender:   "sender",
		Receiver: "receiver",
		Memo:     "{\"memo\":true}",
	}

	seq, err := ics4.SendPacket(ctx, &capabilitytypes.Capability{}, "transfer", "channel-9", clienttypes.Height{}, 0, data.GetBytes())
	require.NoError(t, err)
	require.Equal(t, uint64(5), seq)

	var sent transfertypes.FungibleTokenPacketData
	require.NoError(t, json.Unmarshal(keepers.MockIBCMiddleware.lastData, &sent))
	require.Equal(t, data.Memo, sent.Memo)

	_, err = keepers.IBCHooksKeeper.GetAsyncCallback(ctx, "transfer", "channel-9", seq)
	require.ErrorIs(t, err, collections.ErrNotFound)
}
