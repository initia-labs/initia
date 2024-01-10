package keeper_test

import (
	"sort"
	"testing"

	"cosmossdk.io/collections"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/gov/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/stretchr/testify/require"
)

func TestSimpleDeposits(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	deposit1 := v1.Deposit{ProposalId: 2, Depositor: addrs[0].String(), Amount: []sdk.Coin{sdk.NewCoin(bondDenom, math.NewInt(100))}}
	deposit2 := v1.Deposit{ProposalId: 2, Depositor: addrs[1].String(), Amount: []sdk.Coin{sdk.NewCoin(bondDenom, math.NewInt(50))}}

	proposalDeposits := make(v1.Deposits, 0, 2)
	proposalDeposits = append(proposalDeposits, &deposit1)
	proposalDeposits = append(proposalDeposits, &deposit2)
	sort.Slice(proposalDeposits, func(i, j int) bool { return proposalDeposits[i].GetDepositor() < proposalDeposits[j].GetDepositor() })

	input.GovKeeper.SetDeposit(ctx, deposit1)
	input.GovKeeper.SetDeposit(ctx, deposit2)

	deposits1, err := input.GovKeeper.GetDeposits(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, deposits1, (v1.Deposits)(nil))

	deposits2, err := input.GovKeeper.GetDeposits(ctx, 2)
	require.NoError(t, err)
	sort.Slice(deposits2, func(i, j int) bool { return deposits2[i].GetDepositor() < deposits2[j].GetDepositor() })

	require.Equal(t, deposits2, proposalDeposits)

	err = input.GovKeeper.IterateDeposits(ctx, 1, func(key collections.Pair[uint64, sdk.AccAddress], value v1.Deposit) (bool, error) {
		require.FailNow(t, "should not be called")
		return false, nil
	})
	require.NoError(t, err)

	err = input.GovKeeper.IterateDeposits(ctx, 2, func(key collections.Pair[uint64, sdk.AccAddress], value v1.Deposit) (bool, error) {
		require.Equal(t, key.K1(), uint64(2))
		switch key.K2().String() {
		case addrs[0].String():
			require.Equal(t, value, deposit1)
		case addrs[1].String():
			require.Equal(t, value, deposit2)
		default:
			require.FailNow(t, "not registered deposit")
		}
		return false, nil
	})
	require.NoError(t, err)

}

