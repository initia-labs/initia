package gov_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/initia-labs/initia/x/gov"
	"github.com/initia-labs/initia/x/gov/keeper"
	"github.com/stretchr/testify/require"
)

func TestSimpleProposalPassedEndblocker(t *testing.T) {
	app := createAppWithSimpleValidators(t)
	ctx := app.BaseApp.NewContext(false)
	initTime := ctx.BlockHeader().Time

	govMsgSvr := keeper.NewMsgServerImpl(app.GovKeeper)
	propMsg := createTextProposalMsg(t, 100, false)
	_, err := govMsgSvr.SubmitProposal(ctx, propMsg)
	require.NoError(t, err)

	newHeader := ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(time.Minute)
	ctx = ctx.WithBlockHeader(newHeader)

	proposal, err := app.GovKeeper.Proposals.Get(ctx, 1)
	require.NoError(t, err)
	require.True(t, proposal.SubmitTime.Equal(ctx.BlockTime().Add(-time.Minute)))
	require.True(t, proposal.DepositEndTime.Equal(ctx.BlockTime().Add(depositPeriod-time.Minute)))
	require.Equal(t, proposal.Status, v1.StatusDepositPeriod)
	require.Equal(t, proposal.VotingStartTime, (*time.Time)(nil))

	depositMsg := createDepositMsg(t, addrs[0], proposal.Id, minDeposit)
	_, err = govMsgSvr.Deposit(ctx, depositMsg)
	require.NoError(t, err)

	newHeader = ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(time.Minute)
	ctx = ctx.WithBlockHeader(newHeader)

	proposal, err = app.GovKeeper.Proposals.Get(ctx, 1)
	require.NoError(t, err)
	require.True(t, proposal.SubmitTime.Equal(initTime))
	require.True(t, proposal.DepositEndTime.Equal(initTime.Add(depositPeriod)))
	require.Equal(t, proposal.Status, v1.StatusVotingPeriod)
	require.True(t, proposal.VotingStartTime.Equal(ctx.BlockTime().Add(-time.Minute)))
	require.True(t, proposal.VotingEndTime.Equal(ctx.BlockTime().Add(votingPeriod-time.Minute)))

	voteMsg := createVoteMsg(t, addrs[0], proposal.Id, v1.OptionYes)
	_, err = govMsgSvr.Vote(ctx, voteMsg)
	require.NoError(t, err)
	voteMsg = createVoteMsg(t, addrs[1], proposal.Id, v1.OptionYes)
	_, err = govMsgSvr.Vote(ctx, voteMsg)
	require.NoError(t, err)
	voteMsg = createVoteMsg(t, addrs[2], proposal.Id, v1.OptionYes)
	_, err = govMsgSvr.Vote(ctx, voteMsg)
	require.NoError(t, err)

	newHeader = ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(votingPeriod)
	ctx = ctx.WithBlockHeader(newHeader)

	err = gov.EndBlocker(ctx, app.GovKeeper)
	require.NoError(t, err)

	proposal, err = app.GovKeeper.Proposals.Get(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, proposal.Status, v1.StatusPassed)
}

