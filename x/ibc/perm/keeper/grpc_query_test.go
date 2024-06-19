package keeper_test

import (
	"slices"
	"testing"

	"cosmossdk.io/collections"
	"github.com/cometbft/cometbft/crypto/secp256k1"

	"github.com/stretchr/testify/require"

	dbm "github.com/cosmos/cosmos-db"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/x/ibc/perm/keeper"
	"github.com/initia-labs/initia/x/ibc/perm/types"
)

func Test_QueryPermissionedRelayersOneChannel(t *testing.T) {
	ctx, k := _createTestInput(t, dbm.NewMemDB())

	portID := "port-123"
	channelID := "channel-123"
	pubKey := secp256k1.GenPrivKey().PubKey()
	addr := sdk.AccAddress(pubKey.Address())

	// set relayer
	require.NoError(t, k.PermissionedRelayers.Set(ctx, collections.Join(portID, channelID), types.PermissionedRelayersList{Relayers: []string{addr.String()}}))

	// set channel relayer via msg handler
	queryServer := keeper.NewQueryServer(k)
	res, err := queryServer.PermissionedRelayersByChannel(ctx, &types.QueryPermissionedRelayersByChannelRequest{
		PortId:    portID,
		ChannelId: channelID,
	})
	require.NoError(t, err)
	require.True(t, slices.Contains(res.PermissionedRelayers.Relayers, addr.String()))

	_, err = queryServer.PermissionedRelayersByChannel(ctx, &types.QueryPermissionedRelayersByChannelRequest{
		PortId:    portID,
		ChannelId: channelID + "4",
	})
	require.Error(t, err)

}

func Test_QueryAllPermissionedRelayers(t *testing.T) {
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
	require.NoError(t, k.PermissionedRelayers.Set(ctx, collections.Join(portID1, channelID1), types.PermissionedRelayersList{Relayers: []string{addr1.String()}}))
	require.NoError(t, k.PermissionedRelayers.Set(ctx, collections.Join(portID2, channelID2), types.PermissionedRelayersList{Relayers: []string{addr2.String()}}))

	// set channel relayer via msg handler
	queryServer := keeper.NewQueryServer(k)
	res, err := queryServer.AllPermissionedRelayers(ctx, &types.QueryAllPermissionedRelayersRequest{})
	require.NoError(t, err)
	require.Len(t, res.PermissionedRelayers, 2)
	if res.PermissionedRelayers[0].ChannelId == channelID1 {
		require.Equal(t, res.PermissionedRelayers, []types.PermissionedRelayers{
			{
				PortId:    portID1,
				ChannelId: channelID1,
				Relayers:  []string{addr1.String()},
			},
			{
				PortId:    portID2,
				ChannelId: channelID2,
				Relayers:  []string{addr2.String()},
			},
		})
	} else {
		require.Equal(t, res.PermissionedRelayers, []types.PermissionedRelayers{
			{
				PortId:    portID2,
				ChannelId: channelID2,
				Relayers:  []string{addr2.String()},
			},
			{
				PortId:    portID1,
				ChannelId: channelID1,
				Relayers:  []string{addr1.String()},
			},
		})
	}

}
