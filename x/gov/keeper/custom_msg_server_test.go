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
