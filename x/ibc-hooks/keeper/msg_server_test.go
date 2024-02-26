package keeper_test

import (
	"testing"

	"github.com/initia-labs/initia/x/ibc-hooks/keeper"
	"github.com/initia-labs/initia/x/ibc-hooks/types"
	"github.com/stretchr/testify/require"
)

func Test_UpdateACL(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	ms := keeper.NewMsgServerImpl(&input.IBCHooksKeeper)
	_, err := ms.UpdateACL(ctx, &types.MsgUpdateACL{
		Authority: input.IBCHooksKeeper.GetAuthority(),
		Address:   addrs[0].String(),
		Allowed:   true,
	})
	require.NoError(t, err)

	allowed, err := input.IBCHooksKeeper.GetAllowed(ctx, addrs[0])
	require.NoError(t, err)
	require.True(t, allowed)
}

func Test_UpdateParams(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	allowed, err := input.IBCHooksKeeper.GetAllowed(ctx, addrs[0])
	require.NoError(t, err)
	require.False(t, allowed)

	ms := keeper.NewMsgServerImpl(&input.IBCHooksKeeper)
	_, err = ms.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: input.IBCHooksKeeper.GetAuthority(),
		Params: types.Params{
			DefaultAllowed: true,
		},
	})
	require.NoError(t, err)

	allowed, err = input.IBCHooksKeeper.GetAllowed(ctx, addrs[0])
	require.NoError(t, err)
	require.True(t, allowed)
}
