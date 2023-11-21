package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
)

func Test_InsertEmergencyProposalQueue(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	input.GovKeeper.InsertEmergencyProposalQueue(ctx, 1)
	input.GovKeeper.SetProposal(ctx, v1.Proposal{Id: 1})
	input.GovKeeper.InsertEmergencyProposalQueue(ctx, 3)
	input.GovKeeper.SetProposal(ctx, v1.Proposal{Id: 3})
	input.GovKeeper.InsertEmergencyProposalQueue(ctx, 5)
	input.GovKeeper.SetProposal(ctx, v1.Proposal{Id: 5})
	input.GovKeeper.InsertEmergencyProposalQueue(ctx, 6)
	input.GovKeeper.SetProposal(ctx, v1.Proposal{Id: 6})

	// remove 5 from emergency queue
	input.GovKeeper.RemoveFromEmergencyProposalQueue(ctx, 5)

	i := 0
	input.GovKeeper.IterateEmergencyProposals(ctx, func(proposal v1.Proposal) (stop bool) {
		switch i {
		case 0:
			require.Equal(t, uint64(1), proposal.Id)
		case 1:
			require.Equal(t, uint64(3), proposal.Id)
		case 2:
			require.Equal(t, uint64(6), proposal.Id)
		}

		i++
		return false
	})
	require.Equal(t, 3, i)
}

func Test_LastEmergencyProposalTallyTimestamp(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	ctx = ctx.WithBlockTime(time.Now().UTC())
	input.GovKeeper.RecordLastEmergencyProposalTallyTimestamp(ctx)
	require.Equal(t, ctx.BlockTime(), input.GovKeeper.GetLastEmergencyProposalTallyTimestamp(ctx))

	hourLater := time.Now().UTC().Add(time.Hour)
	input.GovKeeper.SetLastEmergencyProposalTallyTimestamp(ctx, hourLater)
	require.Equal(t, hourLater, input.GovKeeper.GetLastEmergencyProposalTallyTimestamp(ctx))
}
