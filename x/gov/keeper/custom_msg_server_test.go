package keeper_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"

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

	_, err := ms.AddEmergencyProposalSubmitters(ctx, &types.MsgAddEmergencyProposalSubmitters{
		Authority: "invalid authority",
		EmergencySubmitters: []string{
			addrs[0].String(),
		},
	})
	require.Error(t, err)

	_, err = ms.AddEmergencyProposalSubmitters(ctx, &types.MsgAddEmergencyProposalSubmitters{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		EmergencySubmitters: []string{
			"invalid address",
		},
	})
	require.Error(t, err)

	_, err = ms.AddEmergencyProposalSubmitters(ctx, &types.MsgAddEmergencyProposalSubmitters{
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

	_, err = ms.AddEmergencyProposalSubmitters(ctx, &types.MsgAddEmergencyProposalSubmitters{
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

	_, err = ms.RemoveEmergencyProposalSubmitters(ctx, &types.MsgRemoveEmergencyProposalSubmitters{
		Authority: "invalid authority",
		EmergencySubmitters: []string{
			addrs[2].String(),
		},
	})
	require.Error(t, err)

	_, err = ms.RemoveEmergencyProposalSubmitters(ctx, &types.MsgRemoveEmergencyProposalSubmitters{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		EmergencySubmitters: []string{
			addrs[2].String(),
		},
	})
	require.NoError(t, err)

	submitters, err = input.GovKeeper.Params.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, []string{addrs[0].String(), addrs[1].String(), addrs[3].String(), addrs[4].String()}, submitters.EmergencySubmitters)

	_, err = ms.RemoveEmergencyProposalSubmitters(ctx, &types.MsgRemoveEmergencyProposalSubmitters{
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
	_, err = ms.RemoveEmergencyProposalSubmitters(ctx, &types.MsgRemoveEmergencyProposalSubmitters{
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
