package keeper_test

import (
	"testing"
	"time"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/store"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	initiaappparams "github.com/initia-labs/initia/app/params"
	"github.com/initia-labs/initia/x/ibc/perm/keeper"
	"github.com/initia-labs/initia/x/ibc/perm/types"
)

func MakeTestCodec(t testing.TB) codec.Codec {
	return MakeEncodingConfig(t).Marshaler
}

func MakeEncodingConfig(_ testing.TB) initiaappparams.EncodingConfig {
	encodingConfig := initiaappparams.MakeEncodingConfig()
	amino := encodingConfig.Amino
	interfaceRegistry := encodingConfig.InterfaceRegistry

	std.RegisterInterfaces(interfaceRegistry)
	std.RegisterLegacyAminoCodec(amino)

	// add initiad types
	types.RegisterInterfaces(interfaceRegistry)
	types.RegisterLegacyAminoCodec(amino)

	return encodingConfig
}

func _createTestInput(
	t testing.TB,
	db dbm.DB,
) (sdk.Context, *keeper.Keeper) {
	keys := sdk.NewKVStoreKeys(types.StoreKey)
	ms := store.NewCommitMultiStore(db)
	for _, v := range keys {
		ms.MountStoreWithDB(v, storetypes.StoreTypeIAVL, db)
	}

	require.NoError(t, ms.LoadLatestVersion())

	ctx := sdk.NewContext(ms, tmproto.Header{
		Height: 1234567,
		Time:   time.Date(2020, time.April, 22, 12, 0, 0, 0, time.UTC),
	}, false, log.NewNopLogger())

	encodingConfig := MakeEncodingConfig(t)
	appCodec := encodingConfig.Marshaler

	permKeeper := keeper.NewKeeper(
		appCodec,
		keys[types.StoreKey],
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	return ctx, permKeeper
}

func Test_GetChannelRelayer(t *testing.T) {
	ctx, k := _createTestInput(t, dbm.NewMemDB())

	channel := "channel-123"
	pubKey := secp256k1.GenPrivKey().PubKey()
	addr := sdk.AccAddress(pubKey.Address())

	// should be empty
	res := k.GetChannelRelayer(ctx, channel)
	require.Nil(t, res)

	// set channel relayer via msg handler
	k.SetChannelRelayer(ctx, channel, addr)

	// check properly set
	res = k.GetChannelRelayer(ctx, channel)
	require.Equal(t, *res, types.ChannelRelayer{
		Channel: channel,
		Relayer: addr.String(),
	})
}
