package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/initia-labs/initia/x/reward/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func Test_GRPCParams(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	params, err := input.RewardKeeper.Params(sdk.WrapSDKContext(ctx), &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, input.RewardKeeper.GetParams(ctx), params.Params)
}

func Test_GRPCAnnualProvisions(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	annualProvisions, err := input.RewardKeeper.AnnualProvisions(sdk.WrapSDKContext(ctx), &types.QueryAnnualProvisionsRequest{})
	require.NoError(t, err)
	require.Equal(t, input.RewardKeeper.GetAnnualProvisions(ctx), annualProvisions.AnnualProvisions)
}

func Test_GRPCLastDilutionTimestamp(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	now := time.Now().UTC()
	input.RewardKeeper.SetLastDilutionTimestamp(ctx, now)

	lastDilutionTimestamp, err := input.RewardKeeper.LastDilutionTimestamp(sdk.WrapSDKContext(ctx), &types.QueryLastDilutionTimestampRequest{})
	require.NoError(t, err)
	require.Equal(t, now, lastDilutionTimestamp.LastDilutionTimestamp)
}
