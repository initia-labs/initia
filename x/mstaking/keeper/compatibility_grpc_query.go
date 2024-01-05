package keeper

// DONTCOVER

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	cosmostypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// Querier is used as Keeper will have duplicate methods if used directly, and gRPC names take precedence over keeper
type CompatibilityQuerier struct {
	Keeper
}

var _ cosmostypes.QueryServer = CompatibilityQuerier{}

// CosmosParams queries the staking parameters for IBC compatibility
// returns a first bond denom.
func (q CompatibilityQuerier) Params(c context.Context, _ *cosmostypes.QueryParamsRequest) (*cosmostypes.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	params, err := q.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	return &cosmostypes.QueryParamsResponse{Params: cosmostypes.Params{
		UnbondingTime:     params.UnbondingTime,
		MaxValidators:     params.MaxValidators,
		MaxEntries:        params.MaxEntries,
		HistoricalEntries: params.HistoricalEntries,
		BondDenom:         params.BondDenoms[0],
	}}, nil
}

func (q CompatibilityQuerier) Validators(context.Context, *cosmostypes.QueryValidatorsRequest) (*cosmostypes.QueryValidatorsResponse, error) {
	return nil, sdkerrors.ErrNotSupported
}
func (q CompatibilityQuerier) Validator(context.Context, *cosmostypes.QueryValidatorRequest) (*cosmostypes.QueryValidatorResponse, error) {
	return nil, sdkerrors.ErrNotSupported
}
func (q CompatibilityQuerier) ValidatorDelegations(context.Context, *cosmostypes.QueryValidatorDelegationsRequest) (*cosmostypes.QueryValidatorDelegationsResponse, error) {
	return nil, sdkerrors.ErrNotSupported
}
func (q CompatibilityQuerier) ValidatorUnbondingDelegations(context.Context, *cosmostypes.QueryValidatorUnbondingDelegationsRequest) (*cosmostypes.QueryValidatorUnbondingDelegationsResponse, error) {
	return nil, sdkerrors.ErrNotSupported
}
func (q CompatibilityQuerier) Delegation(context.Context, *cosmostypes.QueryDelegationRequest) (*cosmostypes.QueryDelegationResponse, error) {
	return nil, sdkerrors.ErrNotSupported
}
func (q CompatibilityQuerier) UnbondingDelegation(context.Context, *cosmostypes.QueryUnbondingDelegationRequest) (*cosmostypes.QueryUnbondingDelegationResponse, error) {
	return nil, sdkerrors.ErrNotSupported
}
func (q CompatibilityQuerier) DelegatorDelegations(context.Context, *cosmostypes.QueryDelegatorDelegationsRequest) (*cosmostypes.QueryDelegatorDelegationsResponse, error) {
	return nil, sdkerrors.ErrNotSupported
}
func (q CompatibilityQuerier) DelegatorUnbondingDelegations(context.Context, *cosmostypes.QueryDelegatorUnbondingDelegationsRequest) (*cosmostypes.QueryDelegatorUnbondingDelegationsResponse, error) {
	return nil, sdkerrors.ErrNotSupported
}
func (q CompatibilityQuerier) Redelegations(context.Context, *cosmostypes.QueryRedelegationsRequest) (*cosmostypes.QueryRedelegationsResponse, error) {
	return nil, sdkerrors.ErrNotSupported
}
func (q CompatibilityQuerier) DelegatorValidators(context.Context, *cosmostypes.QueryDelegatorValidatorsRequest) (*cosmostypes.QueryDelegatorValidatorsResponse, error) {
	return nil, sdkerrors.ErrNotSupported
}
func (q CompatibilityQuerier) DelegatorValidator(context.Context, *cosmostypes.QueryDelegatorValidatorRequest) (*cosmostypes.QueryDelegatorValidatorResponse, error) {
	return nil, sdkerrors.ErrNotSupported
}
func (q CompatibilityQuerier) HistoricalInfo(context.Context, *cosmostypes.QueryHistoricalInfoRequest) (*cosmostypes.QueryHistoricalInfoResponse, error) {
	return nil, sdkerrors.ErrNotSupported
}
func (q CompatibilityQuerier) Pool(context.Context, *cosmostypes.QueryPoolRequest) (*cosmostypes.QueryPoolResponse, error) {
	return nil, sdkerrors.ErrNotSupported
}
