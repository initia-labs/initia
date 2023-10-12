package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/x/reward/types"
)

// SetReleaseRate update release rate params
func (k Keeper) SetReleaseRate(ctx sdk.Context, rate sdk.Dec) error {
	params := k.GetParams(ctx)
	params.ReleaseRate = rate
	return k.SetParams(ctx, params)
}

// GetReleaseRate return release rate params
func (k Keeper) GetReleaseRate(ctx sdk.Context) sdk.Dec {
	return k.GetParams(ctx).ReleaseRate
}

// GetParams returns the current x/slashing module parameters.
func (k Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return params
	}
	k.cdc.MustUnmarshal(bz, &params)
	return params
}

// SetParams sets the x/slashing module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) error {
	if err := params.Validate(); err != nil {
		return err
	}

	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&params)
	store.Set(types.ParamsKey, bz)

	return nil
}
