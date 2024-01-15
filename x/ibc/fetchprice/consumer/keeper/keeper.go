package keeper

import (
	"context"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"

	consumertypes "github.com/initia-labs/initia/x/ibc/fetchprice/consumer/types"
	"github.com/initia-labs/initia/x/ibc/fetchprice/types"
)

type Keeper struct {
	cdc codec.Codec
	ac  address.Codec

	ics4Wrapper   consumertypes.ICS4Wrapper
	channelKeeper consumertypes.ChannelKeeper
	portKeeper    consumertypes.PortKeeper
	scopedKeeper  capabilitykeeper.ScopedKeeper

	Schema collections.Schema
	PortID collections.Item[string]
	Prices collections.Map[string, types.QuotePrice]
}

func NewKeeper(
	cdc codec.Codec, storeService store.KVStoreService, ac address.Codec,
	ics4Wrapper consumertypes.ICS4Wrapper, channelKeeper consumertypes.ChannelKeeper,
	portKeeper consumertypes.PortKeeper, scopedKeeper capabilitykeeper.ScopedKeeper,
) *Keeper {
	sb := collections.NewSchemaBuilder(storeService)
	k := &Keeper{
		cdc:           cdc,
		ac:            ac,
		ics4Wrapper:   ics4Wrapper,
		channelKeeper: channelKeeper,
		portKeeper:    portKeeper,
		scopedKeeper:  scopedKeeper,

		PortID: collections.NewItem(sb, consumertypes.PortKey, "port_Id", collections.StringValue),
		Prices: collections.NewMap(sb, consumertypes.CurrencyPairPrefix, "currency_pairs", collections.StringKey, codec.CollValue[types.QuotePrice](cdc)),
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
	return sdkCtx.Logger().With("module", "x/"+exported.ModuleName+"-"+consumertypes.SubModuleName)
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
