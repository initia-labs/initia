package keeper

import (
	"context"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/cosmos/cosmos-sdk/x/distribution/types"

	customtypes "github.com/initia-labs/initia/x/distribution/types"
	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"
)

var _ types.QueryServer = QueryServer{}

// QueryServer is the query server to implement
// cosmos distribution module queries
type QueryServer struct {
	Keeper
}

// NewQueryServer create QueryServer instance
func NewQueryServer(k Keeper) QueryServer {
	return QueryServer{k}
}

// Params queries params of distribution module
func (q QueryServer) Params(ctx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {

	customParams, err := q.Keeper.Params.Get(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryParamsResponse{Params: types.Params{
		CommunityTax:        customParams.CommunityTax,
		BaseProposerReward:  math.LegacyZeroDec(),
		BonusProposerReward: math.LegacyZeroDec(),
		WithdrawAddrEnabled: customParams.WithdrawAddrEnabled,
	}}, nil
}

// ValidatorDistributionInfo query validator's commission and self-delegation rewards
func (q QueryServer) ValidatorDistributionInfo(ctx context.Context, req *types.QueryValidatorDistributionInfoRequest) (*types.QueryValidatorDistributionInfoResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.ValidatorAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "empty validator address")
	}

	valAddr, err := q.stakingKeeper.ValidatorAddressCodec().StringToBytes(req.ValidatorAddress)
	if err != nil {
		return nil, err
	}

	// self-delegation rewards
	val, err := q.stakingKeeper.Validator(ctx, valAddr)
	if err != nil {
		return nil, err
	}

	delAddr := sdk.AccAddress(valAddr)

	del, err := q.stakingKeeper.Delegation(ctx, delAddr, valAddr)
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

	// validator's commission
	validatorCommission, err := q.GetValidatorAccumulatedCommission(ctx, valAddr)
	if err != nil {
		return nil, err
	}

	return &types.QueryValidatorDistributionInfoResponse{
		Commission:      validatorCommission.Commissions.Sum(),
		OperatorAddress: delAddr.String(),
		SelfBondRewards: rewards.Sum(),
	}, nil
}

// ValidatorOutstandingRewards queries rewards of a validator address
func (q QueryServer) ValidatorOutstandingRewards(ctx context.Context, req *types.QueryValidatorOutstandingRewardsRequest) (*types.QueryValidatorOutstandingRewardsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.ValidatorAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "empty validator address")
	}

	valAddr, err := q.stakingKeeper.ValidatorAddressCodec().StringToBytes(req.ValidatorAddress)
	if err != nil {
		return nil, err
	}
	rewards, err := q.GetValidatorOutstandingRewards(ctx, valAddr)
	if err != nil {
		return nil, err
	}

	return &types.QueryValidatorOutstandingRewardsResponse{Rewards: types.ValidatorOutstandingRewards{
		Rewards: rewards.Rewards.Sum(),
	}}, nil
}

// ValidatorCommission queries accumulated commission for a validator
func (q QueryServer) ValidatorCommission(ctx context.Context, req *types.QueryValidatorCommissionRequest) (*types.QueryValidatorCommissionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.ValidatorAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "empty validator address")
	}

	valAddr, err := q.stakingKeeper.ValidatorAddressCodec().StringToBytes(req.ValidatorAddress)
	if err != nil {
		return nil, err
	}
	commission, err := q.GetValidatorAccumulatedCommission(ctx, valAddr)
	if err != nil {
		return nil, err
	}

	return &types.QueryValidatorCommissionResponse{Commission: types.ValidatorAccumulatedCommission{
		Commission: commission.Commissions.Sum(),
	}}, nil
}

// ValidatorSlashes queries slash events of a validator
func (q QueryServer) ValidatorSlashes(ctx context.Context, req *types.QueryValidatorSlashesRequest) (*types.QueryValidatorSlashesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.ValidatorAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "empty validator address")
	}

	if req.EndingHeight < req.StartingHeight {
		return nil, status.Errorf(codes.InvalidArgument, "starting height greater than ending height (%d > %d)", req.StartingHeight, req.EndingHeight)
	}

	valAddr, err := q.stakingKeeper.ValidatorAddressCodec().StringToBytes(req.ValidatorAddress)
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
		func(key collections.Triple[[]byte, uint64, uint64], result customtypes.ValidatorSlashEvent) (types.ValidatorSlashEvent, error) {
			return types.NewValidatorSlashEvent(
				result.ValidatorPeriod, result.Fractions[0].Amount,
			), nil
		},
		func(o *query.CollectionsPaginateOptions[collections.Triple[[]byte, uint64, uint64]]) {
			prefix := collections.TriplePrefix[[]byte, uint64, uint64](valAddr)
			o.Prefix = &prefix
		},
	)
	if err != nil {
		return nil, err
	}

	return &types.QueryValidatorSlashesResponse{Slashes: events, Pagination: pageRes}, nil
}

