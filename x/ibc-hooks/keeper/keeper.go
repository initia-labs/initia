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

	Schema          collections.Schema
	TransientSchema collections.Schema
	ACLs            collections.Map[[]byte, bool]
	Params          collections.Item[types.Params]

	ac address.Codec

	// these are used for custom queries
	transferFunds collections.Item[types.TransferFunds]
}

func NewKeeper(
	cdc codec.Codec,
	storeService corestoretypes.KVStoreService,
	transientStoreService corestoretypes.TransientStoreService,
	authority string,
	ac address.Codec,
) *Keeper {
	// ensure that authority is a valid AccAddress
	if _, err := ac.StringToBytes(authority); err != nil {
		panic("authority is not a valid acc address")
	}

	sb := collections.NewSchemaBuilder(storeService)
	transientSb := collections.NewSchemaBuilderFromAccessor(transientStoreService.OpenTransientStore)
	k := &Keeper{
		cdc:          cdc,
		storeService: storeService,
		authority:    authority,

		ACLs:          collections.NewMap(sb, types.ACLPrefix, "acls", collections.BytesKey, collections.BoolValue),
		Params:        collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		transferFunds: collections.NewItem(transientSb, types.TransferFundsKey, "transfer_funds", codec.CollValue[types.TransferFunds](cdc)),

		ac: ac,
	}
	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	transientSchema, err := transientSb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema
	k.TransientSchema = transientSchema
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

func (k Keeper) SetTransferFunds(ctx context.Context, transferFunds types.TransferFunds) error {
	return k.transferFunds.Set(ctx, transferFunds)
}

func (k Keeper) EmptyTransferFunds(ctx context.Context) error {
	return k.transferFunds.Remove(ctx)
}
