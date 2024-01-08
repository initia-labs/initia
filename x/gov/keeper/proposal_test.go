package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	customtypes "github.com/initia-labs/initia/x/gov/types"
)

func Test_Emergency_ActivateProposal(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	now := time.Now().UTC()
	depositEndTime := now.Add(time.Hour)
	proposal, err := customtypes.NewProposal(nil, 1, now, depositEndTime, "", "", "", addrs[0], false)
	require.NoError(t, err)

	params, err := input.GovKeeper.Params.Get(ctx)
	require.NoError(t, err)
	proposal.TotalDeposit = params.EmergencyMinDeposit
	input.GovKeeper.ActivateVotingPeriod(ctx, proposal)

	proposal, err = input.GovKeeper.Proposals.Get(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, v1.StatusVotingPeriod, proposal.Status)

	i := 0
	input.GovKeeper.EmergencyProposals.Walk(ctx, nil, func(proposalID uint64, _ []byte) (stop bool, err error) {
		_proposal, err := input.GovKeeper.Proposals.Get(ctx, proposalID)
		require.NoError(t, err)
		require.Equal(t, proposal, _proposal)
		i++
		return false, nil
	})
	require.Equal(t, 1, i)
}

func Test_NoEmergency_ActivateProposal(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	now := time.Now().UTC()
	depositEndTime := now.Add(time.Hour)
	proposal, err := customtypes.NewProposal(nil, 1, now, depositEndTime, "", "", "", addrs[0], false)
	require.NoError(t, err)

	params, err := input.GovKeeper.Params.Get(ctx)
	require.NoError(t, err)
	proposal.TotalDeposit = params.MinDeposit
	input.GovKeeper.ActivateVotingPeriod(ctx, proposal)

	proposal, err = input.GovKeeper.Proposals.Get(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, v1.StatusVotingPeriod, proposal.Status)

	input.GovKeeper.EmergencyProposals.Walk(ctx, nil, func(proposalID uint64, _ []byte) (stop bool, err error) {
		require.FailNow(t, "should not enter")
		return false, nil
	})
}
