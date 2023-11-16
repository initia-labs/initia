package keeper_test

import (
	"testing"
	"time"

	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/stretchr/testify/require"
)

func Test_Emergency_ActivateProposal(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	now := time.Now().UTC()
	depositEndTime := now.Add(time.Hour)
	proposal, err := v1.NewProposal(nil, 1, now, depositEndTime, "", "", "", addrs[0])
	require.NoError(t, err)

	params := input.GovKeeper.GetParams(ctx)
	proposal.TotalDeposit = params.EmergencyMinDeposit
	input.GovKeeper.ActivateVotingPeriod(ctx, proposal)

	proposal, found := input.GovKeeper.GetProposal(ctx, 1)
	require.True(t, found)
	require.Equal(t, v1.StatusVotingPeriod, proposal.Status)

	i := 0
	input.GovKeeper.IterateEmergencyProposals(ctx, func(_proposal v1.Proposal) (stop bool) {
		require.Equal(t, proposal, _proposal)
		i++
		return false
	})
	require.Equal(t, 1, i)
}

func Test_NoEmergency_ActivateProposal(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	now := time.Now().UTC()
	depositEndTime := now.Add(time.Hour)
	proposal, err := v1.NewProposal(nil, 1, now, depositEndTime, "", "", "", addrs[0])
	require.NoError(t, err)

	params := input.GovKeeper.GetParams(ctx)
	proposal.TotalDeposit = params.MinDeposit
	input.GovKeeper.ActivateVotingPeriod(ctx, proposal)

	proposal, found := input.GovKeeper.GetProposal(ctx, 1)
	require.True(t, found)
	require.Equal(t, v1.StatusVotingPeriod, proposal.Status)

	input.GovKeeper.IterateEmergencyProposals(ctx, func(_proposal v1.Proposal) (stop bool) {
		require.FailNow(t, "should not enter")
		return false
	})
}
