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

func Test_UpdateAdmin(t *testing.T) {
	ctx, k := _createTestInput(t, dbm.NewMemDB())

	portID := "port-123"
	channelID := "channel-123"
	pubKey := secp256k1.GenPrivKey().PubKey()
	addr1 := sdk.AccAddress(pubKey.Address())

	// should be empty
	cs, err := k.GetChannelState(ctx, portID, channelID)
	require.NoError(t, err)
	require.Empty(t, cs.Admin)

	// set channel relayer via msg handler
	msgServer := keeper.NewMsgServerImpl(k)
	_, err = msgServer.UpdateAdmin(ctx, types.NewMsgUpdateAdmin(
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		portID, channelID, addr1.String(),
	))
	require.NoError(t, err)

	// check properly set
	cs, err = k.GetChannelState(ctx, portID, channelID)
	require.NoError(t, err)
	require.True(t, cs.Admin == addr1.String())
}

func Test_UpdatePermissionedRelayer(t *testing.T) {
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
	_, err = msgServer.UpdatePermissionedRelayers(ctx, types.NewMsgUpdatePermissionedRelayers(
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
