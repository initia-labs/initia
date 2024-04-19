package keeper_test

import (
	"fmt"
	"testing"
	"time"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	customtypes "github.com/initia-labs/initia/x/gov/types"
)

func TestEmergencyActivateProposal(t *testing.T) {
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
	input.GovKeeper.ActivateEmergencyProposal(ctx, proposal)

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

	i = 0
	input.GovKeeper.EmergencyProposalsQueue.Walk(ctx, nil, func(key collections.Pair[time.Time, uint64], proposalID uint64) (stop bool, err error) {
		require.Equal(t, key.K2(), proposalID)
		_proposal, err := input.GovKeeper.Proposals.Get(ctx, key.K2())
		require.NoError(t, err)
		require.Equal(t, _proposal.Id, proposalID)
		require.Equal(t, proposal, _proposal)
		require.Equal(t, _proposal.Emergency, true)

		require.Equal(t, _proposal.EmergencyNextTallyTime.Equal(key.K1()), true)
		require.Equal(t, _proposal.EmergencyNextTallyTime.Equal(proposal.EmergencyStartTime.Add(params.EmergencyTallyInterval)), true)
		i++
		return false, nil
	})
	require.Equal(t, 1, i)
}

func TestSubmitProposal(t *testing.T) {
	depositPeriod := time.Hour

	ctx, input := createDefaultTestInput(t)
	params, err := input.GovKeeper.Params.Get(ctx)
	require.NoError(t, err)
	params.MaxDepositPeriod = depositPeriod

	err = input.GovKeeper.Params.Set(ctx, params)
	require.NoError(t, err)

	proposal, err := input.GovKeeper.SubmitProposal(ctx, nil, "", "", "", addrs[0], false)
	require.NoError(t, err)
	require.Equal(t, proposal.Id, uint64(1))

	require.Equal(t, proposal.Proposer, addrs[0].String())
	require.True(t, proposal.DepositEndTime.Equal(ctx.BlockTime().Add(depositPeriod)))
	require.Equal(t, proposal.VotingStartTime, (*time.Time)(nil))
	require.False(t, proposal.Emergency)
}

func TestDeleteProposal(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	proposal, err := input.GovKeeper.SubmitProposal(ctx, nil, "", "", "", addrs[0], false)
	require.NoError(t, err)

	err = input.GovKeeper.DeleteProposal(ctx, proposal.Id)
	require.NoError(t, err)

	err = input.GovKeeper.DeleteProposal(ctx, proposal.Id)
	require.Error(t, err)

	err = input.GovKeeper.DeleteProposal(ctx, 2)
	require.Error(t, err)
}

func TestCancelProposal(t *testing.T) {
	minDeposit := sdk.NewCoin("uinit", math.NewInt(100000))

	testCases := []struct {
		name          string
		deposit       sdk.Coin
		proposer      sdk.AccAddress
		proposalId    uint64
		expectedError bool
	}{
		{
			name:          "normal cancel proposal during deposit period",
			deposit:       sdk.NewCoin("uinit", math.NewInt(1000)),
			proposer:      addrs[0],
			proposalId:    1,
			expectedError: false,
		},
		{
			name:          "normal cancel proposal during voting period",
			deposit:       sdk.NewCoin("uinit", math.NewInt(100000)),
			proposer:      addrs[0],
			proposalId:    1,
			expectedError: false,
		},
		{
			name:          "weird proposer",
			deposit:       sdk.NewCoin("uinit", math.NewInt(100000)),
			proposer:      addrs[1],
			proposalId:    1,
			expectedError: true,
		},
		{
			name:          "weird proposal id",
			deposit:       sdk.NewCoin("uinit", math.NewInt(100000)),
			proposer:      addrs[0],
			proposalId:    2,
			expectedError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, input := createDefaultTestInput(t)
			params, err := input.GovKeeper.Params.Get(ctx)
			require.NoError(t, err)

			params.MinDeposit = sdk.Coins{minDeposit}
			err = input.GovKeeper.Params.Set(ctx, params)
			require.NoError(t, err)

			proposal, err := input.GovKeeper.SubmitProposal(ctx, nil, "", "", "", addrs[0], false)
			require.NoError(t, err)
			proposal.TotalDeposit = []sdk.Coin{tc.deposit}

			if sdk.NewCoins(proposal.TotalDeposit...).IsAllGTE(params.MinDeposit) {
				err = input.GovKeeper.ActivateVotingPeriod(ctx, proposal)
				require.NoError(t, err)
			}

			err = input.GovKeeper.CancelProposal(ctx, tc.proposalId, tc.proposer.String())
			if tc.expectedError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			_, err = input.GovKeeper.Proposals.Get(ctx, 1)
			require.Error(t, err)

			input.GovKeeper.ActiveProposalsQueue.Walk(ctx, nil, func(key collections.Pair[time.Time, uint64], proposalID uint64) (stop bool, err error) {
				require.FailNow(t, "should not exist")
				return false, nil
			})

			input.GovKeeper.InactiveProposalsQueue.Walk(ctx, nil, func(key collections.Pair[time.Time, uint64], proposalID uint64) (stop bool, err error) {
				require.FailNow(t, "should not exist")
				return false, nil
			})

			input.GovKeeper.EmergencyProposalsQueue.Walk(ctx, nil, func(key collections.Pair[time.Time, uint64], proposalID uint64) (stop bool, err error) {
				require.FailNow(t, "should not exist")
				return false, nil
			})

			input.GovKeeper.EmergencyProposals.Walk(ctx, nil, func(key uint64, _ []byte) (stop bool, err error) {
				require.FailNow(t, "should not exist")
				return false, nil
			})
		})
	}
}

