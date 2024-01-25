package keeper

import (
	"context"

	"github.com/initia-labs/initia/x/ibc/fetchprice/types"
)

var _ types.QueryServer = QueryServerImpl{}

type QueryServerImpl struct {
	*Keeper
}

func NewQueryServerImpl(k *Keeper) QueryServerImpl {
	return QueryServerImpl{k}
}

// Params implements the Query/Params gRPC method
func (q QueryServerImpl) Params(ctx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	params, err := q.Keeper.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryParamsResponse{
		Params: &params,
	}, nil
}
