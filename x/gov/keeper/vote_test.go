package keeper_test

import (
	"testing"

	"cosmossdk.io/collections"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
)

func TestSimpleVote(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	proposal, err := input.GovKeeper.SubmitProposal(ctx, nil, "", "", "", addrs[0], false)
	require.NoError(t, err)
	err = input.GovKeeper.VotingPeriodProposals.Set(ctx, proposal.Id, []byte{1})
	require.NoError(t, err)
	proposal.Status = v1.StatusVotingPeriod
	err = input.GovKeeper.Proposals.Set(ctx, proposal.Id, proposal)
	require.NoError(t, err)

	option1 := &v1.WeightedVoteOption{Option: v1.OptionYes, Weight: "1"}
	option2 := &v1.WeightedVoteOption{Option: v1.OptionNoWithVeto, Weight: "1"}
	option3 := &v1.WeightedVoteOption{Option: v1.OptionNo, Weight: "1"}

	err = input.GovKeeper.AddVote(ctx, proposal.Id, addrs[0], []*v1.WeightedVoteOption{option1}, "")
	require.NoError(t, err)
	err = input.GovKeeper.AddVote(ctx, proposal.Id, addrs[1], []*v1.WeightedVoteOption{option2}, "")
	require.NoError(t, err)
	err = input.GovKeeper.AddVote(ctx, proposal.Id, addrs[2], []*v1.WeightedVoteOption{option3}, "")
	require.NoError(t, err)

	input.GovKeeper.Votes.Walk(ctx, nil, func(key collections.Pair[uint64, types.AccAddress], value v1.Vote) (stop bool, err error) {
		switch key.K2().String() {
		case addrs[0].String():
			require.Equal(t, value.Options, []*v1.WeightedVoteOption{option1})
		case addrs[1].String():
			require.Equal(t, value.Options, []*v1.WeightedVoteOption{option2})
		case addrs[2].String():
			require.Equal(t, value.Options, []*v1.WeightedVoteOption{option3})
		}
		return false, nil
	})
}

func TestVoteToInvalidProposer(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	proposal, err := input.GovKeeper.SubmitProposal(ctx, nil, "", "", "", addrs[0], false)
	require.NoError(t, err)

	proposal.Status = v1.StatusDepositPeriod
	err = input.GovKeeper.Proposals.Set(ctx, proposal.Id, proposal)
	require.NoError(t, err)
	err = input.GovKeeper.AddVote(ctx, proposal.Id, addrs[0], v1.WeightedVoteOptions{&v1.WeightedVoteOption{Option: v1.OptionYes, Weight: "1"}}, "")
	require.Error(t, err)

	proposal.Status = v1.StatusPassed
	err = input.GovKeeper.Proposals.Set(ctx, proposal.Id, proposal)
	require.NoError(t, err)
	err = input.GovKeeper.AddVote(ctx, proposal.Id, addrs[0], v1.WeightedVoteOptions{&v1.WeightedVoteOption{Option: v1.OptionYes, Weight: "1"}}, "")
	require.Error(t, err)

	err = input.GovKeeper.AddVote(ctx, 2, addrs[0], v1.WeightedVoteOptions{&v1.WeightedVoteOption{Option: v1.OptionYes, Weight: "1"}}, "")
	require.Error(t, err)
}