type queueResult struct {
	inactive  int
	active    int
	emergency int
}

func (qr queueResult) String() string {
	return fmt.Sprintf("inactive:%d active:%d emergency:%d", qr.inactive, qr.active, qr.emergency)
}

func TestQueueManagement(t *testing.T) {
	minDeposit := sdk.NewCoin("uinit", math.NewInt(10000))
	emergencyMinDeposit := sdk.NewCoin("uinit", math.NewInt(100000))

	testCases := []struct {
		deposits []sdk.Coin
		results  queueResult
	}{
		{
			deposits: sdk.Coins{
				sdk.NewCoin("uinit", math.NewInt(1000)),
				sdk.NewCoin("uinit", math.NewInt(1000)),
				sdk.NewCoin("uinit", math.NewInt(1000)),
				sdk.NewCoin("uinit", math.NewInt(9998)),
				sdk.NewCoin("uinit", math.NewInt(9999)),
			},
			results: queueResult{
				inactive:  5,
				active:    0,
				emergency: 0,
			},
		},
		{
			deposits: sdk.Coins{
				sdk.NewCoin("uinit", math.NewInt(1000)),
				sdk.NewCoin("uinit", math.NewInt(1000)),
				sdk.NewCoin("uinit", math.NewInt(10000)),
				sdk.NewCoin("uinit", math.NewInt(10000)),
				sdk.NewCoin("uinit", math.NewInt(1000)),
			},
			results: queueResult{
				inactive:  3,
				active:    2,
				emergency: 0,
			},
		},
		{
			deposits: sdk.Coins{
				sdk.NewCoin("uinit", math.NewInt(1000)),
				sdk.NewCoin("uinit", math.NewInt(1000)),
				sdk.NewCoin("uinit", math.NewInt(100000)),
				sdk.NewCoin("uinit", math.NewInt(100001)),
				sdk.NewCoin("uinit", math.NewInt(99999)),
			},
			results: queueResult{
				inactive:  2,
				active:    3,
				emergency: 2,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.results.String(), func(t *testing.T) {
			ctx, input := createDefaultTestInput(t)
			params, err := input.GovKeeper.Params.Get(ctx)
			require.NoError(t, err)

			params.MinDeposit = sdk.Coins{minDeposit}
			params.EmergencyMinDeposit = sdk.Coins{emergencyMinDeposit}
			err = input.GovKeeper.Params.Set(ctx, params)
			require.NoError(t, err)

			for _, deposit := range tc.deposits {
				proposal, err := input.GovKeeper.SubmitProposal(ctx, nil, "", "", "", addrs[0], false)
				require.NoError(t, err)
				proposal.TotalDeposit = []sdk.Coin{deposit}

				if sdk.NewCoins(proposal.TotalDeposit...).IsAllGTE(params.MinDeposit) {
					err = input.GovKeeper.ActivateVotingPeriod(ctx, proposal)
					require.NoError(t, err)
				}

				if sdk.NewCoins(proposal.TotalDeposit...).IsAllGTE(params.EmergencyMinDeposit) {
					err = input.GovKeeper.ActivateEmergencyProposal(ctx, proposal)
					require.NoError(t, err)
				}
			}

			qr := queueResult{}
			input.GovKeeper.ActiveProposalsQueue.Walk(ctx, nil, func(key collections.Pair[time.Time, uint64], proposalID uint64) (stop bool, err error) {
				qr.active++
				return false, nil
			})

			input.GovKeeper.InactiveProposalsQueue.Walk(ctx, nil, func(key collections.Pair[time.Time, uint64], proposalID uint64) (stop bool, err error) {
				qr.inactive++
				return false, nil
			})

			input.GovKeeper.EmergencyProposalsQueue.Walk(ctx, nil, func(key collections.Pair[time.Time, uint64], proposalID uint64) (stop bool, err error) {
				qr.emergency++
				return false, nil
			})

			numEmergencyProposals := 0
			input.GovKeeper.EmergencyProposals.Walk(ctx, nil, func(key uint64, _ []byte) (stop bool, err error) {
				numEmergencyProposals++
				return false, nil
			})

			require.Equal(t, qr.emergency, numEmergencyProposals)
			require.Equal(t, tc.results, qr)
		})
	}
}
