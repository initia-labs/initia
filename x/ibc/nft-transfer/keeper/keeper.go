package keeper

import (
	"fmt"

	tmbytes "github.com/cometbft/cometbft/libs/bytes"
	"github.com/cometbft/cometbft/libs/log"

	"cosmossdk.io/store/prefix"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	capabilitytypes "github.com/cosmos/ibc-go/modules/capability/types"

	host "github.com/cosmos/ibc-go/v8/modules/core/24-host"
	"github.com/cosmos/ibc-go/v8/modules/core/exported"

	"github.com/initia-labs/initia/x/ibc/nft-transfer/types"
)

type Keeper struct {
	storeKey   storetypes.StoreKey
	cdc        codec.BinaryCodec
	paramSpace paramtypes.Subspace

	ics4Wrapper   types.ICS4Wrapper
	channelKeeper types.ChannelKeeper
	portKeeper    types.PortKeeper
	authKeeper    types.AccountKeeper
	nftKeeper     types.NftKeeper
	scopedKeeper  capabilitykeeper.ScopedKeeper

	authority string
}

// NewKeeper creates a new IBC transfer Keeper instance
func NewKeeper(
	cdc codec.BinaryCodec, key storetypes.StoreKey, ics4Wrapper types.ICS4Wrapper,
	channelKeeper types.ChannelKeeper, portKeeper types.PortKeeper,
	authKeeper types.AccountKeeper, nftKeeper types.NftKeeper,
	scopedKeeper capabilitykeeper.ScopedKeeper, authority string,
) *Keeper {

	if _, err := sdk.AccAddressFromBech32(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address: %s", authority))
	}

	return &Keeper{
		cdc:           cdc,
		storeKey:      key,
		ics4Wrapper:   ics4Wrapper,
		channelKeeper: channelKeeper,
		portKeeper:    portKeeper,
		authKeeper:    authKeeper,
		nftKeeper:     nftKeeper,
		scopedKeeper:  scopedKeeper,
		authority:     authority,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+exported.ModuleName+"-"+types.ModuleName)
}

// IsBound checks if the nft-transfer module is already bound to the desired port
func (k Keeper) IsBound(ctx sdk.Context, portID string) bool {
	_, ok := k.scopedKeeper.GetCapability(ctx, host.PortPath(portID))
	return ok
}

// BindPort defines a wrapper function for the ort Keeper's function in
// order to expose it to module's InitGenesis function
func (k Keeper) BindPort(ctx sdk.Context, portID string) error {
	cap := k.portKeeper.BindPort(ctx, portID)
	return k.ClaimCapability(ctx, cap, host.PortPath(portID))
}

// GetPort returns the portID for the nft-transfer module. Used in ExportGenesis
func (k Keeper) GetPort(ctx sdk.Context) string {
	store := ctx.KVStore(k.storeKey)
	return string(store.Get(types.PortKey))
}

// SetPort sets the portID for the nft-transfer module. Used in InitGenesis
func (k Keeper) SetPort(ctx sdk.Context, portID string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.PortKey, []byte(portID))
}

// AuthenticateCapability wraps the scopedKeeper's AuthenticateCapability function
func (k Keeper) AuthenticateCapability(ctx sdk.Context, cap *capabilitytypes.Capability, name string) bool {
	return k.scopedKeeper.AuthenticateCapability(ctx, cap, name)
}

// ClaimCapability allows the nft-transfer module that can claim a capability that IBC module
// passes to it
func (k Keeper) ClaimCapability(ctx sdk.Context, cap *capabilitytypes.Capability, name string) error {
	return k.scopedKeeper.ClaimCapability(ctx, cap, name)
}

// GetClassTrace retrieves the full identifiers trace and base denomination from the store.
func (k Keeper) GetClassTrace(ctx sdk.Context, classTraceHash tmbytes.HexBytes) (types.ClassTrace, bool) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ClassTraceKey)
	bz := store.Get(classTraceHash)
	if bz == nil {
		return types.ClassTrace{}, false
	}

	classTrace := k.MustUnmarshalClassTrace(bz)
	return classTrace, true
}

// HasClassTrace checks if a the key with the given denomination trace hash exists on the store.
func (k Keeper) HasClassTrace(ctx sdk.Context, classTraceHash tmbytes.HexBytes) bool {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ClassTraceKey)
	return store.Has(classTraceHash)
}

// SetClassTrace sets a new {trace hash -> denom trace} pair to the store.
func (k Keeper) SetClassTrace(ctx sdk.Context, classTrace types.ClassTrace) {
	store := prefix.NewStore(ctx.KVStore(k.storeKey), types.ClassTraceKey)
	bz := k.MustMarshalClassTrace(classTrace)
	store.Set(classTrace.Hash(), bz)
}

// GetAllClassTraces returns the trace information for all the denominations.
func (k Keeper) GetAllClassTraces(ctx sdk.Context) types.Traces {
	traces := types.Traces{}
	k.IterateClassTraces(ctx, func(classTrace types.ClassTrace) bool {
		traces = append(traces, classTrace)
		return false
	})

	return traces.Sort()
}

// IterateClassTraces iterates over the denomination traces in the store
// and performs a callback function.
func (k Keeper) IterateClassTraces(ctx sdk.Context, cb func(classTrace types.ClassTrace) bool) {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, types.ClassTraceKey)

	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {

		classTrace := k.MustUnmarshalClassTrace(iterator.Value())
		if cb(classTrace) {
			break
		}
	}
}
