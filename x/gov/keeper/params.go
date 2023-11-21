package keeper

import (
	customtypes "github.com/initia-labs/initia/x/gov/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov/types"
)

// SetParams sets the gov module's parameters.
func (k Keeper) SetParams(ctx sdk.Context, params customtypes.Params) error {
	store := ctx.KVStore(k.storeKey)
	bz, err := k.cdc.Marshal(&params)
	if err != nil {
		return err
	}
	store.Set(types.ParamsKey, bz)

	return nil
}

// GetParams gets the gov module's parameters.
func (k Keeper) GetParams(clientCtx sdk.Context) (params customtypes.Params) {
	store := clientCtx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return params
	}

	k.cdc.MustUnmarshal(bz, &params)
	return params
}
