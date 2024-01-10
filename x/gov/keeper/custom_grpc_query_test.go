package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/initia-labs/initia/x/gov/keeper"
	"github.com/initia-labs/initia/x/gov/types"
)

func Test_CustomGrpcQuerier_Params(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	params, err := input.GovKeeper.Params.Get(ctx)
	require.NoError(t, err)

	qs := keeper.NewCustomQueryServer(&input.GovKeeper)
	res, err := qs.Params(ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.Equal(t, params, res.Params)
}

func Test_CustomGrpcQuerier_EmergencyProposals(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	require.NoError(t, input.GovKeeper.EmergencyProposals.Set(ctx, 1, []byte{1}))
	require.NoError(t, input.GovKeeper.SetProposal(ctx, types.Proposal{Id: 1, Emergency: true}))
	require.NoError(t, input.GovKeeper.EmergencyProposals.Set(ctx, 3, []byte{1}))
	require.NoError(t, input.GovKeeper.SetProposal(ctx, types.Proposal{Id: 3, Emergency: true}))
	require.NoError(t, input.GovKeeper.EmergencyProposals.Set(ctx, 5, []byte{1}))
	require.NoError(t, input.GovKeeper.SetProposal(ctx, types.Proposal{Id: 5, Emergency: true}))
	require.NoError(t, input.GovKeeper.EmergencyProposals.Set(ctx, 6, []byte{1}))
	require.NoError(t, input.GovKeeper.SetProposal(ctx, types.Proposal{Id: 6, Emergency: true}))
	require.NoError(t, input.GovKeeper.SetProposal(ctx, types.Proposal{Id: 8}))

	qs := keeper.NewCustomQueryServer(&input.GovKeeper)
	res, err := qs.EmergencyProposals(ctx, &types.QueryEmergencyProposalsRequest{})
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
		case 4:
			require.FailNow(t, "should not exist")
		}

		require.True(t, proposal.Emergency)

		i++
	}

	require.Equal(t, 4, i)
}

func Test_CustomGrpcQuerier_Proposals(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	require.NoError(t, input.GovKeeper.SetProposal(ctx, types.Proposal{Id: 1}))
	require.NoError(t, input.GovKeeper.EmergencyProposals.Set(ctx, 3, []byte{1}))
	require.NoError(t, input.GovKeeper.SetProposal(ctx, types.Proposal{Id: 3, Emergency: true}))
	require.NoError(t, input.GovKeeper.SetProposal(ctx, types.Proposal{Id: 5}))
	require.NoError(t, input.GovKeeper.EmergencyProposals.Set(ctx, 6, []byte{1}))
	require.NoError(t, input.GovKeeper.SetProposal(ctx, types.Proposal{Id: 6, Emergency: true}))

	qs := keeper.NewCustomQueryServer(&input.GovKeeper)
	res, err := qs.Proposals(ctx, &types.QueryProposalsRequest{})
	require.NoError(t, err)

	i := 0
	for _, proposal := range res.Proposals {
		switch i {
		case 0:
			require.Equal(t, uint64(1), proposal.Id)
			require.False(t, proposal.Emergency)
		case 1:
			require.Equal(t, uint64(3), proposal.Id)
			require.True(t, proposal.Emergency)
		case 2:
			require.Equal(t, uint64(5), proposal.Id)
			require.False(t, proposal.Emergency)
		case 3:
			require.Equal(t, uint64(6), proposal.Id)
			require.True(t, proposal.Emergency)
		}

		i++
	}

	require.Equal(t, 4, i)
}

func Test_CustomGrpcQuerier_Proposal(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	require.NoError(t, input.GovKeeper.EmergencyProposals.Set(ctx, 1, []byte{1}))
	require.NoError(t, input.GovKeeper.SetProposal(ctx, types.Proposal{Id: 1, Emergency: true}))
	require.NoError(t, input.GovKeeper.EmergencyProposals.Set(ctx, 3, []byte{1}))
	require.NoError(t, input.GovKeeper.SetProposal(ctx, types.Proposal{Id: 3, Emergency: true}))
	require.NoError(t, input.GovKeeper.SetProposal(ctx, types.Proposal{Id: 5}))
	require.NoError(t, input.GovKeeper.EmergencyProposals.Set(ctx, 6, []byte{1}))
	require.NoError(t, input.GovKeeper.SetProposal(ctx, types.Proposal{Id: 6, Emergency: true}))

	qs := keeper.NewCustomQueryServer(&input.GovKeeper)
	res, err := qs.Proposal(ctx, &types.QueryProposalRequest{ProposalId: 5})
	require.NoError(t, err)
	require.Equal(t, res.Proposal.Id, uint64(5))
	require.False(t, res.Proposal.Emergency)
}