// DelegationRewards the total rewards accrued by a delegation
func (q QueryServer) DelegationRewards(ctx context.Context, req *types.QueryDelegationRewardsRequest) (*types.QueryDelegationRewardsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.DelegatorAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "empty delegator address")
	}

	if req.ValidatorAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "empty validator address")
	}

	valAddr, err := q.stakingKeeper.ValidatorAddressCodec().StringToBytes(req.ValidatorAddress)
	if err != nil {
		return nil, err
	}

	val, err := q.stakingKeeper.Validator(ctx, valAddr)
	if err != nil {
		return nil, err
	}

	delAddr, err := q.authKeeper.AddressCodec().StringToBytes(req.DelegatorAddress)
	if err != nil {
		return nil, err
	}

	del, err := q.stakingKeeper.Delegation(ctx, delAddr, valAddr)
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

	return &types.QueryDelegationRewardsResponse{Rewards: rewards.Sum()}, nil
}

// DelegationTotalRewards the total rewards accrued by a each validator
func (q QueryServer) DelegationTotalRewards(ctx context.Context, req *types.QueryDelegationTotalRewardsRequest) (*types.QueryDelegationTotalRewardsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.DelegatorAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "empty delegator address")
	}

	total := sdk.DecCoins{}
	var delRewards []types.DelegationDelegatorReward

	delAddr, err := q.authKeeper.AddressCodec().StringToBytes(req.DelegatorAddress)
	if err != nil {
		return nil, err
	}

	err = q.stakingKeeper.IterateDelegations(
		ctx, delAddr,
		func(del stakingtypes.DelegationI) (stop bool, err error) {
			valAddr, err := q.stakingKeeper.ValidatorAddressCodec().StringToBytes(del.GetValidatorAddr())
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

			delRewardSum := delReward.Sum()
			delRewards = append(delRewards, types.NewDelegationDelegatorReward(del.GetValidatorAddr(), delRewardSum))
			total = total.Add(delRewardSum...)
			return false, nil
		},
	)
	if err != nil {
		return nil, err
	}

	return &types.QueryDelegationTotalRewardsResponse{Rewards: delRewards, Total: total}, nil
}

// DelegatorValidators queries the validators list of a delegator
func (q QueryServer) DelegatorValidators(ctx context.Context, req *types.QueryDelegatorValidatorsRequest) (*types.QueryDelegatorValidatorsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.DelegatorAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "empty delegator address")
	}

	delAddr, err := q.authKeeper.AddressCodec().StringToBytes(req.DelegatorAddress)
	if err != nil {
		return nil, err
	}
	var validators []string

	err = q.stakingKeeper.IterateDelegations(
		ctx, delAddr,
		func(del stakingtypes.DelegationI) (stop bool, err error) {
			validators = append(validators, del.GetValidatorAddr())
			return false, nil
		},
	)
	if err != nil {
		return nil, err
	}

	return &types.QueryDelegatorValidatorsResponse{Validators: validators}, nil
}

// DelegatorWithdrawAddress queries Query/delegatorWithdrawAddress
func (q QueryServer) DelegatorWithdrawAddress(ctx context.Context, req *types.QueryDelegatorWithdrawAddressRequest) (*types.QueryDelegatorWithdrawAddressResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.DelegatorAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "empty delegator address")
	}
	delAddr, err := q.authKeeper.AddressCodec().StringToBytes(req.DelegatorAddress)
	if err != nil {
		return nil, err
	}

	withdrawAddr, err := q.GetDelegatorWithdrawAddr(ctx, delAddr)
	if err != nil {
		return nil, err
	}

	withdrawAddrStr, err := q.authKeeper.AddressCodec().BytesToString(withdrawAddr)
	if err != nil {
		return nil, err
	}

	return &types.QueryDelegatorWithdrawAddressResponse{WithdrawAddress: withdrawAddrStr}, nil
}

// CommunityPool queries the community pool coins
func (q QueryServer) CommunityPool(ctx context.Context, req *types.QueryCommunityPoolRequest) (*types.QueryCommunityPoolResponse, error) {
	pool, err := q.FeePool.Get(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryCommunityPoolResponse{Pool: pool.CommunityPool}, nil
}
