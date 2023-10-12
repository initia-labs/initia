package keeper

import (
	"context"

	"cosmossdk.io/math"
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
func (q QueryServer) Params(c context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	customParams := q.GetParams(ctx)

	return &types.QueryParamsResponse{Params: types.Params{
		CommunityTax:        customParams.CommunityTax,
		BaseProposerReward:  math.LegacyZeroDec(),
		BonusProposerReward: math.LegacyZeroDec(),
		WithdrawAddrEnabled: customParams.WithdrawAddrEnabled,
	}}, nil
}

// ValidatorDistributionInfo query validator's commission and self-delegation rewards
func (k QueryServer) ValidatorDistributionInfo(c context.Context, req *types.QueryValidatorDistributionInfoRequest) (*types.QueryValidatorDistributionInfoResponse, error) {
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

	// self-delegation rewards
	val := k.stakingKeeper.Validator(ctx, valAdr)
	if val == nil {
		return nil, errors.Wrap(types.ErrNoValidatorExists, req.ValidatorAddress)
	}

	delAdr := sdk.AccAddress(valAdr)

	del := k.stakingKeeper.Delegation(ctx, delAdr, valAdr)
	if del == nil {
		return nil, types.ErrNoDelegationExists
	}

	endingPeriod := k.IncrementValidatorPeriod(ctx, val)
	rewards := k.CalculateDelegationRewards(ctx, val, del, endingPeriod)

	// validator's commission
	validatorCommission := k.GetValidatorAccumulatedCommission(ctx, valAdr)

	return &types.QueryValidatorDistributionInfoResponse{
		Commission:      validatorCommission.Commissions.Sum(),
		OperatorAddress: delAdr.String(),
		SelfBondRewards: rewards.Sum(),
	}, nil
}

// ValidatorOutstandingRewards queries rewards of a validator address
func (q QueryServer) ValidatorOutstandingRewards(c context.Context, req *types.QueryValidatorOutstandingRewardsRequest) (*types.QueryValidatorOutstandingRewardsResponse, error) {
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

	return &types.QueryValidatorOutstandingRewardsResponse{Rewards: types.ValidatorOutstandingRewards{
		Rewards: rewards.Rewards.Sum(),
	}}, nil
}

// ValidatorCommission queries accumulated commission for a validator
func (q QueryServer) ValidatorCommission(c context.Context, req *types.QueryValidatorCommissionRequest) (*types.QueryValidatorCommissionResponse, error) {
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

	return &types.QueryValidatorCommissionResponse{Commission: types.ValidatorAccumulatedCommission{
		Commission: commission.Commissions.Sum(),
	}}, nil
}

// ValidatorSlashes queries slash events of a validator
func (q QueryServer) ValidatorSlashes(c context.Context, req *types.QueryValidatorSlashesRequest) (*types.QueryValidatorSlashesResponse, error) {
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
	events := make([]types.ValidatorSlashEvent, 0)
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
				events = append(events, types.NewValidatorSlashEvent(
					result.ValidatorPeriod, result.Fractions[0].Amount,
				))
			}
		}
		return true, nil
	})

	if err != nil {
		return nil, err
	}

	return &types.QueryValidatorSlashesResponse{Slashes: events, Pagination: pageRes}, nil
}

// DelegationRewards the total rewards accrued by a delegation
func (q QueryServer) DelegationRewards(c context.Context, req *types.QueryDelegationRewardsRequest) (*types.QueryDelegationRewardsResponse, error) {
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
	rewards := q.CalculateDelegationRewards(ctx, val, del, endingPeriod).Sum()

	return &types.QueryDelegationRewardsResponse{Rewards: rewards}, nil
}

// DelegationTotalRewards the total rewards accrued by a each validator
func (q QueryServer) DelegationTotalRewards(c context.Context, req *types.QueryDelegationTotalRewardsRequest) (*types.QueryDelegationTotalRewardsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.DelegatorAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "empty delegator address")
	}

	ctx := sdk.UnwrapSDKContext(c)

	total := sdk.DecCoins{}
	var delRewards []types.DelegationDelegatorReward

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
			delReward := q.CalculateDelegationRewards(ctx, val, del, endingPeriod).Sum()

			delRewards = append(delRewards, types.NewDelegationDelegatorReward(valAddr, delReward))
			total = total.Add(delReward...)
			return false
		},
	)

	return &types.QueryDelegationTotalRewardsResponse{Rewards: delRewards, Total: total}, nil
}

// DelegatorValidators queries the validators list of a delegator
func (q QueryServer) DelegatorValidators(c context.Context, req *types.QueryDelegatorValidatorsRequest) (*types.QueryDelegatorValidatorsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.DelegatorAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "empty delegator address")
	}

	ctx := sdk.UnwrapSDKContext(c)
	delAdr, err := sdk.AccAddressFromBech32(req.DelegatorAddress)
	if err != nil {
		return nil, err
	}
	var validators []string

	q.stakingKeeper.IterateDelegations(
		ctx, delAdr,
		func(_ int64, del stakingtypes.DelegationI) (stop bool) {
			validators = append(validators, del.GetValidatorAddr().String())
			return false
		},
	)

	return &types.QueryDelegatorValidatorsResponse{Validators: validators}, nil
}

// DelegatorWithdrawAddress queries Query/delegatorWithdrawAddress
func (q QueryServer) DelegatorWithdrawAddress(c context.Context, req *types.QueryDelegatorWithdrawAddressRequest) (*types.QueryDelegatorWithdrawAddressResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.DelegatorAddress == "" {
		return nil, status.Error(codes.InvalidArgument, "empty delegator address")
	}
	delAdr, err := sdk.AccAddressFromBech32(req.DelegatorAddress)
	if err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(c)
	withdrawAddr := q.GetDelegatorWithdrawAddr(ctx, delAdr)

	return &types.QueryDelegatorWithdrawAddressResponse{WithdrawAddress: withdrawAddr.String()}, nil
}

// CommunityPool queries the community pool coins
func (q QueryServer) CommunityPool(c context.Context, req *types.QueryCommunityPoolRequest) (*types.QueryCommunityPoolResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	pool := q.GetFeePoolCommunityCoins(ctx)

	return &types.QueryCommunityPoolResponse{Pool: pool}, nil
}