func TestSimpleBurnDeposits(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	input.Faucet.Fund(ctx, addrs[0], sdk.NewCoin(bondDenom, math.NewInt(10000)))
	input.Faucet.Fund(ctx, addrs[1], sdk.NewCoin(bondDenom, math.NewInt(10000)))
	input.Faucet.Fund(ctx, addrs[2], sdk.NewCoin(bondDenom, math.NewInt(10000)))

	deposit1Amount := []sdk.Coin{sdk.NewCoin(bondDenom, math.NewInt(100))}
	deposit1 := v1.Deposit{ProposalId: 2, Depositor: addrs[0].String(), Amount: deposit1Amount}

	deposit2Amount := []sdk.Coin{sdk.NewCoin(bondDenom, math.NewInt(50))}
	deposit2 := v1.Deposit{ProposalId: 2, Depositor: addrs[1].String(), Amount: deposit2Amount}

	proposal1Deposits := make(v1.Deposits, 0, 2)
	proposal1Deposits = append(proposal1Deposits, &deposit1)
	proposal1Deposits = append(proposal1Deposits, &deposit2)
	sort.Slice(proposal1Deposits, func(i, j int) bool { return proposal1Deposits[i].GetDepositor() < proposal1Deposits[j].GetDepositor() })

	deposit3Amount := []sdk.Coin{sdk.NewCoin(bondDenom, math.NewInt(1000))}
	deposit3 := v1.Deposit{ProposalId: 5, Depositor: addrs[2].String(), Amount: deposit3Amount}

	proposal2Deposits := make(v1.Deposits, 0, 1)
	proposal2Deposits = append(proposal2Deposits, &deposit3)
	sort.Slice(proposal2Deposits, func(i, j int) bool { return proposal2Deposits[i].GetDepositor() < proposal2Deposits[j].GetDepositor() })

	input.GovKeeper.SetDeposit(ctx, deposit1)
	err := input.BankKeeper.SendCoinsFromAccountToModule(ctx, addrs[0], types.ModuleName, deposit1Amount)
	require.NoError(t, err)
	afterDepositBalance1 := input.BankKeeper.GetBalance(ctx, addrs[0], bondDenom)
	require.Equal(t, afterDepositBalance1.Amount, math.NewInt(9900))

	input.GovKeeper.SetDeposit(ctx, deposit2)
	err = input.BankKeeper.SendCoinsFromAccountToModule(ctx, addrs[1], types.ModuleName, deposit2Amount)
	require.NoError(t, err)
	afterDepositBalance2 := input.BankKeeper.GetBalance(ctx, addrs[1], bondDenom)
	require.Equal(t, afterDepositBalance2.Amount, math.NewInt(9950))

	input.GovKeeper.SetDeposit(ctx, deposit3)
	err = input.BankKeeper.SendCoinsFromAccountToModule(ctx, addrs[2], types.ModuleName, deposit3Amount)
	afterDepositBalance3 := input.BankKeeper.GetBalance(ctx, addrs[2], bondDenom)
	require.Equal(t, afterDepositBalance3.Amount, math.NewInt(9000))
	require.NoError(t, err)

	err = input.GovKeeper.DeleteAndBurnDeposits(ctx, 1)
	require.NoError(t, err)

	deposits1, err := input.GovKeeper.GetDeposits(ctx, 2)
	require.NoError(t, err)
	sort.Slice(deposits1, func(i, j int) bool { return deposits1[i].GetDepositor() < deposits1[j].GetDepositor() })
	require.Equal(t, deposits1, proposal1Deposits)

	err = input.GovKeeper.DeleteAndBurnDeposits(ctx, 2)
	require.NoError(t, err)
	emptyDeposits, err := input.GovKeeper.GetDeposits(ctx, 2)
	require.NoError(t, err)
	require.Equal(t, emptyDeposits, (v1.Deposits)(nil))
	afterBurnBalance1 := input.BankKeeper.GetBalance(ctx, addrs[0], bondDenom)
	require.Equal(t, afterBurnBalance1.Amount, math.NewInt(9900))
	afterBurnBalance2 := input.BankKeeper.GetBalance(ctx, addrs[1], bondDenom)
	require.Equal(t, afterBurnBalance2.Amount, math.NewInt(9950))

	deposits2, err := input.GovKeeper.GetDeposits(ctx, 5)
	require.NoError(t, err)
	sort.Slice(deposits2, func(i, j int) bool { return deposits2[i].GetDepositor() < deposits2[j].GetDepositor() })
	require.Equal(t, deposits2, proposal2Deposits)
}

