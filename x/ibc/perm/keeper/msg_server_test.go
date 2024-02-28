package keeper_test

import (
	"testing"

	"cosmossdk.io/collections"
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
	addr := sdk.AccAddress(pubKey.Address())

	// should be empty
	_, err := k.PermissionedRelayers.Get(ctx, collections.Join(portID, channelID))
	require.ErrorIs(t, err, collections.ErrNotFound)

	// set channel relayer via msg handler
	msgServer := keeper.NewMsgServerImpl(k)
	_, err = msgServer.SetPermissionedRelayer(ctx, types.NewMsgSetPermissionedRelayer(
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		portID, channelID, addr.String(),
	))
	require.NoError(t, err)

	// check properly set
	res, err := k.PermissionedRelayers.Get(ctx, collections.Join(portID, channelID))
	require.NoError(t, err)
	require.Equal(t, res, addr.Bytes())
}
