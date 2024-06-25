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

// PermissionedRelayersByChannel implements the Query/PermissionedRelayersByChannel gRPC method
func (q QueryServerImpl) PermissionedRelayersByChannel(c context.Context, req *types.QueryPermissionedRelayersByChannelRequest) (*types.QueryPermissionedRelayersByChannelResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	relayersList, err := q.Keeper.PermissionedRelayers.Get(ctx, collections.Join(req.PortId, req.ChannelId))
	if err != nil {
		return nil, err
	}

	return &types.QueryPermissionedRelayersByChannelResponse{
		PermissionedRelayers: &types.PermissionedRelayers{
			PortId:    req.PortId,
			ChannelId: req.ChannelId,
			Relayers:  relayersList.Relayers,
		},
	}, nil
}

// AllPermissionedRelayers implements the Query/AllPermissionedRelayers gRPC method
func (q QueryServerImpl) AllPermissionedRelayers(ctx context.Context, req *types.QueryAllPermissionedRelayersRequest) (*types.QueryAllPermissionedRelayersResponse, error) {
	relayers, pageRes, err := query.CollectionPaginate(
		ctx, q.Keeper.PermissionedRelayers, req.Pagination,
		func(key collections.Pair[string, string], relayersList types.PermissionedRelayersList) (types.PermissionedRelayers, error) {

			return types.PermissionedRelayers{
				PortId:    key.K1(),
				ChannelId: key.K2(),
				Relayers:  relayersList.Relayers,
			}, nil
		},
	)
	if err != nil {
		return nil, err
	}

	return &types.QueryAllPermissionedRelayersResponse{
		PermissionedRelayers: relayers,
		Pagination:           pageRes,
	}, nil
}
