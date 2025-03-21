package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/initia-labs/initia/x/dynamic-fee/keeper"
	"github.com/initia-labs/initia/x/dynamic-fee/types"
)

func TestParams(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	querier := keeper.NewQuerier(&input.DynamicFeeKeeper)
	params, err := querier.Params(ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)

	expectedParams, err := input.DynamicFeeKeeper.GetParams(ctx)
	require.NoError(t, err)
	require.Equal(t, expectedParams, params.Params)
}
