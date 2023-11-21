package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"

	customtypes "github.com/initia-labs/initia/x/distribution/types"
	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"
)

var _ customtypes.QueryServer = CustomQueryServer{}

// CustomQueryServer implement initia distribution queries
type CustomQueryServer struct {
	Keeper
}

// NewCustomQueryServer create CustomQueryServer instance
func NewCustomQueryServer(k Keeper) CustomQueryServer {
	return CustomQueryServer{k}
}

// Params queries params of distribution module
func (q CustomQueryServer) Params(c context.Context, req *customtypes.QueryParamsRequest) (*customtypes.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	customParams := q.GetParams(ctx)

	return &customtypes.QueryParamsResponse{Params: customParams}, nil
}

// ValidatorOutstandingRewards queries rewards of a validator address
func (q CustomQueryServer) ValidatorOutstandingRewards(c context.Context, req *customtypes.QueryValidatorOutstandingRewardsRequest) (*customtypes.QueryValidatorOutstandingRewardsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.ValidatorAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "empty validator address")
	}

	ctx := sdk.UnwrapSDKContext(c)

	valAdr, err := sdk.ValAddressFromBech32(req.ValidatorAddress)
	if err != nil {
		return nil, err
	}

	rewards := q.GetValidatorOutstandingRewards(ctx, valAdr)
	return &customtypes.QueryValidatorOutstandingRewardsResponse{Rewards: rewards}, nil
}

// ValidatorCommission queries accumulated commission for a validator
func (q CustomQueryServer) ValidatorCommission(c context.Context, req *customtypes.QueryValidatorCommissionRequest) (*customtypes.QueryValidatorCommissionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.ValidatorAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "empty validator address")
	}

	ctx := sdk.UnwrapSDKContext(c)

	valAdr, err := sdk.ValAddressFromBech32(req.ValidatorAddress)
	if err != nil {
		return nil, err
	}

	commission := q.GetValidatorAccumulatedCommission(ctx, valAdr)
	return &customtypes.QueryValidatorCommissionResponse{Commission: commission}, nil
}

// ValidatorSlashes queries slash events of a validator
func (q CustomQueryServer) ValidatorSlashes(c context.Context, req *customtypes.QueryValidatorSlashesRequest) (*customtypes.QueryValidatorSlashesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.ValidatorAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "empty validator address")
	}

	if req.EndingHeight < req.StartingHeight {
		return nil, status.Errorf(codes.InvalidArgument, "starting height greater than ending height (%d > %d)", req.StartingHeight, req.EndingHeight)
	}

	ctx := sdk.UnwrapSDKContext(c)
	events := make([]customtypes.ValidatorSlashEvent, 0)
	store := ctx.KVStore(q.storeKey)
	valAddr, err := sdk.ValAddressFromBech32(req.ValidatorAddress)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid validator address")
	}
	slashesStore := prefix.NewStore(store, types.GetValidatorSlashEventPrefix(valAddr))

	pageRes, err := query.FilteredPaginate(slashesStore, req.Pagination, func(key []byte, value []byte, accumulate bool) (bool, error) {
		var result customtypes.ValidatorSlashEvent
		err := q.cdc.Unmarshal(value, &result)

		if err != nil {
			return false, err
		}

		if result.ValidatorPeriod < req.StartingHeight || result.ValidatorPeriod > req.EndingHeight {
			return false, nil
		}

		if accumulate {
			if result.Fractions.Len() > 0 {
				events = append(events, customtypes.NewValidatorSlashEvent(
					result.ValidatorPeriod, result.Fractions,
				))
			}
		}
		return true, nil
	})

	if err != nil {
		return nil, err
	}

	return &customtypes.QueryValidatorSlashesResponse{Slashes: events, Pagination: pageRes}, nil
}

// DelegationRewards the total rewards accrued by a delegation
func (q CustomQueryServer) DelegationRewards(c context.Context, req *customtypes.QueryDelegationRewardsRequest) (*customtypes.QueryDelegationRewardsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.DelegatorAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "empty delegator address")
	}

	if req.ValidatorAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "empty validator address")
	}

	ctx := sdk.UnwrapSDKContext(c)

	valAdr, err := sdk.ValAddressFromBech32(req.ValidatorAddress)
	if err != nil {
		return nil, err
	}

	val := q.stakingKeeper.Validator(ctx, valAdr)
	if val == nil {
		return nil, errors.Wrap(types.ErrNoValidatorExists, req.ValidatorAddress)
	}

	delAdr, err := sdk.AccAddressFromBech32(req.DelegatorAddress)
	if err != nil {
		return nil, err
	}
	del := q.stakingKeeper.Delegation(ctx, delAdr, valAdr)
	if del == nil {
		return nil, types.ErrNoDelegationExists
	}

	endingPeriod := q.IncrementValidatorPeriod(ctx, val)
	rewards := q.CalculateDelegationRewards(ctx, val, del, endingPeriod)

	return &customtypes.QueryDelegationRewardsResponse{Rewards: rewards}, nil
}

// DelegationTotalRewards the total rewards accrued by a each validator
func (q CustomQueryServer) DelegationTotalRewards(c context.Context, req *customtypes.QueryDelegationTotalRewardsRequest) (*customtypes.QueryDelegationTotalRewardsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.DelegatorAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "empty delegator address")
	}

	ctx := sdk.UnwrapSDKContext(c)

	total := customtypes.DecPools{}
	var delRewards []customtypes.DelegationDelegatorReward

	delAdr, err := sdk.AccAddressFromBech32(req.DelegatorAddress)
	if err != nil {
		return nil, err
	}

	q.stakingKeeper.IterateDelegations(
		ctx, delAdr,
		func(_ int64, del stakingtypes.DelegationI) (stop bool) {
			valAddr := del.GetValidatorAddr()
			val := q.stakingKeeper.Validator(ctx, valAddr)
			endingPeriod := q.IncrementValidatorPeriod(ctx, val)

			delReward := q.CalculateDelegationRewards(ctx, val, del, endingPeriod)
			delRewards = append(delRewards, customtypes.NewDelegationDelegatorReward(valAddr, delReward))

			total = total.Add(delReward...)
			return false
		},
	)

	return &customtypes.QueryDelegationTotalRewardsResponse{Rewards: delRewards, Total: total}, nil
}
