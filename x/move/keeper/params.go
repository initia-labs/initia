package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/initia-labs/initia/x/move/types"
)

// BaseDenom - base denom of native move dex
func (k Keeper) BaseDenom(ctx sdk.Context) string {
	return k.GetParams(ctx).BaseDenom
}

// BaseMinGasPrice - min gas price in base denom unit
func (k Keeper) BaseMinGasPrice(ctx sdk.Context) sdk.Dec {
	return k.GetParams(ctx).BaseMinGasPrice
}

// ArbitraryEnabled - arbitrary enabled flag
func (k Keeper) ArbitraryEnabled(ctx sdk.Context) (bool, error) {
	return NewCodeKeeper(&k).GetAllowArbitrary(ctx)
}

// SetArbitraryEnabled - update arbitrary enabled flag
func (k Keeper) SetArbitraryEnabled(ctx sdk.Context, arbitraryEnabled bool) error {
	return NewCodeKeeper(&k).SetAllowArbitrary(ctx, arbitraryEnabled)
}

// ContractSharedRevenueRatio - percentage of fees distributed to developers
func (k Keeper) ContractSharedRevenueRatio(ctx sdk.Context) sdk.Dec {
	return k.GetParams(ctx).ContractSharedRevenueRatio
}

// SetParams sets the x/move module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) error {
	if err := params.Validate(); err != nil {
		return err
	}

	if err := k.SetRawParams(ctx, params.ToRaw()); err != nil {
		return err
	}

	return NewCodeKeeper(&k).SetAllowArbitrary(ctx, params.ArbitraryEnabled)
}

// GetParams returns the x/move module parameters.
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		panic("params not found")
	}

	rawParams := types.RawParams{}
	k.cdc.MustUnmarshal(bz, &rawParams)

	allow, err := NewCodeKeeper(&k).GetAllowArbitrary(ctx)
	if err != nil {
		panic(err)
	}

	return rawParams.ToParams(allow)
}

// SetRawParams stores raw params to store.
func (k Keeper) SetRawParams(ctx sdk.Context, params types.RawParams) error {
	store := ctx.KVStore(k.storeKey)
	if bz, err := k.cdc.Marshal(&params); err != nil {
		return err
	} else {
		store.Set(types.ParamsKey, bz)
	}

	return nil
}
