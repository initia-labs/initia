package keeper

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	customtypes "github.com/initia-labs/initia/x/gov/types"

	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	"cosmossdk.io/collections"
	"cosmossdk.io/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

	proposals, pageRes, err := query.CollectionPaginate(ctx, q.Keeper.EmergencyProposals, req.Pagination, func(proposalID uint64, _ []byte) (customtypes.Proposal, error) {
		return q.Keeper.Proposals.Get(ctx, proposalID)
	})
	if err != nil {
		return nil, err
	}

	return &customtypes.QueryEmergencyProposalsResponse{Proposals: proposals, Pagination: pageRes}, nil
}

// Proposal returns proposal details based on ProposalID
func (q CustomQueryServer) Proposal(ctx context.Context, req *customtypes.QueryProposalRequest) (*customtypes.QueryProposalResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.ProposalId == 0 {
		return nil, status.Error(codes.InvalidArgument, "proposal id can not be 0")
	}

	proposal, err := q.Keeper.Proposals.Get(ctx, req.ProposalId)
	if err != nil {
		if errors.IsOf(err, collections.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "proposal %d doesn't exist", req.ProposalId)
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &customtypes.QueryProposalResponse{Proposal: &proposal}, nil
}

// Proposals implements the Query/Proposals gRPC method
func (q CustomQueryServer) Proposals(ctx context.Context, req *customtypes.QueryProposalsRequest) (*customtypes.QueryProposalsResponse, error) {
	filteredProposals, pageRes, err := query.CollectionFilteredPaginate(ctx, q.Keeper.Proposals, req.Pagination, func(key uint64, p customtypes.Proposal) (include bool, err error) {
		matchVoter, matchDepositor, matchStatus := true, true, true

		// match status (if supplied/valid)
		if v1.ValidProposalStatus(req.ProposalStatus) {
			matchStatus = p.Status == req.ProposalStatus
		}

		// match voter address (if supplied)
		if len(req.Voter) > 0 {
			voter, err := q.Keeper.authKeeper.AddressCodec().StringToBytes(req.Voter)
			if err != nil {
				return false, err
			}

			has, err := q.Votes.Has(ctx, collections.Join(p.Id, sdk.AccAddress(voter)))
			// if no error, vote found, matchVoter = true
			matchVoter = err == nil && has
		}

		// match depositor (if supplied)
		if len(req.Depositor) > 0 {
			depositor, err := q.Keeper.authKeeper.AddressCodec().StringToBytes(req.Depositor)
			if err != nil {
				return false, err
			}
			has, err := q.Deposits.Has(ctx, collections.Join(p.Id, sdk.AccAddress(depositor)))
			// if no error, deposit found, matchDepositor = true
			matchDepositor = err == nil && has
		}

		// if all match, append to results
		if matchVoter && matchDepositor && matchStatus {
			return true, nil
		}
		// continue to next item, do not include because we're appending results above.
		return false, nil
	}, func(_ uint64, value customtypes.Proposal) (*customtypes.Proposal, error) {
		return &value, nil
	})

	if err != nil && !errors.IsOf(err, collections.ErrInvalidIterator) {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &customtypes.QueryProposalsResponse{Proposals: filteredProposals, Pagination: pageRes}, nil
}

// TallyResult queries the tally of a proposal vote
func (q CustomQueryServer) TallyResult(ctx context.Context, req *customtypes.QueryTallyResultRequest) (*customtypes.QueryTallyResultResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.ProposalId == 0 {
		return nil, status.Error(codes.InvalidArgument, "proposal id can not be 0")
	}

	proposal, err := q.Keeper.Proposals.Get(ctx, req.ProposalId)
	if err != nil {
		if errors.IsOf(err, collections.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "proposal %d doesn't exist", req.ProposalId)
		}
		return nil, status.Error(codes.Internal, err.Error())
	}

	var tallyResult customtypes.TallyResult

	switch proposal.Status {
	case v1.StatusDepositPeriod:
		tallyResult = customtypes.EmptyTallyResult()

	case v1.StatusPassed, v1.StatusRejected:
		tallyResult = proposal.FinalTallyResult

	default:
		// proposal is in voting period
		params, err := q.Keeper.Params.Get(ctx)
		if err != nil {
			return nil, err
		}

		_, _, _, tallyResult, err = q.Tally(ctx, params, proposal)
		if err != nil {
			return nil, err
		}
	}

	return &customtypes.QueryTallyResultResponse{TallyResult: tallyResult}, nil
}

func (q CustomQueryServer) SimulateProposal(ctx context.Context, req *customtypes.QuerySimulateProposalRequest) (*customtypes.QuerySimulateProposalResponse, error) {
	results := make([]sdk.Result, 0, len(req.MsgSubmitProposal.GetMessages()))

	msgs := req.MsgSubmitProposal.GetMessages()

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	cacheCtx, _ := sdkCtx.CacheContext()

	executeHandler := func(sdkCtx sdk.Context, msgIndex int, msg sdk.Msg, handler baseapp.MsgServiceHandler) (res *sdk.Result, err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic from msg %d", msgIndex)
			}
		}()
		res, err = handler(sdkCtx, msg)
		return
	}

	for msgIndex, anyMsg := range msgs {
		var msg sdk.Msg
		err := q.cdc.UnpackAny(anyMsg, &msg)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid message %d", msgIndex)
		}

		handler := q.router.Handler(msg)
		if handler == nil {
			return nil, status.Errorf(codes.InvalidArgument, "invalid message %d", msgIndex)
		}

		result, err := executeHandler(cacheCtx, msgIndex, msg, handler)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to execute message %d: %s", msgIndex, err)
		}
		results = append(results, *result)
	}

	return &customtypes.QuerySimulateProposalResponse{
		GasInfo: sdk.GasInfo{
			GasWanted: cacheCtx.GasMeter().Limit(),
			GasUsed:   cacheCtx.GasMeter().GasConsumed(),
		},
		Results: results,
	}, nil
}
