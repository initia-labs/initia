package keeper

import (
	"context"

	"github.com/initia-labs/initia/v1/x/reward/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ types.QueryServer = QueryServer{}

type QueryServer struct {
	*Keeper
}

func NewQueryServerImpl(k *Keeper) QueryServer {
	return QueryServer{k}
}

// Params returns params of the reward module.
func (qs QueryServer) Params(c context.Context, _ *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	params, err := qs.GetParams(ctx)

	return &types.QueryParamsResponse{Params: params}, err
}

// AnnualProvisions returns calculated annual rewards.
func (qs QueryServer) AnnualProvisions(c context.Context, _ *types.QueryAnnualProvisionsRequest) (*types.QueryAnnualProvisionsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	annualProvisions, err := qs.GetAnnualProvisions(ctx)
	return &types.QueryAnnualProvisionsResponse{AnnualProvisions: annualProvisions}, err
}

// LastDilutionTimestamp returns calculated annual rewards.
func (qs QueryServer) LastDilutionTimestamp(c context.Context, _ *types.QueryLastDilutionTimestampRequest) (*types.QueryLastDilutionTimestampResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	lastDilutionTimestamp, err := qs.GetLastDilutionTimestamp(ctx)
	return &types.QueryLastDilutionTimestampResponse{LastDilutionTimestamp: lastDilutionTimestamp}, err
}
