package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/initia-labs/initia/x/reward/keeper"
	"github.com/initia-labs/initia/x/reward/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func Test_GRPCParams(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	qs := keeper.NewQueryServerImpl(&input.RewardKeeper)
	params, err := qs.Params(ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)

	_params, err := input.RewardKeeper.GetParams(ctx)
	require.NoError(t, err)
	require.Equal(t, _params, params.Params)
}

func Test_GRPCAnnualProvisions(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	qs := keeper.NewQueryServerImpl(&input.RewardKeeper)
	annualProvisions, err := qs.AnnualProvisions(sdk.WrapSDKContext(ctx), &types.QueryAnnualProvisionsRequest{})
	require.NoError(t, err)
	_annualProvisions, err := input.RewardKeeper.GetAnnualProvisions(ctx)
	require.NoError(t, err)
	require.Equal(t, _annualProvisions, annualProvisions.AnnualProvisions)
}

func Test_GRPCLastDilutionTimestamp(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	now := time.Now().UTC()
	input.RewardKeeper.SetLastDilutionTimestamp(ctx, now)

	qs := keeper.NewQueryServerImpl(&input.RewardKeeper)
	lastDilutionTimestamp, err := qs.LastDilutionTimestamp(sdk.WrapSDKContext(ctx), &types.QueryLastDilutionTimestampRequest{})
	require.NoError(t, err)
	require.Equal(t, now, lastDilutionTimestamp.LastDilutionTimestamp)
}