func TestSimpleRefundDeposits(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	input.Faucet.Fund(ctx, addrs[0], sdk.NewCoin(bondDenom, math.NewInt(10000)))
	input.Faucet.Fund(ctx, addrs[1], sdk.NewCoin(bondDenom, math.NewInt(10000)))

	deposit1Amount := []sdk.Coin{sdk.NewCoin(bondDenom, math.NewInt(100))}
	deposit1 := v1.Deposit{ProposalId: 2, Depositor: addrs[0].String(), Amount: deposit1Amount}

	deposit2Amount := []sdk.Coin{sdk.NewCoin(bondDenom, math.NewInt(50))}
	deposit2 := v1.Deposit{ProposalId: 2, Depositor: addrs[1].String(), Amount: deposit2Amount}

	proposal1Deposits := make(v1.Deposits, 0, 2)
	proposal1Deposits = append(proposal1Deposits, &deposit1)
	proposal1Deposits = append(proposal1Deposits, &deposit2)
	sort.Slice(proposal1Deposits, func(i, j int) bool { return proposal1Deposits[i].GetDepositor() < proposal1Deposits[j].GetDepositor() })

	input.GovKeeper.SetDeposit(ctx, deposit1)
	err := input.BankKeeper.SendCoinsFromAccountToModule(ctx, addrs[0], types.ModuleName, deposit1Amount)
	require.NoError(t, err)
	afterDepositBalance1 := input.BankKeeper.GetBalance(ctx, addrs[0], bondDenom)
	require.Equal(t, afterDepositBalance1.Amount, math.NewInt(9900))

	input.GovKeeper.SetDeposit(ctx, deposit2)
	err = input.BankKeeper.SendCoinsFromAccountToModule(ctx, addrs[1], types.ModuleName, deposit2Amount)
	require.NoError(t, err)
	afterDepositBalance2 := input.BankKeeper.GetBalance(ctx, addrs[1], bondDenom)
	require.Equal(t, afterDepositBalance2.Amount, math.NewInt(9950))

	deposits1, err := input.GovKeeper.GetDeposits(ctx, 2)
	require.NoError(t, err)
	sort.Slice(deposits1, func(i, j int) bool { return deposits1[i].GetDepositor() < deposits1[j].GetDepositor() })
	require.Equal(t, deposits1, proposal1Deposits)

	err = input.GovKeeper.RefundAndDeleteDeposits(ctx, 2)
	require.NoError(t, err)
	emptyDeposits, err := input.GovKeeper.GetDeposits(ctx, 2)
	require.NoError(t, err)
	require.Equal(t, emptyDeposits, (v1.Deposits)(nil))
	afterRefundBalance1 := input.BankKeeper.GetBalance(ctx, addrs[0], bondDenom)
	require.Equal(t, afterRefundBalance1.Amount, math.NewInt(10000))
	afterRefundBalance2 := input.BankKeeper.GetBalance(ctx, addrs[1], bondDenom)
	require.Equal(t, afterRefundBalance2.Amount, math.NewInt(10000))
}

