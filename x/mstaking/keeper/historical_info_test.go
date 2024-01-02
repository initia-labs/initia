package keeper_test

import (
	"testing"

	"cosmossdk.io/collections"
	"github.com/stretchr/testify/require"

	cosmostypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func Test_HistoricalInfo(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	params, err := input.StakingKeeper.GetParams(ctx)
	require.NoError(t, err)

	params.HistoricalEntries = 2
	input.StakingKeeper.SetParams(ctx, params)

	input.StakingKeeper.TrackHistoricalInfo(ctx.WithBlockHeight(1))
	input.StakingKeeper.TrackHistoricalInfo(ctx.WithBlockHeight(2))
	input.StakingKeeper.TrackHistoricalInfo(ctx.WithBlockHeight(3))

	_, err = input.StakingKeeper.GetHistoricalInfo(ctx, 1)
	require.ErrorIs(t, err, collections.ErrNotFound)

	historicalInfo, err := input.StakingKeeper.GetHistoricalInfo(ctx, 2)
	require.NoError(t, err)
	require.Equal(t, cosmostypes.HistoricalInfo{
		Header: ctx.WithBlockHeight(2).BlockHeader(),
		Valset: nil,
	}, historicalInfo)

	historicalInfo, err = input.StakingKeeper.GetHistoricalInfo(ctx, 3)
	require.NoError(t, err)
	require.Equal(t, cosmostypes.HistoricalInfo{
		Header: ctx.WithBlockHeight(3).BlockHeader(),
		Valset: nil,
	}, historicalInfo)
}
