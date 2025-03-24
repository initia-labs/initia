package keeper

import (
	"context"

	"github.com/initia-labs/initia/x/ibc-hooks/types"

	"github.com/cosmos/cosmos-sdk/types/query"
)

type queryServer struct {
	*Keeper
}

var _ types.QueryServer = queryServer{}

// NewQueryServerImpl returns an implementation of the hook QueryServer interface
// for the provided Keeper.
func NewQueryServerImpl(k *Keeper) types.QueryServer {
	return &queryServer{Keeper: k}
}

func (qs queryServer) ACL(ctx context.Context, req *types.QueryACLRequest) (*types.QueryACLResponse, error) {
	addr, err := qs.ac.StringToBytes(req.Address)
	if err != nil {
		return nil, err
	}

	allowed, err := qs.GetAllowed(ctx, addr)
	if err != nil {
		return nil, err
	}

	return &types.QueryACLResponse{
		Acl: types.ACL{
			Address: req.Address,
			Allowed: allowed,
		},
	}, nil
}

func (qs queryServer) ACLs(ctx context.Context, req *types.QueryACLsRequest) (*types.QueryACLsResponse, error) {
	acls, pageRes, err := query.CollectionPaginate(ctx, qs.Keeper.ACLs, req.Pagination, func(addr []byte, allowed bool) (types.ACL, error) {
		addrStr, err := qs.ac.BytesToString(addr)
		if err != nil {
			return types.ACL{}, err
		}

		return types.ACL{
			Address: addrStr,
			Allowed: allowed,
		}, nil
	})
	if err != nil {
		return nil, err
	}

	return &types.QueryACLsResponse{
		Acls:       acls,
		Pagination: pageRes,
	}, nil
}

func (qs queryServer) Params(ctx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	params, err := qs.Keeper.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryParamsResponse{
		Params: params,
	}, nil
}
