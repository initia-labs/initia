package keeper

import (
	"context"

	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/initia-labs/initia/x/ibc/perm/types"
)

var _ types.QueryServer = QueryServerImpl{}

type QueryServerImpl struct {
	*Keeper
}

func NewQueryServer(k *Keeper) QueryServerImpl {
	return QueryServerImpl{k}
}

func (q QueryServerImpl) ChannelStates(ctx context.Context, req *types.QueryChannelStatesRequest) (*types.QueryChannelStatesResponse, error) {
	channelStates, pageRes, err := query.CollectionPaginate(ctx, q.Keeper.ChannelStates, req.Pagination, func(key collections.Pair[string, string], channelState types.ChannelState) (types.ChannelState, error) {
		return channelState, nil
	})
	if err != nil {
		return nil, err
	}

	return &types.QueryChannelStatesResponse{ChannelStates: channelStates, Pagination: pageRes}, nil
}

func (q QueryServerImpl) ChannelState(ctx context.Context, req *types.QueryChannelStateRequest) (*types.QueryChannelStateResponse, error) {
	ctx = sdk.UnwrapSDKContext(ctx)

	channelState, err := q.GetChannelState(ctx, req.PortId, req.ChannelId)
	if err != nil {
		return nil, err
	}

	return &types.QueryChannelStateResponse{ChannelState: channelState}, nil
}
