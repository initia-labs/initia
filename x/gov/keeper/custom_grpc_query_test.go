package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
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

func Test_CustomGrpcQuerier_TallyResult(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	now := time.Now().UTC()
	setupVesting(t, ctx, input, now)

	proposal, err := input.GovKeeper.SubmitProposal(ctx, nil, "", "test", "description", addrs[0], false)
	require.NoError(t, err)

	proposalID := proposal.Id
	proposal.Status = v1.StatusVotingPeriod
	err = input.GovKeeper.SetProposal(ctx, proposal)
	require.NoError(t, err)

	proposal, err = input.GovKeeper.Proposals.Get(ctx, proposalID)
	require.NoError(t, err)

	params, err := input.GovKeeper.Params.Get(ctx)
	require.NoError(t, err)

	quorumReached, passed, _, _, err := input.GovKeeper.Tally(ctx, params, proposal)
	require.NoError(t, err)
	require.False(t, quorumReached)
	require.False(t, passed)

	valAddr1 := createValidatorWithCoin(ctx, input,
		sdk.NewCoins(sdk.NewInt64Coin(bondDenom, 100_000_000)),
		sdk.NewCoins(sdk.NewInt64Coin(bondDenom, 100_000_000)),
		1,
	)
	valAddr2 := createValidatorWithCoin(ctx, input,
		sdk.NewCoins(sdk.NewInt64Coin(bondDenom, 100_000_000)),
		sdk.NewCoins(sdk.NewInt64Coin(bondDenom, 100_000_000)),
		2,
	)

	voterAddr1 := sdk.AccAddress(valAddr1)
	voterAddr2 := sdk.AccAddress(valAddr2)

	// vote yes
	err = input.GovKeeper.AddVote(ctx, proposalID, voterAddr1, v1.WeightedVoteOptions{
		{
			Option: v1.OptionYes,
			Weight: "1",
		},
	}, "")
	require.NoError(t, err)

	// vote no
	err = input.GovKeeper.AddVote(ctx, proposalID, voterAddr2, v1.WeightedVoteOptions{
		{
			Option: v1.OptionNo,
			Weight: "1",
		},
	}, "")
	require.NoError(t, err)

	// add vesting vote
	vestingVoter := addrs[1]
	err = input.GovKeeper.AddVote(ctx, proposalID, vestingVoter, v1.WeightedVoteOptions{
		{
			Option: v1.OptionYes,
			Weight: "1",
		},
	}, "")
	require.NoError(t, err)

	// 15 minutes passed
	ctx = ctx.WithBlockTime(now.Add(time.Minute * 15))
	cacheCtx, _ := ctx.CacheContext()

	quorumReached, passed, burnDeposits, tallyResults, err := input.GovKeeper.Tally(cacheCtx, params, proposal)
	require.NoError(t, err)
	require.True(t, quorumReached)
	require.True(t, passed)
	require.False(t, burnDeposits)
	require.Equal(t, tallyResults.V1TallyResult.YesCount, math.LegacyNewDec(1_500_000+100_000_000).TruncateInt().String())
	require.Equal(t, tallyResults.V1TallyResult.NoCount, math.LegacyNewDec(100_000_000).TruncateInt().String())

	qs := keeper.NewCustomQueryServer(&input.GovKeeper)
	res, err := qs.TallyResult(ctx, &types.QueryTallyResultRequest{ProposalId: proposalID})
	require.NoError(t, err)
	require.Equal(t, tallyResults, res.TallyResult)
}
