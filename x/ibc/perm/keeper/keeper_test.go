package keeper_test

import (
	"testing"
	"time"

	"github.com/cometbft/cometbft/crypto/secp256k1"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/address"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

	initiaappparams "github.com/initia-labs/initia/app/params"
	"github.com/initia-labs/initia/x/ibc/perm/keeper"
	"github.com/initia-labs/initia/x/ibc/perm/types"
)

func MakeTestCodec(t testing.TB) codec.Codec {
	return MakeEncodingConfig(t).Codec
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
	keys := storetypes.NewKVStoreKeys(types.StoreKey)
	ms := store.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	for _, v := range keys {
		ms.MountStoreWithDB(v, storetypes.StoreTypeIAVL, db)
	}

	require.NoError(t, ms.LoadLatestVersion())

	ctx := sdk.NewContext(ms, tmproto.Header{
		Height: 1234567,
		Time:   time.Date(2020, time.April, 22, 12, 0, 0, 0, time.UTC),
	}, false, log.NewNopLogger())

	encodingConfig := MakeEncodingConfig(t)
	appCodec := encodingConfig.Codec

	ac := address.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	permKeeper := keeper.NewKeeper(
		appCodec,
		runtime.NewKVStoreService(keys[types.StoreKey]),
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		ac,
	)

	return ctx, permKeeper
}

func Test_HasPermission(t *testing.T) {
	ctx, k := _createTestInput(t, dbm.NewMemDB())

	portID := "port-123"
	channelID := "channel-123"
	pubKey := secp256k1.GenPrivKey().PubKey()
	addr := sdk.AccAddress(pubKey.Address())

	cs, err := k.GetChannelState(ctx, portID, channelID)
	require.NoError(t, err)
	cs.Relayers = []string{addr.String()}
	err = k.SetChannelState(ctx, cs)
	require.NoError(t, err)

	ok, err := k.HasRelayerPermission(ctx, portID, channelID, addr)
	require.NoError(t, err)
	require.True(t, ok)

	// if no permissioned relayers are set, all relayers are allowed
	ok, err = k.HasRelayerPermission(ctx, portID, channelID+"2", addr)
	require.NoError(t, err)
	require.True(t, ok)

	pubKey2 := secp256k1.GenPrivKey().PubKey()
	addr2 := sdk.AccAddress(pubKey2.Address())
	ok, err = k.HasRelayerPermission(ctx, portID, channelID, addr2)
	require.NoError(t, err)
	require.False(t, ok)
}
