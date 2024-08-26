package keeper_test

import (
	"testing"

	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/stretchr/testify/require"

	dbm "github.com/cosmos/cosmos-db"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/x/ibc/perm/types"
)

func Test_InitGenesis(t *testing.T) {
	ctx, k := _createTestInput(t, dbm.NewMemDB())
	channelA := "channel-123"
	channelB := "channel-456"
	portA := "port-123"
	portB := "port-456"

	pubKeyA := secp256k1.GenPrivKey().PubKey()
	pubKeyB := secp256k1.GenPrivKey().PubKey()

	addrA := sdk.AccAddress(pubKeyA.Address())
	addrB := sdk.AccAddress(pubKeyB.Address())

	k.InitGenesis(ctx, types.GenesisState{
		ChannelStates: []types.ChannelState{
			{
				PortId:    portA,
				ChannelId: channelA,
				Relayers:  []string{addrA.String()},
				HaltState: types.HaltState{
					Halted:   true,
					HaltedBy: addrA.String(),
				},
			},
			{
				PortId:    portB,
				ChannelId: channelB,
				Relayers:  []string{addrB.String()},
				HaltState: types.HaltState{
					Halted:   false,
					HaltedBy: "",
				},
			},
		},
	})

	ok, err := k.HasPermission(ctx, portA, channelA, addrA)
	require.NoError(t, err)
	require.True(t, ok)

	cs, err := k.GetChannelState(ctx, portA, channelA)
	require.NoError(t, err)
	require.Equal(t, types.HaltState{
		Halted:   true,
		HaltedBy: addrA.String(),
	}, cs.HaltState)

	ok, _ = k.HasPermission(ctx, portB, channelB, addrB)
	require.NoError(t, err)
	require.True(t, ok)

	cs, err = k.GetChannelState(ctx, portA, channelB)
	require.NoError(t, err)
	require.Equal(t, types.HaltState{
		Halted:   false,
		HaltedBy: "",
	}, cs.HaltState)
}
func Test_ExportGenesis(t *testing.T) {
	ctx, k := _createTestInput(t, dbm.NewMemDB())

	channelA := "channel-123"
	channelB := "channel-456"
	portA := "port-123"
	portB := "port-456"

	pubKeyA := secp256k1.GenPrivKey().PubKey()
	pubKeyB := secp256k1.GenPrivKey().PubKey()

	addrA := sdk.AccAddress(pubKeyA.Address())
	addrB := sdk.AccAddress(pubKeyB.Address())

	genState := types.NewGenesisState(
		[]types.ChannelState{
			{
				PortId:    portA,
				ChannelId: channelA,
				Relayers:  []string{addrA.String()},
				HaltState: types.HaltState{
					Halted:   true,
					HaltedBy: addrA.String(),
				},
			},
			{
				PortId:    portB,
				ChannelId: channelB,
				Relayers:  []string{addrB.String()},
			},
		},
	)
	k.InitGenesis(ctx, *genState)
	exportedState := k.ExportGenesis(ctx)
	require.Equal(t, genState, exportedState)
}
