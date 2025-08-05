package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/ibc-go/v10/modules/core/exported"

	"github.com/initia-labs/initia/x/ibc/nft-transfer/types"
)

type Keeper struct {
	cdc codec.Codec

	ics4Wrapper   types.ICS4Wrapper
	channelKeeper types.ChannelKeeper
	authKeeper    types.AccountKeeper
	nftKeeper     types.NftKeeper

	authority string

	Schema      collections.Schema
	PortID      collections.Item[string]
	Params      collections.Item[types.Params]
	ClassTraces collections.Map[[]byte, types.ClassTrace]
	ClassData   collections.Map[[]byte, string]
	TokenData   collections.Map[collections.Pair[[]byte, string], string]
}

// NewKeeper creates a new IBC nft-transfer Keeper instance
func NewKeeper(
	cdc codec.Codec, storeService store.KVStoreService, ics4Wrapper types.ICS4Wrapper,
	channelKeeper types.ChannelKeeper,
	authKeeper types.AccountKeeper, nftKeeper types.NftKeeper,
	authority string,
) *Keeper {

	if _, err := authKeeper.AddressCodec().StringToBytes(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	sb := collections.NewSchemaBuilder(storeService)
	k := &Keeper{
		cdc:           cdc,
		ics4Wrapper:   ics4Wrapper,
		channelKeeper: channelKeeper,
		authKeeper:    authKeeper,
		nftKeeper:     nftKeeper,
		authority:     authority,

		PortID:      collections.NewItem(sb, types.PortKey, "port_id", collections.StringValue),
		Params:      collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		ClassTraces: collections.NewMap(sb, types.ClassTraceKey, "class_traces", collections.BytesKey, codec.CollValue[types.ClassTrace](cdc)),
		ClassData:   collections.NewMap(sb, types.ClassDataPrefix, "class_data", collections.BytesKey, collections.StringValue),
		TokenData:   collections.NewMap(sb, types.TokenDataPrefix, "token_data", collections.PairKeyCodec(collections.BytesKey, collections.StringKey), collections.StringValue),
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

func (k Keeper) Codec() codec.Codec {
	return k.cdc
}

// GetAllClassTraces returns the trace information for all the denominations.
func (k Keeper) GetAllClassTraces(ctx context.Context) (types.Traces, error) {
	traces := types.Traces{}
	err := k.IterateClassTraces(ctx, func(classTrace types.ClassTrace) (bool, error) {
		traces = append(traces, classTrace)
		return false, nil
	})
	if err != nil {
		return nil, err
	}

	return traces.Sort(), nil
}

// IterateClassTraces iterates over the denomination traces in the store
// and performs a callback function.
func (k Keeper) IterateClassTraces(ctx context.Context, cb func(classTrace types.ClassTrace) (bool, error)) error {
	return k.ClassTraces.Walk(ctx, nil, func(key []byte, value types.ClassTrace) (stop bool, err error) {
		return cb(value)
	})
}
