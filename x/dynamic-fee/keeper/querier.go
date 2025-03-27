package keeper

import (
	"context"

	"github.com/initia-labs/initia/x/dynamic-fee/types"
)

type Querier struct {
	*Keeper
}

var _ types.QueryServer = &Querier{}

// NewQuerier return new Querier instance
func NewQuerier(k *Keeper) Querier {
	return Querier{k}
}

func (q Querier) Params(ctx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	params, err := q.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryParamsResponse{
		Params: params,
	}, nil
}
