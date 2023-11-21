package keeper

import (
	"context"

	"github.com/initia-labs/initia/x/reward/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ types.QueryServer = Keeper{}

// Params returns params of the reward module.
func (k Keeper) Params(c context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	params := k.GetParams(ctx)

	return &types.QueryParamsResponse{Params: params}, nil
}

// AnnualProvisions returns calculated annual rewards.
func (k Keeper) AnnualProvisions(c context.Context, _ *types.QueryAnnualProvisionsRequest) (*types.QueryAnnualProvisionsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	annualProvisions := k.GetAnnualProvisions(ctx)
	return &types.QueryAnnualProvisionsResponse{AnnualProvisions: annualProvisions}, nil
}

// LastDilutionTimestamp returns calculated annual rewards.
func (k Keeper) LastDilutionTimestamp(c context.Context, _ *types.QueryLastDilutionTimestampRequest) (*types.QueryLastDilutionTimestampResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	lastDilutionTimestamp := k.GetLastDilutionTimestamp(ctx)
	return &types.QueryLastDilutionTimestampResponse{LastDilutionTimestamp: lastDilutionTimestamp}, nil
}
