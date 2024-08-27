package keeper

import (
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

	Schema        collections.Schema
	ChannelStates collections.Map[collections.Pair[string, string], types.ChannelState]
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
		cdc:           cdc,
		authority:     authority,
		ChannelStates: collections.NewMap(sb, types.ChannelStatePrefix, "channel_state", collections.PairKeyCodec[string, string](collections.StringKey, collections.StringKey), codec.CollValue[types.ChannelState](cdc)),
		ac:            ac,
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

// IterateChannelState iterates over all the permissioned relayers.
func (k Keeper) IterateChannelStates(ctx context.Context, cb func(channelRelayer types.ChannelState) (bool, error)) error {
	return k.ChannelStates.Walk(ctx, nil, func(key collections.Pair[string, string], channelState types.ChannelState) (stop bool, err error) {
		return cb(channelState)
	})
}

// SetChannelState sets the relayer as the permissioned relayer for the channel.
func (k Keeper) SetChannelState(ctx context.Context, channelState types.ChannelState) error {
	return k.ChannelStates.Set(ctx, collections.Join(channelState.PortId, channelState.ChannelId), channelState)
}

// GetChannelState returns the permissioned relayer for the channel.
func (k Keeper) GetChannelState(ctx context.Context, portID, channelID string) (types.ChannelState, error) {
	channelState, err := k.ChannelStates.Get(ctx, collections.Join(portID, channelID))
	if err != nil && errors.Is(err, collections.ErrNotFound) {
		cs := types.NewChannelState(portID, channelID)
		return cs, nil
	} else if err != nil {
		return types.ChannelState{}, err
	}

	return channelState, nil
}

func (k Keeper) SetPermissionedRelayers(ctx context.Context, portID, channelID string, relayers []sdk.AccAddress) error {
	permRelayers, err := k.GetChannelState(ctx, portID, channelID)
	if err != nil {
		return err
	}

	strRelayers := make([]string, len(relayers))
	for _, relayer := range relayers {
		relayerStr, err := k.ac.BytesToString(relayer)
		if err != nil {
			return err
		}
		strRelayers = append(strRelayers, relayerStr)
	}

	permRelayers.Relayers = strRelayers
	return k.SetChannelState(ctx, permRelayers)
}

func (k Keeper) GetPermissionedRelayers(ctx context.Context, portID, channelID string) ([]sdk.AccAddress, error) {
	permRelayers, err := k.GetChannelState(ctx, portID, channelID)
	if err != nil {
		return nil, err
	}

	relayers := make([]sdk.AccAddress, len(permRelayers.Relayers))
	for i, relayer := range permRelayers.Relayers {
		relayerBytes, err := k.ac.StringToBytes(relayer)
		if err != nil {
			return nil, err
		}
		relayers[i] = relayerBytes
	}

	return relayers, nil
}

// HasPermission checks if the relayer has permission to relay packets on the channel.
func (k Keeper) HasPermission(ctx context.Context, portID, channelID string, relayer sdk.AccAddress) (bool, error) {
	permRelayers, err := k.ChannelStates.Get(ctx, collections.Join(portID, channelID))
	if err != nil && errors.Is(err, collections.ErrNotFound) {
		// if no permissioned relayers are set, all relayers are allowed
		return true, nil
	} else if err != nil {
		return false, err
	}

	relayerStr, err := k.ac.BytesToString(relayer)
	if err != nil {
		return false, err
	}

	return permRelayers.HasRelayer(relayerStr), nil
}

func (k Keeper) IsHalted(ctx context.Context, portID, channelID string) (bool, error) {
	cs, err := k.GetChannelState(ctx, portID, channelID)
	if err != nil {
		return false, err
	}

	return cs.HaltState.Halted, nil
}