func TestEmergencyProposalPassedEndblocker(t *testing.T) {
	app := createAppWithSimpleValidators(t)
	ctx := app.BaseApp.NewContext(false)
	initTime := ctx.BlockHeader().Time

	govMsgSvr := keeper.NewMsgServerImpl(app.GovKeeper)
	propMsg := createTextProposalMsg(t, emergencyMinDeposit[0].Amount.Int64(), false)
	_, err := govMsgSvr.SubmitProposal(ctx, propMsg)
	require.NoError(t, err)

	newHeader := ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(time.Minute)
	ctx = ctx.WithBlockHeader(newHeader)

	proposal, err := app.GovKeeper.Proposals.Get(ctx, 1)
	require.NoError(t, err)
	require.True(t, proposal.Emergency)
	require.True(t, proposal.EmergencyStartTime.Equal(ctx.BlockTime().Add(-time.Minute)))
	require.True(t, proposal.EmergencyNextTallyTime.Equal(ctx.BlockTime().Add(emergencyTallyInterval-time.Minute)))
	require.True(t, proposal.SubmitTime.Equal(initTime))
	require.True(t, proposal.DepositEndTime.Equal(initTime.Add(depositPeriod)))
	require.Equal(t, proposal.Status, v1.StatusVotingPeriod)
	require.True(t, proposal.VotingStartTime.Equal(ctx.BlockTime().Add(-time.Minute)))
	require.True(t, proposal.VotingEndTime.Equal(ctx.BlockTime().Add(votingPeriod-time.Minute)))

	voteMsg := createVoteMsg(t, addrs[0], proposal.Id, v1.OptionYes)
	_, err = govMsgSvr.Vote(ctx, voteMsg)
	require.NoError(t, err)
	voteMsg = createVoteMsg(t, addrs[1], proposal.Id, v1.OptionYes)
	_, err = govMsgSvr.Vote(ctx, voteMsg)
	require.NoError(t, err)
	voteMsg = createVoteMsg(t, addrs[2], proposal.Id, v1.OptionYes)
	_, err = govMsgSvr.Vote(ctx, voteMsg)
	require.NoError(t, err)
	voteMsg = createVoteMsg(t, addrs[3], proposal.Id, v1.OptionYes)
	_, err = govMsgSvr.Vote(ctx, voteMsg)
	require.NoError(t, err)
	voteMsg = createVoteMsg(t, addrs[4], proposal.Id, v1.OptionYes)
	_, err = govMsgSvr.Vote(ctx, voteMsg)
	require.NoError(t, err)

	newHeader = ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(time.Minute)
	ctx = ctx.WithBlockHeader(newHeader)

	err = gov.EndBlocker(ctx, app.GovKeeper)
	require.NoError(t, err)
	proposal, err = app.GovKeeper.Proposals.Get(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, proposal.Status, v1.StatusVotingPeriod)

	newHeader = ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(time.Minute * 8)
	ctx = ctx.WithBlockHeader(newHeader)

	require.True(t, ctx.BlockHeader().Time.Equal(*proposal.EmergencyNextTallyTime))

	err = gov.EndBlocker(ctx, app.GovKeeper)
	require.NoError(t, err)
	proposal, err = app.GovKeeper.Proposals.Get(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, proposal.Status, v1.StatusPassed)
}

func TestTickSingleProposal(t *testing.T) {
	testCases := []struct {
		name      string
		maxBlocks int
		deposit   sdk.Coin // deposit every block
		votes     []v1.VoteOption
		status    v1.ProposalStatus
		emergency bool
	}{
		{
			name:      "normal passed",
			maxBlocks: 150,
			deposit:   sdk.NewCoin("uinit", math.NewInt(1000)),
			votes: []v1.VoteOption{
				v1.OptionYes,
				v1.OptionYes,
				v1.OptionYes,
				v1.OptionYes,
				v1.OptionYes,
			},
			status:    v1.StatusPassed,
			emergency: false,
		},
		{
			name:      "normal rejected",
			maxBlocks: 150,
			deposit:   sdk.NewCoin("uinit", math.NewInt(1000)),
			votes: []v1.VoteOption{
				v1.OptionYes,
				v1.OptionNo,
				v1.OptionNo,
				v1.OptionNo,
				v1.OptionNo,
			},
			status:    v1.StatusRejected,
			emergency: false,
		},
		{
			name:      "normal voting period",
			maxBlocks: 20,
			deposit:   sdk.NewCoin("uinit", math.NewInt(1000)),
			votes: []v1.VoteOption{
				v1.OptionYes,
				v1.OptionYes,
			},
			status:    v1.StatusVotingPeriod,
			emergency: false,
		},
		{
			name:      "emergency rejected",
			maxBlocks: 150,
			deposit:   sdk.NewCoin("uinit", math.NewInt(5000)),
			votes: []v1.VoteOption{
				v1.OptionNo,
				v1.OptionNo,
				v1.OptionNo,
				v1.OptionNo,
				v1.OptionNo,
			},
			status:    v1.StatusRejected,
			emergency: true,
		},
		{
			name:      "normal deposit period",
			maxBlocks: 2,
			deposit:   sdk.NewCoin("uinit", math.NewInt(100)),
			votes: []v1.VoteOption{
				v1.OptionNo,
				v1.OptionNo,
				v1.OptionNo,
				v1.OptionNo,
				v1.OptionNo,
			},
			status:    v1.StatusDepositPeriod,
			emergency: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			app := createAppWithSimpleValidators(t)
			ctx := app.BaseApp.NewContext(false)

			govMsgSvr := keeper.NewMsgServerImpl(app.GovKeeper)
			propMsg := createTextProposalMsg(t, 100, false)
			_, err := govMsgSvr.SubmitProposal(ctx, propMsg)
			require.NoError(t, err)

			voteCheck := false
			for i := 0; i < tc.maxBlocks; i++ {
				newHeader := ctx.BlockHeader()
				newHeader.Time = ctx.BlockHeader().Time.Add(time.Minute)
				ctx = ctx.WithBlockHeader(newHeader)

				proposal, err := app.GovKeeper.Proposals.Get(ctx, 1)
				require.NoError(t, err)

				if proposal.Status != v1.StatusDepositPeriod && proposal.Status != v1.StatusVotingPeriod {
					break
				}

				depositMsg := createDepositMsg(t, addrs[1], proposal.Id, sdk.Coins{tc.deposit})
				_, err = govMsgSvr.Deposit(ctx, depositMsg)
				require.NoError(t, err)

				if proposal.Status == v1.StatusVotingPeriod && !voteCheck {
					for j, option := range tc.votes {
						voteMsg := createVoteMsg(t, addrs[j], proposal.Id, option)
						_, err = govMsgSvr.Vote(ctx, voteMsg)
						require.NoError(t, err)
					}
					voteCheck = true
				}

				err = gov.EndBlocker(ctx, app.GovKeeper)
				require.NoError(t, err)
			}
			proposal, err := app.GovKeeper.Proposals.Get(ctx, 1)
			require.NoError(t, err)
			require.Equal(t, proposal.Status, tc.status)
			require.Equal(t, proposal.Emergency, tc.emergency)
		})
	}
}

