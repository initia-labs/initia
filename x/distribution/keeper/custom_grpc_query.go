package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cosmossdk.io/collections"

	"github.com/cosmos/cosmos-sdk/types/query"

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
func (q CustomQueryServer) Params(ctx context.Context, req *customtypes.QueryParamsRequest) (*customtypes.QueryParamsResponse, error) {
	customParams, err := q.Keeper.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	return &customtypes.QueryParamsResponse{Params: customParams}, nil
}

// ValidatorOutstandingRewards queries rewards of a validator address
func (q CustomQueryServer) ValidatorOutstandingRewards(ctx context.Context, req *customtypes.QueryValidatorOutstandingRewardsRequest) (*customtypes.QueryValidatorOutstandingRewardsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.ValidatorAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "empty validator address")
	}

	valAdr, err := q.Keeper.stakingKeeper.ValidatorAddressCodec().StringToBytes(req.ValidatorAddress)
	if err != nil {
		return nil, err
	}

	rewards, err := q.GetValidatorOutstandingRewards(ctx, valAdr)
	if err != nil {
		return nil, err
	}

	return &customtypes.QueryValidatorOutstandingRewardsResponse{Rewards: rewards}, nil
}

// ValidatorCommission queries accumulated commission for a validator
func (q CustomQueryServer) ValidatorCommission(ctx context.Context, req *customtypes.QueryValidatorCommissionRequest) (*customtypes.QueryValidatorCommissionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.ValidatorAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "empty validator address")
	}

	valAdr, err := q.Keeper.stakingKeeper.ValidatorAddressCodec().StringToBytes(req.ValidatorAddress)
	if err != nil {
		return nil, err
	}

	commission, err := q.GetValidatorAccumulatedCommission(ctx, valAdr)
	if err != nil {
		return nil, err
	}

	return &customtypes.QueryValidatorCommissionResponse{Commission: commission}, nil
}

// ValidatorSlashes queries slash events of a validator
func (q CustomQueryServer) ValidatorSlashes(ctx context.Context, req *customtypes.QueryValidatorSlashesRequest) (*customtypes.QueryValidatorSlashesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.ValidatorAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "empty validator address")
	}

	if req.EndingHeight < req.StartingHeight {
		return nil, status.Errorf(codes.InvalidArgument, "starting height greater than ending height (%d > %d)", req.StartingHeight, req.EndingHeight)
	}

	valAddr, err := q.Keeper.stakingKeeper.ValidatorAddressCodec().StringToBytes(req.ValidatorAddress)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid validator address")
	}
	events, pageRes, err := query.CollectionFilteredPaginate(ctx, q.ValidatorSlashEvents, req.Pagination,
		func(key collections.Triple[[]byte, uint64, uint64], result customtypes.ValidatorSlashEvent) (bool, error) {
			if result.ValidatorPeriod < req.StartingHeight || result.ValidatorPeriod > req.EndingHeight {
				return false, nil
			}

			if result.Fractions.Len() <= 0 {
				return false, nil
			}

			return true, nil
		},
		func(key collections.Triple[[]byte, uint64, uint64], result customtypes.ValidatorSlashEvent) (customtypes.ValidatorSlashEvent, error) {
			return result, nil
		},
		func(o *query.CollectionsPaginateOptions[collections.Triple[[]byte, uint64, uint64]]) {
			prefix := collections.TriplePrefix[[]byte, uint64, uint64](valAddr)
			o.Prefix = &prefix
		},
	)
	if err != nil {
		return nil, err
	}

	return &customtypes.QueryValidatorSlashesResponse{Slashes: events, Pagination: pageRes}, nil
}

// DelegationRewards the total rewards accrued by a delegation
func (q CustomQueryServer) DelegationRewards(ctx context.Context, req *customtypes.QueryDelegationRewardsRequest) (*customtypes.QueryDelegationRewardsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.DelegatorAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "empty delegator address")
	}

	if req.ValidatorAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "empty validator address")
	}

	valAdr, err := q.Keeper.stakingKeeper.ValidatorAddressCodec().StringToBytes(req.ValidatorAddress)
	if err != nil {
		return nil, err
	}

	val, err := q.stakingKeeper.Validator(ctx, valAdr)
	if err != nil {
		return nil, err
	}

	delAdr, err := q.Keeper.authKeeper.AddressCodec().StringToBytes(req.DelegatorAddress)
	if err != nil {
		return nil, err
	}

	del, err := q.stakingKeeper.Delegation(ctx, delAdr, valAdr)
	if err != nil {
		return nil, err
	}

	endingPeriod, err := q.IncrementValidatorPeriod(ctx, val)
	if err != nil {
		return nil, err
	}

	rewards, err := q.CalculateDelegationRewards(ctx, val, del, endingPeriod)
	if err != nil {
		return nil, err
	}

	return &customtypes.QueryDelegationRewardsResponse{Rewards: rewards}, nil
}

// DelegationTotalRewards the total rewards accrued by a each validator
func (q CustomQueryServer) DelegationTotalRewards(ctx context.Context, req *customtypes.QueryDelegationTotalRewardsRequest) (*customtypes.QueryDelegationTotalRewardsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.DelegatorAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "empty delegator address")
	}

	total := customtypes.DecPools{}
	var delRewards []customtypes.DelegationDelegatorReward

	delAdr, err := q.Keeper.authKeeper.AddressCodec().StringToBytes(req.DelegatorAddress)
	if err != nil {
		return nil, err
	}

	err = q.stakingKeeper.IterateDelegations(
		ctx, delAdr,
		func(del stakingtypes.DelegationI) (stop bool, err error) {
			valAddr, err := q.Keeper.stakingKeeper.ValidatorAddressCodec().StringToBytes(del.GetValidatorAddr())
			if err != nil {
				return false, err
			}

			val, err := q.stakingKeeper.Validator(ctx, valAddr)
			if err != nil {
				return false, err
			}
			endingPeriod, err := q.IncrementValidatorPeriod(ctx, val)
			if err != nil {
				return false, err
			}

			delReward, err := q.CalculateDelegationRewards(ctx, val, del, endingPeriod)
			if err != nil {
				return false, err
			}

			delRewards = append(delRewards, customtypes.NewDelegationDelegatorReward(valAddr, delReward))
			total = total.Add(delReward...)
			return false, nil
		},
	)
	if err != nil {
		return nil, err
	}

	return &customtypes.QueryDelegationTotalRewardsResponse{Rewards: delRewards, Total: total}, nil
}
