package keeper

import (
	"context"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"
	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"

	providertypes "github.com/initia-labs/initia/x/ibc/fetchprice/provider/types"
)

type Keeper struct {
	cdc codec.Codec

	oracleKeeper providertypes.OracleKeeper
	portKeeper   providertypes.PortKeeper
	scopedKeeper capabilitykeeper.ScopedKeeper

	Schema collections.Schema
	PortID collections.Item[string]
}

func NewKeeper(
	cdc codec.Codec, storeService store.KVStoreService,
	oracleKeeper providertypes.OracleKeeper,
	portKeeper providertypes.PortKeeper,
	scopedKeeper capabilitykeeper.ScopedKeeper,
) *Keeper {
	sb := collections.NewSchemaBuilder(storeService)
	k := &Keeper{
		cdc:          cdc,
		oracleKeeper: oracleKeeper,
		portKeeper:   portKeeper,
		scopedKeeper: scopedKeeper,

		PortID: collections.NewItem(sb, providertypes.PortKey, "port_Id", collections.StringValue),
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
	return sdkCtx.Logger().With("module", "x/"+exported.ModuleName+"-"+providertypes.SubModuleName)
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
