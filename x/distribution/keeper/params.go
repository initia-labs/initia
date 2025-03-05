package keeper

import (
	"context"

	customtypes "github.com/initia-labs/initia/v1/x/distribution/types"
)

func (k Keeper) GetRewardWeights(ctx context.Context) (rewardWeights []customtypes.RewardWeight, err error) {
	params, err := k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	return params.RewardWeights, nil
}

func (k Keeper) SetRewardWeights(ctx context.Context, rewardWeights []customtypes.RewardWeight) error {
	params, err := k.Params.Get(ctx)
	if err != nil {
		return err
	}

	params.RewardWeights = rewardWeights
	return k.Params.Set(ctx, params)
}
