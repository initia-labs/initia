package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/initia-labs/initia/x/mstaking/keeper"
	"github.com/initia-labs/initia/x/mstaking/types"
)

func Test_UpdateParams(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	params, err := input.StakingKeeper.Params.Get(ctx)
	require.NoError(t, err)

	params.MaxValidators = 10
	ms := keeper.NewMsgServerImpl(input.StakingKeeper)
	_, err = ms.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: input.StakingKeeper.GetAuthority(),
		Params:    params,
	})
	require.NoError(t, err)

	paramsAfter, err := input.StakingKeeper.Params.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, params, paramsAfter)
}
