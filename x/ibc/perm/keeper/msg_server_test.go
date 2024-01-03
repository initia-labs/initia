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

func Test_UpdateChannelRelayer(t *testing.T) {
	ctx, k := _createTestInput(t, dbm.NewMemDB())

	channel := "channel-123"
	pubKey := secp256k1.GenPrivKey().PubKey()
	addr := sdk.AccAddress(pubKey.Address())

	// should be empty
	_, err := k.ChannelRelayers.Get(ctx, channel)
	require.ErrorIs(t, err, collections.ErrNotFound)

	// set channel relayer via msg handler
	msgServer := keeper.NewMsgServerImpl(k)
	_, err = msgServer.UpdateChannelRelayer(sdk.WrapSDKContext(ctx), types.NewMsgUpdateChannelRelayer(
		authtypes.NewModuleAddress(govtypes.ModuleName).String(), channel, addr.String(),
	))
	require.NoError(t, err)

	// check properly set
	res, err := k.ChannelRelayers.Get(ctx, channel)
	require.NoError(t, err)
	require.Equal(t, res, types.ChannelRelayer{
		Channel: channel,
		Relayer: addr.String(),
	})
}