func TestEmergencyProposal_Rejected_VotingPeriodOver(t *testing.T) {
	app := createAppWithSimpleValidators(t)
	ctx := app.BaseApp.NewContext(false)
	initTime := ctx.BlockHeader().Time

	govMsgSvr := keeper.NewMsgServerImpl(app.GovKeeper)
	propMsg := createTextProposalMsg(t, emergencyMinDeposit[0].Amount.Int64(), false)
	_, err := govMsgSvr.SubmitProposal(ctx, propMsg)
	require.NoError(t, err)

	newHeader := ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(time.Minute)
	ctx = ctx.WithBlockHeader(newHeader)

	proposal, err := app.GovKeeper.Proposals.Get(ctx, 1)
	require.NoError(t, err)
	require.True(t, proposal.Emergency)
	require.True(t, proposal.EmergencyStartTime.Equal(ctx.BlockTime().Add(-time.Minute)))
	require.True(t, proposal.EmergencyNextTallyTime.Equal(ctx.BlockTime().Add(emergencyTallyInterval-time.Minute)))
	require.True(t, proposal.SubmitTime.Equal(initTime))
	require.True(t, proposal.DepositEndTime.Equal(initTime.Add(depositPeriod)))
	require.Equal(t, proposal.Status, v1.StatusVotingPeriod)
	require.True(t, proposal.VotingStartTime.Equal(ctx.BlockTime().Add(-time.Minute)))
	require.True(t, proposal.VotingEndTime.Equal(ctx.BlockTime().Add(votingPeriod-time.Minute)))

	// not enough votes

	newHeader = ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(time.Minute)
	ctx = ctx.WithBlockHeader(newHeader)

	err = gov.EndBlocker(ctx, app.GovKeeper)
	require.NoError(t, err)
	proposal, err = app.GovKeeper.Proposals.Get(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, proposal.Status, v1.StatusVotingPeriod)

	// voting period is over; so proposal should be finished
	newHeader = ctx.BlockHeader()
	newHeader.Time = ctx.BlockHeader().Time.Add(votingPeriod)
	ctx = ctx.WithBlockHeader(newHeader)

	err = gov.EndBlocker(ctx, app.GovKeeper)
	require.NoError(t, err)
	proposal, err = app.GovKeeper.Proposals.Get(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, proposal.Status, v1.StatusRejected)
}
