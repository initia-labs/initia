package keeper_test

import (
	"testing"

	"cosmossdk.io/collections"
	"github.com/cometbft/cometbft/crypto/secp256k1"

	"github.com/stretchr/testify/require"

	dbm "github.com/cosmos/cosmos-db"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/x/ibc/perm/keeper"
	"github.com/initia-labs/initia/x/ibc/perm/types"
)

func Test_QueryPermissionedRelayer(t *testing.T) {
	ctx, k := _createTestInput(t, dbm.NewMemDB())

	portID := "port-123"
	channelID := "channel-123"
	pubKey := secp256k1.GenPrivKey().PubKey()
	addr := sdk.AccAddress(pubKey.Address())

	// set relayer
	require.NoError(t, k.PermissionedRelayers.Set(ctx, collections.Join(portID, channelID), addr))

	// set channel relayer via msg handler
	queryServer := keeper.NewQueryServer(k)
	res, err := queryServer.PermissionedRelayer(ctx, &types.QueryPermissionedRelayerRequest{
		PortId:    portID,
		ChannelId: channelID,
	})
	require.NoError(t, err)
	require.Equal(t, addr.String(), res.PermissionedRelayer.Relayer)

	_, err = queryServer.PermissionedRelayer(ctx, &types.QueryPermissionedRelayerRequest{
		PortId:    portID,
		ChannelId: channelID + "4",
	})
	require.Error(t, err)

}

func Test_QueryPermissionedRelayers(t *testing.T) {
	ctx, k := _createTestInput(t, dbm.NewMemDB())

	portID1 := "port-123"
	channelID1 := "channel-123"
	portID2 := "port-456"
	channelID2 := "channel-456"
	pubKey1 := secp256k1.GenPrivKey().PubKey()
	pubKey2 := secp256k1.GenPrivKey().PubKey()
	addr1 := sdk.AccAddress(pubKey1.Address())
	addr2 := sdk.AccAddress(pubKey2.Address())

	// set relayers
	require.NoError(t, k.PermissionedRelayers.Set(ctx, collections.Join(portID1, channelID1), addr1))
	require.NoError(t, k.PermissionedRelayers.Set(ctx, collections.Join(portID2, channelID2), addr2))

	// set channel relayer via msg handler
	queryServer := keeper.NewQueryServer(k)
	res, err := queryServer.PermissionedRelayers(ctx, &types.QueryPermissionedRelayersRequest{})
	require.NoError(t, err)
	require.Len(t, res.PermissionedRelayers, 2)
	if res.PermissionedRelayers[0].ChannelId == channelID1 {
		require.Equal(t, res.PermissionedRelayers, []types.PermissionedRelayer{
			{
				PortId:    portID1,
				ChannelId: channelID1,
				Relayer:   addr1.String(),
			},
			{
				PortId:    portID2,
				ChannelId: channelID2,
				Relayer:   addr2.String(),
			},
		})
	} else {
		require.Equal(t, res.PermissionedRelayers, []types.PermissionedRelayer{
			{
				PortId:    portID2,
				ChannelId: channelID2,
				Relayer:   addr2.String(),
			},
			{
				PortId:    portID1,
				ChannelId: channelID1,
				Relayer:   addr1.String(),
			},
		})
	}

}
