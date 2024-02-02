package keeper

import (
	"context"
	"time"
)

func (k Keeper) GetFetchEnabled(ctx context.Context) (bool, error) {
	params, err := k.Params.Get(ctx)
	if err != nil {
		return false, err
	}

	return params.FetchEnabled, nil
}

func (k Keeper) GetFetchActivated(ctx context.Context) (bool, error) {
	params, err := k.Params.Get(ctx)
	if err != nil {
		return false, err
	}

	return params.FetchActivated, nil
}

func (k Keeper) GetTimeoutDuration(ctx context.Context) (time.Duration, error) {
	params, err := k.Params.Get(ctx)
	if err != nil {
		return 0, err
	}

	return params.TimeoutDuration, nil
}
