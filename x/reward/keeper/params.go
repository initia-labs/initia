package keeper

import (
	"context"

	"cosmossdk.io/math"

	"github.com/initia-labs/initia/v1/x/reward/types"
)

// SetReleaseRate update release rate params
func (k Keeper) SetReleaseRate(ctx context.Context, rate math.LegacyDec) error {
	params, err := k.GetParams(ctx)
	if err != nil {
		return err
	}

	params.ReleaseRate = rate
	return k.SetParams(ctx, params)
}

// GetReleaseRate return release rate params
func (k Keeper) GetReleaseRate(ctx context.Context) (math.LegacyDec, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return math.LegacyZeroDec(), err
	}

	return params.ReleaseRate, nil
}

// GetParams returns the current x/slashing module parameters.
func (k Keeper) GetParams(ctx context.Context) (types.Params, error) {
	return k.Params.Get(ctx)
}

// SetParams sets the x/slashing module parameters.
func (k Keeper) SetParams(ctx context.Context, params types.Params) error {
	return k.Params.Set(ctx, params)
}
