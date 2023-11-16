package keeper

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/cosmos/cosmos-sdk/x/gov/types"
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
func (q CustomQueryServer) Params(c context.Context, req *customtypes.QueryParamsRequest) (*customtypes.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	customParams := q.GetParams(ctx)

	return &customtypes.QueryParamsResponse{Params: customParams}, nil
}

// EmergencyProposals implements the Query/EmergencyProposals gRPC method
func (q Keeper) EmergencyProposals(c context.Context, req *customtypes.QueryEmergencyProposalsRequest) (*customtypes.QueryEmergencyProposalsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(q.storeKey)
	proposalStore := prefix.NewStore(store, customtypes.EmergencyProposalsPrefix)

	proposals := []v1.Proposal{}
	pageRes, err := query.FilteredPaginate(proposalStore, req.Pagination, func(key []byte, value []byte, accumulate bool) (bool, error) {
		proposalID := types.GetProposalIDFromBytes(key)
		proposal, found := q.GetProposal(ctx, proposalID)
		if !found {
			panic(fmt.Sprintf("proposal %d does not exist", proposalID))
		}

		if accumulate {
			proposals = append(proposals, proposal)
		}
		return true, nil
	})

	if err != nil {
		return nil, err
	}

	return &customtypes.QueryEmergencyProposalsResponse{Proposals: proposals, Pagination: pageRes}, nil
}

func (q Keeper) LastEmergencyProposalTallyTimestamp(c context.Context, req *customtypes.QueryLastEmergencyProposalTallyTimestampRequest) (*customtypes.QueryLastEmergencyProposalTallyTimestampResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)

	return &customtypes.QueryLastEmergencyProposalTallyTimestampResponse{
		TallyTime: q.GetLastEmergencyProposalTallyTimestamp(ctx),
	}, nil
}
