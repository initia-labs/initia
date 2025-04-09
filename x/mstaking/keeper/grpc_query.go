package keeper

import (
	"context"
	"errors"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"cosmossdk.io/collections"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"

	"github.com/initia-labs/initia/x/mstaking/types"
)

// Querier is used as Keeper will have duplicate methods if used directly, and gRPC names take precedence over keeper
type Querier struct {
	*Keeper
}

var _ types.QueryServer = Querier{}

// Validators queries all validators that match the given status
func (q Querier) Validators(ctx context.Context, req *types.QueryValidatorsRequest) (*types.QueryValidatorsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	// validate the provided status, return all the validators if the status is empty
	if req.Status != "" && !(req.Status == types.Bonded.String() || req.Status == types.Unbonded.String() || req.Status == types.Unbonding.String()) {
		return nil, status.Errorf(codes.InvalidArgument, "invalid validator status %s", req.Status)
	}

	var validators []types.Validator
	var pageRes *query.PageResponse
	var err error

	if req.Status == types.Bonded.String() {
		validators, pageRes, err = query.CollectionPaginate(ctx, q.Keeper.ValidatorsByConsPowerIndex, req.Pagination, func(key collections.Pair[int64, []byte], _ bool) (types.Validator, error) {
			valAddr := key.K2()
			return q.Keeper.Validators.Get(ctx, valAddr)
		})
	} else {
		validators, pageRes, err = query.CollectionFilteredPaginate(ctx, q.Keeper.Validators, req.Pagination, func(valAddr []byte, val types.Validator) (include bool, err error) {
			return (req.Status == "" || strings.EqualFold(val.GetStatus().String(), req.Status)), nil
		}, func(valAddr []byte, val types.Validator) (types.Validator, error) {
			return val, nil
		})
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryValidatorsResponse{Validators: validators, Pagination: pageRes}, nil
}

// Validator queries validator info for given validator address
func (q Querier) Validator(ctx context.Context, req *types.QueryValidatorRequest) (*types.QueryValidatorResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if req.ValidatorAddr == "" {
		return nil, status.Error(codes.InvalidArgument, "validator address cannot be empty")
	}

	valAddr, err := q.Keeper.validatorAddressCodec.StringToBytes(req.ValidatorAddr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	validator, err := q.Keeper.Validators.Get(ctx, valAddr)
	if err != nil {
		return nil, status.Error(codes.NotFound, err.Error())
	}

	return &types.QueryValidatorResponse{Validator: validator}, nil
}

// ValidatorDelegations queries delegate info for given validator
func (q Querier) ValidatorDelegations(ctx context.Context, req *types.QueryValidatorDelegationsRequest) (*types.QueryValidatorDelegationsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if req.ValidatorAddr == "" {
		return nil, status.Error(codes.InvalidArgument, "validator address cannot be empty")
	}

	valAddr, err := q.Keeper.validatorAddressCodec.StringToBytes(req.ValidatorAddr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	delegations, pageRes, err := query.CollectionPaginate(
		ctx, q.Keeper.DelegationsByValIndex, req.Pagination,
		func(key collections.Pair[[]byte, []byte], _ bool) (types.Delegation, error) {
			valAddr, delAddr := key.K1(), key.K2()
			return q.GetDelegation(ctx, delAddr, valAddr)
		}, query.WithCollectionPaginationPairPrefix[[]byte, []byte](valAddr),
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	delResponses, err := delegationsToDelegationResponses(ctx, q.Keeper, delegations)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryValidatorDelegationsResponse{
		DelegationResponses: delResponses,
		Pagination:          pageRes,
	}, nil
}

// ValidatorUnbondingDelegations queries unbonding delegations of a validator
func (q Querier) ValidatorUnbondingDelegations(ctx context.Context, req *types.QueryValidatorUnbondingDelegationsRequest) (*types.QueryValidatorUnbondingDelegationsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if req.ValidatorAddr == "" {
		return nil, status.Error(codes.InvalidArgument, "validator address cannot be empty")
	}

	valAddr, err := q.Keeper.validatorAddressCodec.StringToBytes(req.ValidatorAddr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	unbondings, pageRes, err := query.CollectionPaginate(
		ctx, q.Keeper.UnbondingDelegationsByValIndex, req.Pagination,
		func(key collections.Pair[[]byte, []byte], _ bool) (types.UnbondingDelegation, error) {
			valAddr, delAddr := key.K1(), key.K2()
			return q.GetUnbondingDelegation(ctx, delAddr, valAddr)
		}, query.WithCollectionPaginationPairPrefix[[]byte, []byte](valAddr),
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryValidatorUnbondingDelegationsResponse{
		UnbondingResponses: unbondings,
		Pagination:         pageRes,
	}, nil
}

// Delegation queries delegate info for given validator delegator pair
func (q Querier) Delegation(ctx context.Context, req *types.QueryDelegationRequest) (*types.QueryDelegationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if req.DelegatorAddr == "" {
		return nil, status.Error(codes.InvalidArgument, "delegator address cannot be empty")
	}
	if req.ValidatorAddr == "" {
		return nil, status.Error(codes.InvalidArgument, "validator address cannot be empty")
	}

	delAddr, err := q.authKeeper.AddressCodec().StringToBytes(req.DelegatorAddr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	valAddr, err := q.validatorAddressCodec.StringToBytes(req.ValidatorAddr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	delegation, err := q.GetDelegation(ctx, delAddr, valAddr)
	if err != nil && errors.Is(err, collections.ErrNotFound) {
		return nil, status.Errorf(
			codes.NotFound,
			"delegation with delegator %s not found for validator %s",
			req.DelegatorAddr, req.ValidatorAddr)
	} else if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	delResponse, err := delegationToDelegationResponse(ctx, q.Keeper, delegation)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryDelegationResponse{DelegationResponse: &delResponse}, nil
}

// UnbondingDelegation queries unbonding info for give validator delegator pair
func (q Querier) UnbondingDelegation(ctx context.Context, req *types.QueryUnbondingDelegationRequest) (*types.QueryUnbondingDelegationResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	if req.DelegatorAddr == "" {
		return nil, status.Errorf(codes.InvalidArgument, "delegator address cannot be empty")
	}
	if req.ValidatorAddr == "" {
		return nil, status.Errorf(codes.InvalidArgument, "validator address cannot be empty")
	}

	delAddr, err := q.authKeeper.AddressCodec().StringToBytes(req.DelegatorAddr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	valAddr, err := q.validatorAddressCodec.StringToBytes(req.ValidatorAddr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	unbond, err := q.GetUnbondingDelegation(ctx, delAddr, valAddr)
	if err != nil && errors.Is(err, collections.ErrNotFound) {
		return nil, status.Errorf(
			codes.NotFound,
			"unbonding delegation with delegator %s not found for validator %s",
			req.DelegatorAddr, req.ValidatorAddr)
	} else if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryUnbondingDelegationResponse{Unbond: unbond}, nil
}

// DelegatorDelegations queries all delegations of a give delegator address
func (q Querier) DelegatorDelegations(ctx context.Context, req *types.QueryDelegatorDelegationsRequest) (*types.QueryDelegatorDelegationsResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	if req.DelegatorAddr == "" {
		return nil, status.Error(codes.InvalidArgument, "delegator address cannot be empty")
	}

	delAddr, err := q.authKeeper.AddressCodec().StringToBytes(req.DelegatorAddr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	delegations, pageRes, err := query.CollectionPaginate(
		ctx, q.Keeper.Delegations, req.Pagination,
		func(key collections.Pair[[]byte, []byte], delegation types.Delegation) (types.Delegation, error) {
			return delegation, nil
		}, query.WithCollectionPaginationPairPrefix[[]byte, []byte](delAddr),
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	delegationResps, err := delegationsToDelegationResponses(ctx, q.Keeper, delegations)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryDelegatorDelegationsResponse{DelegationResponses: delegationResps, Pagination: pageRes}, nil

}

// DelegatorValidator queries validator info for given delegator validator pair
func (k Querier) DelegatorValidator(c context.Context, req *types.QueryDelegatorValidatorRequest) (*types.QueryDelegatorValidatorResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if req.DelegatorAddr == "" {
		return nil, status.Error(codes.InvalidArgument, "delegator address cannot be empty")
	}
	if req.ValidatorAddr == "" {
		return nil, status.Error(codes.InvalidArgument, "validator address cannot be empty")
	}

	ctx := sdk.UnwrapSDKContext(c)
	delAddr, err := k.authKeeper.AddressCodec().StringToBytes(req.DelegatorAddr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	valAddr, err := k.validatorAddressCodec.StringToBytes(req.ValidatorAddr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	validator, err := k.GetDelegatorValidator(ctx, delAddr, valAddr)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryDelegatorValidatorResponse{Validator: validator}, nil
}

// DelegatorUnbondingDelegations queries all unbonding delegations of a given delegator address
func (q Querier) DelegatorUnbondingDelegations(ctx context.Context, req *types.QueryDelegatorUnbondingDelegationsRequest) (*types.QueryDelegatorUnbondingDelegationsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if req.DelegatorAddr == "" {
		return nil, status.Error(codes.InvalidArgument, "delegator address cannot be empty")
	}
	delAddr, err := q.authKeeper.AddressCodec().StringToBytes(req.DelegatorAddr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	unbondings, pageRes, err := query.CollectionPaginate(
		ctx, q.Keeper.UnbondingDelegations, req.Pagination,
		func(key collections.Pair[[]byte, []byte], unbonding types.UnbondingDelegation) (types.UnbondingDelegation, error) {
			return unbonding, nil
		}, query.WithCollectionPaginationPairPrefix[[]byte, []byte](delAddr),
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryDelegatorUnbondingDelegationsResponse{
		UnbondingResponses: unbondings, Pagination: pageRes}, nil
}

// RedelegationsOfDelegator queries redelegations of given delegator address
func (q Querier) RedelegationsOfDelegator(ctx context.Context, req *types.QueryRedelegationsOfDelegatorRequest) (*types.QueryRedelegationsOfDelegatorResponse, error) {
	res, err := q.Redelegations(ctx, &types.QueryRedelegationsRequest{
		DelegatorAddr: req.DelegatorAddr,
		Pagination:    req.Pagination,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryRedelegationsOfDelegatorResponse{
		RedelegationResponses: res.RedelegationResponses,
		Pagination:            res.Pagination,
	}, nil
}

// Redelegations queries redelegations of given address
func (q Querier) Redelegations(ctx context.Context, req *types.QueryRedelegationsRequest) (*types.QueryRedelegationsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	var redels types.Redelegations
	var pageRes *query.PageResponse
	var err error

	switch {
	case req.DelegatorAddr != "" && req.SrcValidatorAddr != "" && req.DstValidatorAddr != "":
		redels, err = queryRedelegation(ctx, q, req)
	case req.DelegatorAddr == "" && req.SrcValidatorAddr != "" && req.DstValidatorAddr == "":
		redels, pageRes, err = queryRedelegationsFromSrcValidator(ctx, q, req)
	case req.DelegatorAddr == "" && req.SrcValidatorAddr == "" && req.DstValidatorAddr != "":
		redels, pageRes, err = queryRedelegationsFromDstValidator(ctx, q, req)
	case req.DelegatorAddr != "" && req.SrcValidatorAddr == "" && req.DstValidatorAddr == "":
		redels, pageRes, err = queryDelegatorRedelegations(ctx, q, req)
	default:
		redels, pageRes, err = queryAllRedelegations(ctx, q, req)
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	redelResponses, err := redelegationsToRedelegationResponses(ctx, q.Keeper, redels)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryRedelegationsResponse{RedelegationResponses: redelResponses, Pagination: pageRes}, nil
}

// DelegatorValidators queries all validators info for given delegator address
func (q Querier) DelegatorValidators(ctx context.Context, req *types.QueryDelegatorValidatorsRequest) (*types.QueryDelegatorValidatorsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if req.DelegatorAddr == "" {
		return nil, status.Error(codes.InvalidArgument, "delegator address cannot be empty")
	}
	delAddr, err := q.authKeeper.AddressCodec().StringToBytes(req.DelegatorAddr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	validators, pageRes, err := query.CollectionPaginate(ctx, q.Keeper.Delegations, req.Pagination, func(key collections.Pair[[]byte, []byte], delegation types.Delegation) (types.Validator, error) {
		valAddr, err := q.validatorAddressCodec.StringToBytes(delegation.GetValidatorAddr())
		if err != nil {
			return types.Validator{}, err
		}

		return q.Keeper.Validators.Get(ctx, valAddr)
	}, query.WithCollectionPaginationPairPrefix[[]byte, []byte](delAddr))

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryDelegatorValidatorsResponse{Validators: validators, Pagination: pageRes}, nil
}

func (q Querier) DelegatorTotalDelegationBalance(ctx context.Context, req *types.QueryDelegatorTotalDelegationBalanceRequest) (*types.QueryDelegatorTotalDelegationBalanceResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if req.DelegatorAddr == "" {
		return nil, status.Error(codes.InvalidArgument, "delegator address cannot be empty")
	}

	delAddr, err := q.authKeeper.AddressCodec().StringToBytes(req.DelegatorAddr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	delegations, err := q.GetAllDelegatorDelegations(ctx, delAddr)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	delegationResps, err := delegationsToDelegationResponses(ctx, q.Keeper, delegations)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	var allBalances sdk.Coins
	for _, delegationResp := range delegationResps {
		allBalances = allBalances.Add(delegationResp.Balance...)
	}

	return &types.QueryDelegatorTotalDelegationBalanceResponse{Balance: allBalances}, nil
}

// Pool queries the pool info
func (q Querier) Pool(ctx context.Context, _ *types.QueryPoolRequest) (*types.QueryPoolResponse, error) {
	bondedPool := q.GetBondedPool(ctx)
	notBondedPool := q.GetNotBondedPool(ctx)
	powerWeights, err := q.GetVotingPowerWeights(ctx)
	if err != nil {
		return nil, err
	}

	pool := types.NewPool(
		q.bankKeeper.GetAllBalances(ctx, notBondedPool.GetAddress()),
		q.bankKeeper.GetAllBalances(ctx, bondedPool.GetAddress()),
		powerWeights,
	)

	return &types.QueryPoolResponse{Pool: pool}, nil
}

// Params queries the staking parameters
func (q Querier) Params(ctx context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	params, err := q.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	return &types.QueryParamsResponse{Params: params}, nil
}

func queryRedelegation(ctx context.Context, q Querier, req *types.QueryRedelegationsRequest) (redels types.Redelegations, err error) {
	delAddr, err := q.authKeeper.AddressCodec().StringToBytes(req.DelegatorAddr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	srcValAddr, err := q.validatorAddressCodec.StringToBytes(req.SrcValidatorAddr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	dstValAddr, err := q.validatorAddressCodec.StringToBytes(req.DstValidatorAddr)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	redel, err := q.GetRedelegation(ctx, delAddr, srcValAddr, dstValAddr)
	if err != nil && errors.Is(err, collections.ErrNotFound) {
		return nil, status.Errorf(
			codes.NotFound,
			"redelegation not found for delegator address %s from validator address %s",
			req.DelegatorAddr, req.SrcValidatorAddr)
	} else if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	redels = []types.Redelegation{redel}
	return redels, err
}

func queryRedelegationsFromSrcValidator(ctx context.Context, q Querier, req *types.QueryRedelegationsRequest) (redels types.Redelegations, res *query.PageResponse, err error) {
	valAddr, err := q.validatorAddressCodec.StringToBytes(req.SrcValidatorAddr)
	if err != nil {
		return nil, nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return query.CollectionPaginate(ctx, q.RedelegationsByValSrcIndex, req.Pagination, func(key collections.Triple[[]byte, []byte, []byte], _ bool) (types.Redelegation, error) {
		srcValAddr, delAddr, dstValAddr := key.K1(), key.K2(), key.K3()
		return q.GetRedelegation(ctx, delAddr, srcValAddr, dstValAddr)
	}, func(o *query.CollectionsPaginateOptions[collections.Triple[[]byte, []byte, []byte]]) {
		prefix := collections.TriplePrefix[[]byte, []byte, []byte](valAddr)
		o.Prefix = &prefix
	})
}

func queryRedelegationsFromDstValidator(ctx context.Context, q Querier, req *types.QueryRedelegationsRequest) (redels types.Redelegations, res *query.PageResponse, err error) {
	valAddr, err := q.validatorAddressCodec.StringToBytes(req.DstValidatorAddr)
	if err != nil {
		return nil, nil, status.Error(codes.InvalidArgument, err.Error())
	}

	return query.CollectionPaginate(ctx, q.RedelegationsByValDstIndex, req.Pagination, func(key collections.Triple[[]byte, []byte, []byte], _ bool) (types.Redelegation, error) {
		dstValAddr, delAddr, srcValAddr := key.K1(), key.K2(), key.K3()
		return q.GetRedelegation(ctx, delAddr, srcValAddr, dstValAddr)
	}, func(o *query.CollectionsPaginateOptions[collections.Triple[[]byte, []byte, []byte]]) {
		prefix := collections.TriplePrefix[[]byte, []byte, []byte](valAddr)
		o.Prefix = &prefix
	})
}

func queryDelegatorRedelegations(ctx context.Context, q Querier, req *types.QueryRedelegationsRequest) (redels types.Redelegations, res *query.PageResponse, err error) {
	delAddr, err := q.authKeeper.AddressCodec().StringToBytes(req.DelegatorAddr)
	if err != nil {
		return nil, nil, err
	}

	return query.CollectionPaginate(ctx, q.Keeper.Redelegations, req.Pagination, func(key collections.Triple[[]byte, []byte, []byte], redelegation types.Redelegation) (types.Redelegation, error) {
		return redelegation, nil
	}, func(o *query.CollectionsPaginateOptions[collections.Triple[[]byte, []byte, []byte]]) {
		prefix := collections.TriplePrefix[[]byte, []byte, []byte](delAddr)
		o.Prefix = &prefix
	})
}

func queryAllRedelegations(ctx context.Context, q Querier, req *types.QueryRedelegationsRequest) (redels types.Redelegations, res *query.PageResponse, err error) {
	return query.CollectionPaginate(ctx, q.Keeper.Redelegations, req.Pagination, func(key collections.Triple[[]byte, []byte, []byte], redelegation types.Redelegation) (types.Redelegation, error) {
		return redelegation, nil
	})
}

// util

func delegationToDelegationResponse(ctx context.Context, k *Keeper, del types.Delegation) (types.DelegationResponse, error) {
	valAddr, err := k.validatorAddressCodec.StringToBytes(del.GetValidatorAddr())
	if err != nil {
		return types.DelegationResponse{}, err
	}

	val, err := k.Validators.Get(ctx, valAddr)
	if err != nil {
		return types.DelegationResponse{}, err
	}

	_, err = k.authKeeper.AddressCodec().StringToBytes(del.DelegatorAddress)
	if err != nil {
		return types.DelegationResponse{}, err
	}

	balance, _ := val.TokensFromShares(del.Shares).TruncateDecimal()
	return types.NewDelegationResp(
		del.GetDelegatorAddr(),
		del.GetValidatorAddr(),
		del.Shares,
		balance,
	), nil
}

func delegationsToDelegationResponses(ctx context.Context, k *Keeper, delegations types.Delegations) (types.DelegationResponses, error) {
	resp := make(types.DelegationResponses, len(delegations))

	for i, del := range delegations {
		delResp, err := delegationToDelegationResponse(ctx, k, del)
		if err != nil {
			return nil, err
		}

		resp[i] = delResp
	}

	return resp, nil
}

func redelegationsToRedelegationResponses(ctx context.Context, k *Keeper, redels types.Redelegations) (types.RedelegationResponses, error) {
	resp := make(types.RedelegationResponses, len(redels))

	for i, redel := range redels {
		valDstAddr, err := k.validatorAddressCodec.StringToBytes(redel.ValidatorDstAddress)
		if err != nil {
			panic(err)
		}

		val, err := k.Validators.Get(ctx, valDstAddr)
		if err != nil {
			return nil, err
		}

		entryResponses := make([]types.RedelegationEntryResponse, len(redel.Entries))
		for j, entry := range redel.Entries {
			balance, _ := val.TokensFromShares(entry.SharesDst).TruncateDecimal()
			entryResponses[j] = types.NewRedelegationEntryResponse(
				entry.CreationHeight,
				entry.CompletionTime,
				entry.SharesDst,
				entry.InitialBalance,
				balance,
				entry.UnbondingId,
			)
		}

		resp[i] = types.NewRedelegationResponse(
			redel.DelegatorAddress,
			redel.ValidatorSrcAddress,
			redel.ValidatorDstAddress,
			entryResponses,
		)
	}

	return resp, nil
}
