package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"

	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"

	"github.com/initia-labs/initia/x/ibc/nft-transfer/types"
)

type Keeper struct {
	cdc codec.Codec

	ics4Wrapper   types.ICS4Wrapper
	channelKeeper types.ChannelKeeper
	portKeeper    types.PortKeeper
	authKeeper    types.AccountKeeper
	nftKeeper     types.NftKeeper
	scopedKeeper  capabilitykeeper.ScopedKeeper

	authority string

	Schema      collections.Schema
	PortID      collections.Item[string]
	Params      collections.Item[types.Params]
	ClassTraces collections.Map[[]byte, types.ClassTrace]
}

// NewKeeper creates a new IBC nft-transfer Keeper instance
func NewKeeper(
	cdc codec.Codec, storeService store.KVStoreService, ics4Wrapper types.ICS4Wrapper,
	channelKeeper types.ChannelKeeper, portKeeper types.PortKeeper,
	authKeeper types.AccountKeeper, nftKeeper types.NftKeeper,
	scopedKeeper capabilitykeeper.ScopedKeeper, authority string,
) *Keeper {

	if _, err := authKeeper.AddressCodec().StringToBytes(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	sb := collections.NewSchemaBuilder(storeService)
	k := &Keeper{
		cdc:           cdc,
		ics4Wrapper:   ics4Wrapper,
		channelKeeper: channelKeeper,
		portKeeper:    portKeeper,
		authKeeper:    authKeeper,
		nftKeeper:     nftKeeper,
		scopedKeeper:  scopedKeeper,
		authority:     authority,

		PortID:      collections.NewItem(sb, types.PortKey, "port_Id", collections.StringValue),
		Params:      collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		ClassTraces: collections.NewMap(sb, types.ClassTraceKey, "class_traces", collections.BytesKey, codec.CollValue[types.ClassTrace](cdc)),
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

// IsBound checks if the nft-transfer module is already bound to the desired port
func (k Keeper) IsBound(ctx context.Context, portID string) bool {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	_, ok := k.scopedKeeper.GetCapability(sdkCtx, host.PortPath(portID))
	return ok
}

// BindPort defines a wrapper function for the ort Keeper's function in
// order to expose it to module's InitGenesis function
func (k Keeper) BindPort(ctx context.Context, portID string) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	cap := k.portKeeper.BindPort(sdkCtx, portID)
	return k.ClaimCapability(ctx, cap, host.PortPath(portID))
}

// AuthenticateCapability wraps the scopedKeeper's AuthenticateCapability function
func (k Keeper) AuthenticateCapability(ctx context.Context, cap *capabilitytypes.Capability, name string) bool {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return k.scopedKeeper.AuthenticateCapability(sdkCtx, cap, name)
}

// ClaimCapability allows the nft-transfer module that can claim a capability that IBC module
// passes to it
func (k Keeper) ClaimCapability(ctx context.Context, cap *capabilitytypes.Capability, name string) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return k.scopedKeeper.ClaimCapability(sdkCtx, cap, name)
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
