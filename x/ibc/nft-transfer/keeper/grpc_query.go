package keeper

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/initia-labs/initia/x/ibc/nft-transfer/types"
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

// EscrowAddress implements the EscrowAddress gRPC method
func (q QueryServerImpl) EscrowAddress(ctx context.Context, req *types.QueryEscrowAddressRequest) (*types.QueryEscrowAddressResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	addr := types.GetEscrowAddress(req.PortId, req.ChannelId)

	return &types.QueryEscrowAddressResponse{
		EscrowAddress: addr.String(),
	}, nil
}

// ClassTrace implements the Query/ClassTrace gRPC method
func (q QueryServerImpl) ClassTrace(ctx context.Context, req *types.QueryClassTraceRequest) (*types.QueryClassTraceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	hash, err := types.ParseHexHash(strings.TrimPrefix(req.Hash, "ibc/"))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid class id trace hash: %s, error: %s", hash.String(), err))
	}

	classTrace, err := q.Keeper.ClassTraces.Get(ctx, hash)
	if err != nil && errors.Is(err, collections.ErrNotFound) {
		return nil, status.Error(
			codes.NotFound,
			errorsmod.Wrap(types.ErrTraceNotFound, req.Hash).Error(),
		)
	} else if err != nil {
		return nil, status.Error(
			codes.Internal,
			err.Error(),
		)
	}

	return &types.QueryClassTraceResponse{
		ClassTrace: &classTrace,
	}, nil
}

// ClassTraces implements the Query/ClassTraces gRPC method
func (q QueryServerImpl) ClassTraces(ctx context.Context, req *types.QueryClassTracesRequest) (*types.QueryClassTracesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	traces, pageRes, err := query.CollectionPaginate(ctx, q.Keeper.ClassTraces, req.Pagination, func(_ []byte, trace types.ClassTrace) (types.ClassTrace, error) {
		return trace, nil
	})
	if err != nil {
		return nil, err
	}

	return &types.QueryClassTracesResponse{
		ClassTraces: types.Traces(traces).Sort(),
		Pagination:  pageRes,
	}, nil
}

// ClassHash implements the Query/ClassHash gRPC method
func (q QueryServerImpl) ClassHash(c context.Context, req *types.QueryClassHashRequest) (*types.QueryClassHashResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	// Convert given request trace path to ClassTrace struct to confirm the path in a valid classId trace format
	classTrace := types.ParseClassTrace(req.Trace)
	if err := classTrace.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	ctx := sdk.UnwrapSDKContext(c)
	classIdHash := classTrace.Hash()
	found, err := q.Keeper.ClassTraces.Has(ctx, classIdHash)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	} else if !found {
		return nil, status.Error(
			codes.NotFound,
			errorsmod.Wrap(types.ErrTraceNotFound, req.Trace).Error(),
		)
	}

	return &types.QueryClassHashResponse{
		Hash: classIdHash.String(),
	}, nil
}
