package keeper_test

import (
	"testing"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/crypto/secp256k1"

	"github.com/stretchr/testify/require"

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
	res := k.GetChannelRelayer(ctx, channel)
	require.Nil(t, res)

	// set channel relayer via msg handler
	msgServer := keeper.NewMsgServerImpl(k)
	_, err := msgServer.UpdateChannelRelayer(sdk.WrapSDKContext(ctx), types.NewMsgUpdateChannelRelayer(
		authtypes.NewModuleAddress(govtypes.ModuleName).String(), channel, addr.String(),
	))
	require.NoError(t, err)

	// check properly set
	res = k.GetChannelRelayer(ctx, channel)
	require.Equal(t, *res, types.ChannelRelayer{
		Channel: channel,
		Relayer: addr.String(),
	})
}
