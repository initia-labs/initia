package keeper

import (
	"context"

	"cosmossdk.io/math"

	"github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"
)

// BaseDenom - base denom of native move dex
func (k Keeper) BaseDenom(ctx context.Context) (string, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return "", err
	}

	return params.BaseDenom, nil
}

// BaseMinGasPrice - min gas price in base denom unit
func (k Keeper) BaseMinGasPrice(ctx context.Context) (math.LegacyDec, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return math.LegacyDec{}, err
	}
	return params.BaseMinGasPrice, nil
}

// ArbitraryEnabled - arbitrary enabled flag
func (k Keeper) ArbitraryEnabled(ctx context.Context) (bool, error) {
	return NewCodeKeeper(&k).GetAllowArbitrary(ctx)
}

// AllowedPublishers - allowed publishers
func (k Keeper) AllowedPublishers(ctx context.Context) ([]vmtypes.AccountAddress, error) {
	return NewCodeKeeper(&k).GetAllowedPublishers(ctx)
}

// SetArbitraryEnabled - update arbitrary enabled flag
func (k Keeper) SetArbitraryEnabled(ctx context.Context, arbitraryEnabled bool) error {
	return NewCodeKeeper(&k).SetAllowArbitrary(ctx, arbitraryEnabled)
}

// SetAllowedPublishers - update allowed publishers
func (k Keeper) SetAllowedPublishers(ctx context.Context, allowedPublishers []vmtypes.AccountAddress) error {
	return NewCodeKeeper(&k).SetAllowedPublishers(ctx, allowedPublishers)
}

// ContractSharedRevenueRatio - percentage of fees distributed to developers
func (k Keeper) ContractSharedRevenueRatio(ctx context.Context) (math.LegacyDec, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return math.LegacyDec{}, err
	}

	return params.ContractSharedRevenueRatio, nil
}

// SetParams sets the x/move module parameters.
func (k Keeper) SetParams(ctx context.Context, params types.Params) error {
	if err := k.SetRawParams(ctx, params.ToRaw()); err != nil {
		return err
	}

	return NewCodeKeeper(&k).SetAllowArbitrary(ctx, params.ArbitraryEnabled)
}

// GetParams returns the x/move module parameters.
func (k Keeper) GetParams(ctx context.Context) (types.Params, error) {
	rawParams, err := k.Params.Get(ctx)
	if err != nil {
		return types.Params{}, err
	}

	allowArbitrary, allowedPublishers, err := NewCodeKeeper(&k).GetParams(ctx)
	if err != nil {
		return types.Params{}, err
	}

	_allowedPublishers := make([]string, len(allowedPublishers))
	for i, addr := range allowedPublishers {
		_allowedPublishers[i] = addr.String()
	}

	return rawParams.ToParams(allowArbitrary, _allowedPublishers), nil
}

// SetRawParams stores raw params to store.
func (k Keeper) SetRawParams(ctx context.Context, params types.RawParams) error {
	return k.Params.Set(ctx, params)
}
