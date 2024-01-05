package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	"github.com/initia-labs/initia/x/gov/keeper"
	"github.com/initia-labs/initia/x/gov/types"
)

func Test_CustomGrpcQuerier_Params(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	params, err := input.GovKeeper.Params.Get(ctx)
	require.NoError(t, err)

	qs := keeper.NewCustomQueryServer(&input.GovKeeper)
	res, err := qs.Params(sdk.WrapSDKContext(ctx), &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, params, res.Params)
}

func Test_CustomGrpcQuerier_EmergencyProposals(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	require.NoError(t, input.GovKeeper.EmergencyProposals.Set(ctx, 1, []byte{1}))
	require.NoError(t, input.GovKeeper.SetProposal(ctx, v1.Proposal{Id: 1}))
	require.NoError(t, input.GovKeeper.EmergencyProposals.Set(ctx, 3, []byte{1}))
	require.NoError(t, input.GovKeeper.SetProposal(ctx, v1.Proposal{Id: 3}))
	require.NoError(t, input.GovKeeper.EmergencyProposals.Set(ctx, 5, []byte{1}))
	require.NoError(t, input.GovKeeper.SetProposal(ctx, v1.Proposal{Id: 5}))
	require.NoError(t, input.GovKeeper.EmergencyProposals.Set(ctx, 6, []byte{1}))
	require.NoError(t, input.GovKeeper.SetProposal(ctx, v1.Proposal{Id: 6}))

	qs := keeper.NewCustomQueryServer(&input.GovKeeper)
	res, err := qs.EmergencyProposals(sdk.WrapSDKContext(ctx), &types.QueryEmergencyProposalsRequest{})
	require.NoError(t, err)

	i := 0
	for _, proposal := range res.Proposals {
		switch i {
		case 0:
			require.Equal(t, uint64(1), proposal.Id)
		case 1:
			require.Equal(t, uint64(3), proposal.Id)
		case 2:
			require.Equal(t, uint64(5), proposal.Id)
		case 3:
			require.Equal(t, uint64(6), proposal.Id)
		}

		i++
	}

	require.Equal(t, 4, i)
}

func Test_CustomGrpcQuerier_LastEmergencyProposalTallyTimestamp(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	now := time.Now().UTC()
	require.NoError(t, input.GovKeeper.LastEmergencyProposalTallyTimestamp.Set(ctx, now))

	qs := keeper.NewCustomQueryServer(&input.GovKeeper)
	res, err := qs.LastEmergencyProposalTallyTimestamp(sdk.WrapSDKContext(ctx), &types.QueryLastEmergencyProposalTallyTimestampRequest{})
	require.NoError(t, err)
	require.Equal(t, now, res.TallyTimestamp)
}