func TestSimpleAddDeposit(t *testing.T) {
	initAmount := math.NewInt(10000000)
	minDepositRatio := "0.01"
	bondDenomMinDeposit := math.NewInt(10000)

	testCases := []struct {
		name        string
		bondDenom   string
		proposalId  uint64
		deposit     math.Int
		expectError bool
	}{
		{
			name:        "good denom, deposits and proposal id",
			bondDenom:   "uinit",
			proposalId:  1,
			deposit:     math.NewInt(100),
			expectError: false,
		},
		{
			name:        "bad denom",
			bondDenom:   "foo",
			proposalId:  1,
			deposit:     math.NewInt(100),
			expectError: true,
		},
		{
			name:        "not existing proposal",
			bondDenom:   "uinit",
			proposalId:  2,
			deposit:     math.NewInt(100),
			expectError: true,
		},
		{
			name:        "too small deposit amount",
			bondDenom:   "uinit",
			proposalId:  1,
			deposit:     math.NewInt(99),
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, input := createDefaultTestInput(t)
			params, err := input.GovKeeper.Params.Get(ctx)
			require.NoError(t, err)

			params.MinDepositRatio = minDepositRatio
			params.MinDeposit = sdk.Coins{sdk.NewCoin(bondDenom, bondDenomMinDeposit)}
			err = input.GovKeeper.Params.Set(ctx, params)
			require.NoError(t, err)

			input.Faucet.Fund(ctx, addrs[0], sdk.NewCoin(tc.bondDenom, initAmount))

			proposal, err := input.GovKeeper.SubmitProposal(ctx, nil, "", "", "", addrs[0], false)
			require.NoError(t, err)
			require.Equal(t, proposal.Id, uint64(1))

			input.Faucet.Fund(ctx, addrs[0], sdk.NewCoin("foo", bondDenomMinDeposit.MulRaw(10)))
			_, err = input.GovKeeper.AddDeposit(ctx, tc.proposalId, addrs[0], []sdk.Coin{sdk.NewCoin(tc.bondDenom, tc.deposit)})
			if tc.expectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestAddDepositAfterVotePeriod(t *testing.T) {
	initAmount := math.NewInt(10000000)
	minDepositRatio := "0.01"
	bondDenomMinDeposit := math.NewInt(10000)

	ctx, input := createDefaultTestInput(t)
	params, err := input.GovKeeper.Params.Get(ctx)
	require.NoError(t, err)

	params.MinDepositRatio = minDepositRatio
	params.MinDeposit = sdk.Coins{sdk.NewCoin(bondDenom, bondDenomMinDeposit)}
	err = input.GovKeeper.Params.Set(ctx, params)
	require.NoError(t, err)

	input.Faucet.Fund(ctx, addrs[0], sdk.NewCoin(bondDenom, initAmount))

	proposal, err := input.GovKeeper.SubmitProposal(ctx, nil, "", "", "", addrs[0], false)
	require.NoError(t, err)
	require.Equal(t, proposal.Id, uint64(1))

	proposal.Status = v1.StatusPassed
	err = input.GovKeeper.SetProposal(ctx, proposal)
	require.NoError(t, err)
	_, err = input.GovKeeper.AddDeposit(ctx, proposal.Id, addrs[0], []sdk.Coin{sdk.NewCoin(bondDenom, bondDenomMinDeposit)})
	require.Error(t, err)
}

func TestEmergencyAfterActivateVotingPeriod(t *testing.T) {
	initAmount := math.NewInt(10000000)
	minDepositRatio := "0.01"
	bondDenomMinDeposit := math.NewInt(10000)
	emergencyMinDeposit := math.NewInt(100000)

	ctx, input := createDefaultTestInput(t)
	params, err := input.GovKeeper.Params.Get(ctx)
	require.NoError(t, err)

	params.MinDepositRatio = minDepositRatio
	params.MinDeposit = sdk.Coins{sdk.NewCoin(bondDenom, bondDenomMinDeposit)}
	params.EmergencyMinDeposit = sdk.Coins{sdk.NewCoin(bondDenom, emergencyMinDeposit)}

	err = input.GovKeeper.Params.Set(ctx, params)
	require.NoError(t, err)

	input.Faucet.Fund(ctx, addrs[0], sdk.NewCoin(bondDenom, initAmount))
	input.Faucet.Fund(ctx, addrs[1], sdk.NewCoin(bondDenom, initAmount))

	proposal, err := input.GovKeeper.SubmitProposal(ctx, nil, "", "", "", addrs[0], false)
	require.NoError(t, err)
	require.Equal(t, proposal.Id, uint64(1))

	isActivated, err := input.GovKeeper.AddDeposit(ctx, 1, addrs[0], []sdk.Coin{sdk.NewCoin(bondDenom, math.NewInt(100))})
	require.NoError(t, err)
	require.False(t, isActivated)

	isActivated, err = input.GovKeeper.AddDeposit(ctx, 1, addrs[1], []sdk.Coin{sdk.NewCoin(bondDenom, math.NewInt(9900))})
	require.NoError(t, err)
	require.True(t, isActivated)

	prop, err := input.GovKeeper.Proposals.Get(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, prop.Status, v1.StatusVotingPeriod)
	require.False(t, prop.Emergency)

	isActivated, err = input.GovKeeper.AddDeposit(ctx, 1, addrs[0], []sdk.Coin{sdk.NewCoin(bondDenom, math.NewInt(90000))})
	require.NoError(t, err)
	require.False(t, isActivated)

	prop, err = input.GovKeeper.Proposals.Get(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, prop.Status, v1.StatusVotingPeriod)
	require.True(t, prop.Emergency)
}

func TestActivateVotingPeriodAndEmergency(t *testing.T) {
	initAmount := math.NewInt(10000000)
	minDepositRatio := "0.01"
	bondDenomMinDeposit := math.NewInt(10000)
	emergencyMinDeposit := math.NewInt(100000)

	ctx, input := createDefaultTestInput(t)
	params, err := input.GovKeeper.Params.Get(ctx)
	require.NoError(t, err)

	params.MinDepositRatio = minDepositRatio
	params.MinDeposit = sdk.Coins{sdk.NewCoin(bondDenom, bondDenomMinDeposit)}
	params.EmergencyMinDeposit = sdk.Coins{sdk.NewCoin(bondDenom, emergencyMinDeposit)}

	err = input.GovKeeper.Params.Set(ctx, params)
	require.NoError(t, err)

	input.Faucet.Fund(ctx, addrs[0], sdk.NewCoin(bondDenom, initAmount))
	input.Faucet.Fund(ctx, addrs[1], sdk.NewCoin(bondDenom, initAmount))

	proposal, err := input.GovKeeper.SubmitProposal(ctx, nil, "", "", "", addrs[0], false)
	require.NoError(t, err)
	require.Equal(t, proposal.Id, uint64(1))

	isActivated, err := input.GovKeeper.AddDeposit(ctx, 1, addrs[1], []sdk.Coin{sdk.NewCoin(bondDenom, emergencyMinDeposit)})
	require.NoError(t, err)
	require.True(t, isActivated)

	prop, err := input.GovKeeper.Proposals.Get(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, prop.Status, v1.StatusVotingPeriod)
	require.True(t, prop.Emergency)
}

func TestChargeDeposit(t *testing.T) {
	initAmount := math.NewInt(10000000)
	minDepositRatio := "0.01"
	bondDenomMinDeposit := math.NewInt(10000)
	proposalCancelRatio := "0.5"

	testCases := []struct {
		name        string
		proposalId  uint64
		deposit1    math.Int
		deposit2    math.Int
		destAddress string
		expectError bool
	}{
		{
			name:        "good proposal id during deposit period and burning remaining deposits",
			proposalId:  1,
			deposit1:    math.NewInt(100),
			deposit2:    math.NewInt(200),
			destAddress: "",
			expectError: false,
		},
		{
			name:        "voting period and burning remaining deposits",
			proposalId:  1,
			deposit1:    math.NewInt(100),
			deposit2:    math.NewInt(9900),
			destAddress: "",
			expectError: false,
		},
		{
			name:        "specific destAddr",
			proposalId:  1,
			deposit1:    math.NewInt(100),
			deposit2:    math.NewInt(9900),
			destAddress: addrs[3].String(),
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, input := createDefaultTestInput(t)
			params, err := input.GovKeeper.Params.Get(ctx)
			require.NoError(t, err)

			params.MinDepositRatio = minDepositRatio
			params.MinDeposit = sdk.Coins{sdk.NewCoin(bondDenom, bondDenomMinDeposit)}
			params.ProposalCancelRatio = proposalCancelRatio
			params.ProposalCancelDest = tc.destAddress
			err = input.GovKeeper.Params.Set(ctx, params)
			require.NoError(t, err)

			input.Faucet.Fund(ctx, addrs[0], sdk.NewCoin(bondDenom, initAmount))
			input.Faucet.Fund(ctx, addrs[1], sdk.NewCoin(bondDenom, initAmount))

			proposal, err := input.GovKeeper.SubmitProposal(ctx, nil, "", "", "", addrs[0], false)
			require.NoError(t, err)
			require.Equal(t, proposal.Id, uint64(1))

			deposit1 := sdk.NewCoin(bondDenom, tc.deposit1)
			_, err = input.GovKeeper.AddDeposit(ctx, tc.proposalId, addrs[0], []sdk.Coin{deposit1})
			require.NoError(t, err)

			deposit2 := sdk.NewCoin(bondDenom, tc.deposit2)
			_, err = input.GovKeeper.AddDeposit(ctx, tc.proposalId, addrs[1], []sdk.Coin{deposit2})
			require.NoError(t, err)

			err = input.GovKeeper.ChargeDeposit(ctx, tc.proposalId, tc.destAddress, proposalCancelRatio)
			if tc.expectError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			deposits, err := input.GovKeeper.GetDeposits(ctx, 1)
			require.NoError(t, err)
			require.Equal(t, deposits, (v1.Deposits)(nil))

			afterChargeBalance1 := input.BankKeeper.GetBalance(ctx, addrs[0], bondDenom)
			afterChargeBalance2 := input.BankKeeper.GetBalance(ctx, addrs[1], bondDenom)

			rate := math.LegacyMustNewDecFromStr(proposalCancelRatio)

			chargedDeposit1 := tc.deposit1.ToLegacyDec().Mul(rate).TruncateInt()
			chargedDeposit2 := tc.deposit2.ToLegacyDec().Mul(rate).TruncateInt()
			require.Equal(t, afterChargeBalance1.Amount, initAmount.Sub(chargedDeposit1))
			require.Equal(t, afterChargeBalance2.Amount, initAmount.Sub(chargedDeposit2))
		})
	}
}
