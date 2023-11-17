package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"

	"github.com/initia-labs/initia/x/gov/keeper"
	"github.com/initia-labs/initia/x/gov/types"
)

func Test_CustomMsgServer_UpdateParams(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	params := input.GovKeeper.GetParams(ctx)
	params.EmergencyTallyInterval = time.Hour

	ms := keeper.NewCustomMsgServerImpl(&input.GovKeeper)
	_, err := ms.UpdateParams(sdk.WrapSDKContext(ctx), &types.MsgUpdateParams{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Params:    params,
	})
	require.NoError(t, err)
	require.Equal(t, params, input.GovKeeper.GetParams(ctx))

	// unauthorized
	_, err = ms.UpdateParams(sdk.WrapSDKContext(ctx), &types.MsgUpdateParams{
		Authority: authtypes.NewModuleAddress("invalid").String(),
		Params:    params,
	})
	require.Error(t, err)
}
