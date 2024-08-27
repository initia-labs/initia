package keeper_test

import (
	"testing"

	"github.com/cometbft/cometbft/crypto/secp256k1"

	"github.com/stretchr/testify/require"

	dbm "github.com/cosmos/cosmos-db"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	"github.com/initia-labs/initia/x/ibc/perm/keeper"
	"github.com/initia-labs/initia/x/ibc/perm/types"
)

func Test_SetPermissionedRelayer(t *testing.T) {
	ctx, k := _createTestInput(t, dbm.NewMemDB())

	portID := "port-123"
	channelID := "channel-123"
	pubKey := secp256k1.GenPrivKey().PubKey()
	addr1 := sdk.AccAddress(pubKey.Address())
	addr2 := sdk.AccAddress(pubKey.Address())

	// should be empty
	cs, err := k.GetChannelState(ctx, portID, channelID)
	require.NoError(t, err)
	require.Empty(t, cs.Relayers)

	// set channel relayer via msg handler
	msgServer := keeper.NewMsgServerImpl(k)
	_, err = msgServer.SetPermissionedRelayers(ctx, types.NewMsgSetPermissionedRelayers(
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		portID, channelID, []string{addr1.String(), addr2.String()},
	))
	require.NoError(t, err)

	// check properly set
	cs, err = k.GetChannelState(ctx, portID, channelID)
	require.NoError(t, err)
	require.True(t, cs.HasRelayer(addr1.String()))
	require.True(t, cs.HasRelayer(addr2.String()))
}

func Test_HaltChannel(t *testing.T) {
	ctx, k := _createTestInput(t, dbm.NewMemDB())

	portID := "port-123"
	channelID := "channel-123"
	pubKey := secp256k1.GenPrivKey().PubKey()
	addr1 := sdk.AccAddress(pubKey.Address())
	addr2 := sdk.AccAddress(pubKey.Address())

	// set channel relayer via msg handler
	msgServer := keeper.NewMsgServerImpl(k)
	_, err := msgServer.SetPermissionedRelayers(ctx, types.NewMsgSetPermissionedRelayers(
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		portID, channelID, []string{addr1.String(), addr2.String()},
	))
	require.NoError(t, err)

	// check properly set
	cs, err := k.GetChannelState(ctx, portID, channelID)
	require.NoError(t, err)
	require.True(t, cs.HasRelayer(addr1.String()))
	require.True(t, cs.HasRelayer(addr2.String()))

	// halt channel
	_, err = msgServer.HaltChannel(ctx, types.NewMsgHaltChannel(
		addr1.String(), portID, channelID,
	))
	require.NoError(t, err)

	// check properly set
	cs, err = k.GetChannelState(ctx, portID, channelID)
	require.NoError(t, err)
	require.True(t, cs.HaltState.Halted)
	require.Equal(t, addr1.String(), cs.HaltState.HaltedBy)

	// resume channel
	_, err = msgServer.ResumeChannel(ctx, types.NewMsgResumeChannel(
		addr1.String(), portID, channelID,
	))
	require.NoError(t, err)

	// check properly set
	cs, err = k.GetChannelState(ctx, portID, channelID)
	require.NoError(t, err)
	require.False(t, cs.HaltState.Halted)
	require.Empty(t, cs.HaltState.HaltedBy)

	// halt channel
	_, err = msgServer.HaltChannel(ctx, types.NewMsgHaltChannel(
		addr2.String(), portID, channelID,
	))
	require.NoError(t, err)

	// check properly set
	cs, err = k.GetChannelState(ctx, portID, channelID)
	require.NoError(t, err)
	require.True(t, cs.HaltState.Halted)
	require.Equal(t, addr2.String(), cs.HaltState.HaltedBy)

	// resume channel by govtypes.ModuleName
	_, err = msgServer.ResumeChannel(ctx, types.NewMsgResumeChannel(
		authtypes.NewModuleAddress(govtypes.ModuleName).String(), portID, channelID,
	))
	require.NoError(t, err)

	// check properly set
	cs, err = k.GetChannelState(ctx, portID, channelID)
	require.NoError(t, err)
	require.False(t, cs.HaltState.Halted)
	require.Empty(t, cs.HaltState.HaltedBy)

	// halt channel by govtypes.ModuleName
	_, err = msgServer.HaltChannel(ctx, types.NewMsgHaltChannel(
		authtypes.NewModuleAddress(govtypes.ModuleName).String(), portID, channelID,
	))
	require.NoError(t, err)

	// check properly set
	cs, err = k.GetChannelState(ctx, portID, channelID)
	require.NoError(t, err)
	require.True(t, cs.HaltState.Halted)
	require.Equal(t, authtypes.NewModuleAddress(govtypes.ModuleName).String(), cs.HaltState.HaltedBy)

	// resume channel by addr1
	_, err = msgServer.ResumeChannel(ctx, types.NewMsgResumeChannel(
		addr1.String(), portID, channelID,
	))
	require.Error(t, err)

	// resume channel by govtypes.ModuleName
	_, err = msgServer.ResumeChannel(ctx, types.NewMsgResumeChannel(
		authtypes.NewModuleAddress(govtypes.ModuleName).String(), portID, channelID,
	))
	require.NoError(t, err)
}
