package keeper

import (
	"context"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	corestoretypes "cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/x/ibc-hooks/types"
)

type Keeper struct {
	cdc          codec.Codec
	storeService corestoretypes.KVStoreService

	authority string

	Schema collections.Schema
	ACLs   collections.Map[[]byte, bool]
	Params collections.Item[types.Params]

	ac address.Codec
}

func NewKeeper(
	cdc codec.Codec,
	storeService corestoretypes.KVStoreService,
	authority string,
	ac address.Codec,
) *Keeper {
	// ensure that authority is a valid AccAddress
	if _, err := ac.StringToBytes(authority); err != nil {
		panic("authority is not a valid acc address")
	}

	sb := collections.NewSchemaBuilder(storeService)
	k := &Keeper{
		cdc:          cdc,
		storeService: storeService,
		authority:    authority,

		ACLs:   collections.NewMap(sb, types.ACLPrefix, "acls", collections.BytesKey, collections.BoolValue),
		Params: collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),

		ac: ac,
	}
	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema
	return k
}

// GetAuthority returns the x/move module's authority.
func (ak Keeper) GetAuthority() string {
	return ak.authority
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", "x/"+types.ModuleName)
}
