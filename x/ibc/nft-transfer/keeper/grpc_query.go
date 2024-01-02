package keeper

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cosmossdk.io/errors"

	"cosmossdk.io/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/initia-labs/initia/x/ibc/nft-transfer/types"
)

var _ types.QueryServer = Keeper{}

// Params implements the Query/Params gRPC method
func (q Keeper) Params(c context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	params := q.GetParams(ctx)

	return &types.QueryParamsResponse{
		Params: &params,
	}, nil
}

// EscrowAddress implements the EscrowAddress gRPC method
func (q Keeper) EscrowAddress(c context.Context, req *types.QueryEscrowAddressRequest) (*types.QueryEscrowAddressResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	addr := types.GetEscrowAddress(req.PortId, req.ChannelId)

	return &types.QueryEscrowAddressResponse{
		EscrowAddress: addr.String(),
	}, nil
}

// ClassTrace implements the Query/ClassTrace gRPC method
func (q Keeper) ClassTrace(c context.Context, req *types.QueryClassTraceRequest) (*types.QueryClassTraceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	hash, err := types.ParseHexHash(strings.TrimPrefix(req.Hash, "ibc/"))
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid class id trace hash: %s, error: %s", hash.String(), err))
	}

	ctx := sdk.UnwrapSDKContext(c)
	classTrace, found := q.GetClassTrace(ctx, hash)
	if !found {
		return nil, status.Error(
			codes.NotFound,
			errors.Wrap(types.ErrTraceNotFound, req.Hash).Error(),
		)
	}

	return &types.QueryClassTraceResponse{
		ClassTrace: &classTrace,
	}, nil
}

// ClassTraces implements the Query/ClassTraces gRPC method
func (q Keeper) ClassTraces(c context.Context, req *types.QueryClassTracesRequest) (*types.QueryClassTracesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(c)

	traces := types.Traces{}
	store := prefix.NewStore(ctx.KVStore(q.storeKey), types.ClassTraceKey)

	pageRes, err := query.Paginate(store, req.Pagination, func(_, value []byte) error {
		result, err := q.UnmarshalClassTrace(value)
		if err != nil {
			return err
		}

		traces = append(traces, result)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &types.QueryClassTracesResponse{
		ClassTraces: traces.Sort(),
		Pagination:  pageRes,
	}, nil
}

// ClassHash implements the Query/ClassHash gRPC method
func (q Keeper) ClassHash(c context.Context, req *types.QueryClassHashRequest) (*types.QueryClassHashResponse, error) {
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
	found := q.HasClassTrace(ctx, classIdHash)
	if !found {
		return nil, status.Error(
			codes.NotFound,
			errors.Wrap(types.ErrTraceNotFound, req.Trace).Error(),
		)
	}

	return &types.QueryClassHashResponse{
		Hash: classIdHash.String(),
	}, nil
}
