package keeper

import (
	"bytes"
	"context"
	"errors"
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

	Schema               collections.Schema
	PermissionedRelayers collections.Map[collections.Pair[string, string], []byte]
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
		cdc:                  cdc,
		authority:            authority,
		PermissionedRelayers: collections.NewMap(sb, types.PermissionedRelayerPrefixKey, "channel_relayers", collections.PairKeyCodec[string, string](collections.StringKey, collections.StringKey), collections.BytesValue),
		ac:                   ac,
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

// IteratePermissionedRelayers iterates over all the permissioned relayers.
func (k Keeper) IteratePermissionedRelayers(ctx context.Context, cb func(channelRelayer types.PermissionedRelayer) (bool, error)) error {
	return k.PermissionedRelayers.Walk(ctx, nil, func(key collections.Pair[string, string], relayer []byte) (stop bool, err error) {
		relayerStr, err := k.ac.BytesToString(relayer)
		if err != nil {
			return true, err
		}

		return cb(types.PermissionedRelayer{
			PortId:    key.K1(),
			ChannelId: key.K2(),
			Relayer:   relayerStr,
		})
	})
}

// SetPermissionedRelayer sets the relayer as the permissioned relayer for the channel.
func (k Keeper) SetPermissionedRelayer(ctx context.Context, portID, channelID string, relayer sdk.AccAddress) error {
	return k.PermissionedRelayers.Set(ctx, collections.Join(portID, channelID), relayer)
}

// HasPermission checks if the relayer has permission to relay packets on the channel.
func (k Keeper) HasPermission(ctx context.Context, portID, channelID string, relayer sdk.AccAddress) (bool, error) {
	permRelayer, err := k.PermissionedRelayers.Get(ctx, collections.Join(portID, channelID))
	if err != nil && errors.Is(err, collections.ErrNotFound) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return bytes.Equal(permRelayer, relayer.Bytes()), nil
}
