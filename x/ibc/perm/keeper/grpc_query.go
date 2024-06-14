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

// PermissionedRelayer implements the Query/PermissionedRelayer gRPC method
func (q QueryServerImpl) PermissionedRelayersOneChannel(c context.Context, req *types.QueryPermissionedRelayersOneChannelRequest) (*types.QueryPermissionedRelayersOneChannelResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	relayerLists, err := q.Keeper.PermissionedRelayers.Get(ctx, collections.Join(req.PortId, req.ChannelId))
	if err != nil {
		return nil, err
	}

	return &types.QueryPermissionedRelayersOneChannelResponse{
		PermissionedRelayersSet: &types.PermissionedRelayersSet{
			PortId:      req.PortId,
			ChannelId:   req.ChannelId,
			RelayerList: &relayerLists,
		},
	}, nil
}

// PermissionedRelayersAllChannel implements the Query/PermissionedRelayersAllChannel gRPC method
func (q QueryServerImpl) PermissionedRelayersAllChannel(ctx context.Context, req *types.QueryPermissionedRelayersAllChannelRequest) (*types.QueryPermissionedRelayersAllChannelResponse, error) {
	relayerSets, pageRes, err := query.CollectionPaginate(
		ctx, q.Keeper.PermissionedRelayers, req.Pagination,
		func(key collections.Pair[string, string], relayerLists types.PermissionedRelayerList) (types.PermissionedRelayersSet, error) {

			return types.PermissionedRelayersSet{
				PortId:      key.K1(),
				ChannelId:   key.K2(),
				RelayerList: &relayerLists,
			}, nil
		},
	)
	if err != nil {
		return nil, err
	}

	return &types.QueryPermissionedRelayersAllChannelResponse{
		PermissionedRelayersSets: relayerSets,
		Pagination:               pageRes,
	}, nil
}
