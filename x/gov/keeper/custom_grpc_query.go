package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	customtypes "github.com/initia-labs/initia/x/gov/types"
)

var _ customtypes.QueryServer = CustomQueryServer{}

// CustomQueryServer implement initia distribution queries
type CustomQueryServer struct {
	*Keeper
}

// NewCustomQueryServer create CustomQueryServer instance
func NewCustomQueryServer(k *Keeper) CustomQueryServer {
	return CustomQueryServer{k}
}

// Params queries params of distribution module
func (q CustomQueryServer) Params(ctx context.Context, req *customtypes.QueryParamsRequest) (*customtypes.QueryParamsResponse, error) {
	params, err := q.Keeper.Params.Get(ctx)
	return &customtypes.QueryParamsResponse{Params: params}, err
}

// EmergencyProposals implements the Query/EmergencyProposals gRPC method
func (q CustomQueryServer) EmergencyProposals(c context.Context, req *customtypes.QueryEmergencyProposalsRequest) (*customtypes.QueryEmergencyProposalsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	proposals, pageRes, err := query.CollectionPaginate(ctx, q.Keeper.EmergencyProposals, req.Pagination, func(proposalID uint64, _ []byte) (v1.Proposal, error) {
		return q.Proposals.Get(ctx, proposalID)
	})
	if err != nil {
		return nil, err
	}

	return &customtypes.QueryEmergencyProposalsResponse{Proposals: proposals, Pagination: pageRes}, nil
}

func (q CustomQueryServer) LastEmergencyProposalTallyTimestamp(ctx context.Context, req *customtypes.QueryLastEmergencyProposalTallyTimestampRequest) (*customtypes.QueryLastEmergencyProposalTallyTimestampResponse, error) {
	timestamp, err := q.Keeper.LastEmergencyProposalTallyTimestamp.Get(ctx)
	if err != nil {
		return nil, err
	}

	return &customtypes.QueryLastEmergencyProposalTallyTimestampResponse{
		TallyTimestamp: timestamp,
	}, nil
}
