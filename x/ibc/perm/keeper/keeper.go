package keeper

import (
	"fmt"

	"github.com/cometbft/cometbft/libs/log"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v8/modules/core/exported"

	"github.com/initia-labs/initia/x/ibc/perm/types"
)

type Keeper struct {
	storeKey storetypes.StoreKey
	cdc      codec.BinaryCodec

	authority string
}

// NewKeeper creates a new IBC perm Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec,
	key storetypes.StoreKey,
	authority string,
) *Keeper {
	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	return &Keeper{
		cdc:       cdc,
		storeKey:  key,
		authority: authority,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+exported.ModuleName+"-"+types.ModuleName)
}

// GetChannelRelayer return channel permissioned relayer address.
func (k Keeper) GetChannelRelayer(ctx sdk.Context, channel string) *types.ChannelRelayer {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.GetChannelRelayerKey(channel))
	if bz == nil {
		return nil
	}

	var channelRelayer types.ChannelRelayer
	k.cdc.MustUnmarshal(bz, &channelRelayer)

	return &channelRelayer
}

// SetChannelRelayer set channel relayer in store.
func (k Keeper) SetChannelRelayer(ctx sdk.Context, channel string, relayer sdk.AccAddress) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.GetChannelRelayerKey(channel), k.cdc.MustMarshal(&types.ChannelRelayer{
		Channel: channel,
		Relayer: relayer.String(),
	}))
}

func (k Keeper) IterateChannelRelayer(ctx sdk.Context, cb func(channelRelayer types.ChannelRelayer) bool) {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, types.ChannelRelayerPrefixKey)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {

		var channelRelayer types.ChannelRelayer
		k.cdc.MustUnmarshal(iterator.Value(), &channelRelayer)

		if cb(channelRelayer) {
			break
		}
	}
}
