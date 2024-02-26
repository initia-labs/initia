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
func (q QueryServerImpl) PermissionedRelayer(c context.Context, req *types.QueryPermissionedRelayerRequest) (*types.QueryPermissionedRelayerResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	relayer, err := q.Keeper.PermissionedRelayers.Get(ctx, collections.Join(req.PortId, req.ChannelId))
	if err != nil {
		return nil, err
	}

	relayerStr, err := q.ac.BytesToString(relayer)
	if err != nil {
		return nil, err
	}

	return &types.QueryPermissionedRelayerResponse{
		PermissionedRelayer: &types.PermissionedRelayer{
			PortId:    req.PortId,
			ChannelId: req.ChannelId,
			Relayer:   relayerStr,
		},
	}, nil
}

// PermissionedRelayers implements the Query/PermissionedRelayers gRPC method
func (q QueryServerImpl) PermissionedRelayers(ctx context.Context, req *types.QueryPermissionedRelayersRequest) (*types.QueryPermissionedRelayersResponse, error) {
	relayers, pageRes, err := query.CollectionPaginate(
		ctx, q.Keeper.PermissionedRelayers, req.Pagination,
		func(key collections.Pair[string, string], relayer []byte) (types.PermissionedRelayer, error) {
			relayerStr, err := q.ac.BytesToString(relayer)
			if err != nil {
				return types.PermissionedRelayer{}, err
			}

			return types.PermissionedRelayer{
				PortId:    key.K1(),
				ChannelId: key.K2(),
				Relayer:   relayerStr,
			}, nil
		},
	)
	if err != nil {
		return nil, err
	}

	return &types.QueryPermissionedRelayersResponse{
		PermissionedRelayers: relayers,
		Pagination:           pageRes,
	}, nil
}
