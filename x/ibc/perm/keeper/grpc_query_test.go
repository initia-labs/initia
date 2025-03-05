package keeper_test

import (
	"slices"
	"testing"

	"github.com/cometbft/cometbft/crypto/secp256k1"

	"github.com/stretchr/testify/require"

	dbm "github.com/cosmos/cosmos-db"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/v1/x/ibc/perm/keeper"
	"github.com/initia-labs/initia/v1/x/ibc/perm/types"
)

func Test_QueryChannelState(t *testing.T) {
	ctx, k := _createTestInput(t, dbm.NewMemDB())

	portID := "port-123"
	channelID := "channel-123"
	pubKey := secp256k1.GenPrivKey().PubKey()
	addr := sdk.AccAddress(pubKey.Address())

	// set relayer
	require.NoError(t, k.SetChannelState(ctx, types.ChannelState{
		PortId:    portID,
		ChannelId: channelID,
		Relayers:  []string{addr.String()},
	}))

	// set channel relayer via msg handler
	queryServer := keeper.NewQueryServer(k)
	res, err := queryServer.ChannelState(ctx, &types.QueryChannelStateRequest{
		PortId:    portID,
		ChannelId: channelID,
	})
	require.NoError(t, err)
	require.True(t, slices.Contains(res.ChannelState.Relayers, addr.String()))

	res, err = queryServer.ChannelState(ctx, &types.QueryChannelStateRequest{
		PortId:    portID,
		ChannelId: channelID + "4",
	})
	require.NoError(t, err)
	require.Empty(t, res.ChannelState.Relayers)
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
	err := k.SetChannelState(ctx, types.ChannelState{
		PortId:    portID1,
		ChannelId: channelID1,
		Relayers:  []string{addr1.String()},
	})
	require.NoError(t, err)
	err = k.SetChannelState(ctx, types.ChannelState{
		PortId:    portID2,
		ChannelId: channelID2,
		Relayers:  []string{addr2.String()},
	})
	require.NoError(t, err)

	// set channel relayer via msg handler
	queryServer := keeper.NewQueryServer(k)
	res, err := queryServer.ChannelStates(ctx, &types.QueryChannelStatesRequest{})
	require.NoError(t, err)
	require.Len(t, res.ChannelStates, 2)
	if res.ChannelStates[0].ChannelId == channelID1 {
		require.Equal(t, res.ChannelStates, []types.ChannelState{
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
		require.Equal(t, res.ChannelStates, []types.ChannelState{
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
