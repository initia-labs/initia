package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	"github.com/initia-labs/initia/x/gov/keeper"
	"github.com/initia-labs/initia/x/gov/types"
)

func Test_CustomMsgServer_UpdateParams(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	params, err := input.GovKeeper.Params.Get(ctx)
	require.NoError(t, err)
	params.EmergencyTallyInterval = time.Hour

	ms := keeper.NewCustomMsgServerImpl(&input.GovKeeper)
	_, err = ms.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Params:    params,
	})
	require.NoError(t, err)
	_params, err := input.GovKeeper.Params.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, params, _params)

	// unauthorized
	_, err = ms.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: authtypes.NewModuleAddress("invalid").String(),
		Params:    params,
	})
	require.Error(t, err)
}

func Test_CustomMsgServer_EmergencyProposalSubmitters(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	ms := keeper.NewCustomMsgServerImpl(&input.GovKeeper)

	_, err := ms.AddEmergencySubmitters(ctx, &types.MsgAddEmergencySubmitters{
		Authority: "invalid authority",
		EmergencySubmitters: []string{
			addrs[0].String(),
		},
	})
	require.Error(t, err)

	_, err = ms.AddEmergencySubmitters(ctx, &types.MsgAddEmergencySubmitters{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		EmergencySubmitters: []string{
			"invalid address",
		},
	})
	require.Error(t, err)

	_, err = ms.AddEmergencySubmitters(ctx, &types.MsgAddEmergencySubmitters{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		EmergencySubmitters: []string{
			addrs[0].String(),
			addrs[1].String(),
			addrs[2].String(),
		},
	})
	require.NoError(t, err)

	submitters, err := input.GovKeeper.Params.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, []string{addrs[0].String(), addrs[1].String(), addrs[2].String()}, submitters.EmergencySubmitters)

	_, err = ms.AddEmergencySubmitters(ctx, &types.MsgAddEmergencySubmitters{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		EmergencySubmitters: []string{
			addrs[2].String(),
			addrs[3].String(),
			addrs[4].String(),
		},
	})
	require.NoError(t, err)

	submitters, err = input.GovKeeper.Params.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, []string{addrs[0].String(), addrs[1].String(), addrs[2].String(), addrs[3].String(), addrs[4].String()}, submitters.EmergencySubmitters)

	_, err = ms.RemoveEmergencySubmitters(ctx, &types.MsgRemoveEmergencySubmitters{
		Authority: "invalid authority",
		EmergencySubmitters: []string{
			addrs[2].String(),
		},
	})
	require.Error(t, err)

	_, err = ms.RemoveEmergencySubmitters(ctx, &types.MsgRemoveEmergencySubmitters{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		EmergencySubmitters: []string{
			addrs[2].String(),
		},
	})
	require.NoError(t, err)

	submitters, err = input.GovKeeper.Params.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, []string{addrs[0].String(), addrs[1].String(), addrs[3].String(), addrs[4].String()}, submitters.EmergencySubmitters)

	_, err = ms.RemoveEmergencySubmitters(ctx, &types.MsgRemoveEmergencySubmitters{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		EmergencySubmitters: []string{
			addrs[0].String(),
			addrs[1].String(),
			addrs[2].String(),
		},
	})
	require.Error(t, err)

	submitters, err = input.GovKeeper.Params.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, []string{addrs[0].String(), addrs[1].String(), addrs[3].String(), addrs[4].String()}, submitters.EmergencySubmitters)

	// remove all submitters
	_, err = ms.RemoveEmergencySubmitters(ctx, &types.MsgRemoveEmergencySubmitters{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		EmergencySubmitters: []string{
			addrs[0].String(),
			addrs[1].String(),
			addrs[3].String(),
			addrs[4].String(),
		},
	})
	require.NoError(t, err)

	submitters, err = input.GovKeeper.Params.Get(ctx)
	require.NoError(t, err)
	require.Len(t, submitters.EmergencySubmitters, 0)
}

func Test_CustomMsgServer_ActivateEmergencyProposal(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	ms := keeper.NewCustomMsgServerImpl(&input.GovKeeper)

	// create a proposal
	proposal, err := input.GovKeeper.SubmitProposal(ctx, nil, "", "", "", addrs[0], false)
	require.NoError(t, err)
	require.Equal(t, proposal.Id, uint64(1))
	require.False(t, proposal.Emergency)

	// activate emergency proposal without adding addrs[0] to emergency submitters
	_, err = ms.ActivateEmergencyProposal(ctx, &types.MsgActivateEmergencyProposal{
		ProposalId: 1,
		Sender:     addrs[0].String(),
	})
	require.Error(t, err)

	// add addrs[0] to emergency submitters
	_, err = ms.AddEmergencySubmitters(ctx, &types.MsgAddEmergencySubmitters{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		EmergencySubmitters: []string{
			addrs[0].String(),
		},
	})
	require.NoError(t, err)

	// activate emergency proposal without adding deposit
	_, err = ms.ActivateEmergencyProposal(ctx, &types.MsgActivateEmergencyProposal{
		ProposalId: 1,
		Sender:     addrs[0].String(),
	})
	require.Error(t, err)

	// get emergency min deposit
	params, err := input.GovKeeper.Params.Get(ctx)
	require.NoError(t, err)
	minDeposit := params.MinDeposit
	emergencyMinDeposit := sdk.Coins(params.EmergencyMinDeposit)

	input.Faucet.Fund(ctx, addrs[1], emergencyMinDeposit...)

	// add deposit to proposal to make it voting period
	_, err = input.GovKeeper.AddDeposit(ctx, 1, addrs[1], minDeposit)
	require.NoError(t, err)

	proposal, err = input.GovKeeper.Proposals.Get(ctx, 1)
	require.NoError(t, err)
	require.Equal(t, proposal.Status, v1.StatusVotingPeriod)
	require.False(t, proposal.Emergency)

	// activate emergency proposal with not enough deposit should fail
	_, err = ms.ActivateEmergencyProposal(ctx, &types.MsgActivateEmergencyProposal{
		ProposalId: 1,
		Sender:     addrs[0].String(),
	})
	require.Error(t, err)

	// add deposit to proposal to make it emergency proposal
	_, err = input.GovKeeper.AddDeposit(ctx, 1, addrs[1], emergencyMinDeposit.Sub(minDeposit...))
	require.NoError(t, err)

	// activate emergency proposal with unauthorized sender should fail
	_, err = ms.ActivateEmergencyProposal(ctx, &types.MsgActivateEmergencyProposal{
		ProposalId: 1,
		Sender:     addrs[1].String(),
	})
	require.Error(t, err)

	// activate emergency proposal
	_, err = ms.ActivateEmergencyProposal(ctx, &types.MsgActivateEmergencyProposal{
		ProposalId: 1,
		Sender:     addrs[0].String(),
	})
	require.NoError(t, err)

	proposal, err = input.GovKeeper.Proposals.Get(ctx, 1)
	require.NoError(t, err)
	require.True(t, proposal.Emergency)

	// activate emergency proposal again should fail
	_, err = ms.ActivateEmergencyProposal(ctx, &types.MsgActivateEmergencyProposal{
		ProposalId: 1,
		Sender:     addrs[0].String(),
	})
	require.Error(t, err)
}
