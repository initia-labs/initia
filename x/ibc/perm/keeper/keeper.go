package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v8/modules/core/exported"

	"github.com/initia-labs/initia/x/ibc/perm/types"
)

type Keeper struct {
	cdc codec.Codec
	ac  address.Codec

	authority string

	Schema          collections.Schema
	ChannelRelayers collections.Map[string, []byte]
}

// NewKeeper creates a new IBC perm Keeper instance
func NewKeeper(
	cdc codec.Codec,
	storeService store.KVStoreService,
	authority string,
	ac address.Codec,
) *Keeper {
	if _, err := ac.StringToBytes(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	sb := collections.NewSchemaBuilder(storeService)
	k := &Keeper{
		cdc:             cdc,
		authority:       authority,
		ChannelRelayers: collections.NewMap(sb, types.ChannelRelayerPrefixKey, "channel_relayers", collections.StringKey, collections.BytesValue),
		ac:              ac,
	}

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema

	return k
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", "x/"+exported.ModuleName+"-"+types.ModuleName)
}

func (k Keeper) IterateChannelRelayer(ctx context.Context, cb func(channelRelayer types.ChannelRelayer) (bool, error)) error {
	return k.ChannelRelayers.Walk(ctx, nil, func(channel string, relayer []byte) (stop bool, err error) {
		relayerStr, err := k.ac.BytesToString(relayer)
		if err != nil {
			return true, err
		}

		return cb(types.ChannelRelayer{
			Channel: channel,
			Relayer: relayerStr,
		})
	})
}

func (k Keeper) SetChannelRelayer(ctx context.Context, channel string, relayer sdk.AccAddress) error {
	return k.ChannelRelayers.Set(ctx, channel, relayer)
}
