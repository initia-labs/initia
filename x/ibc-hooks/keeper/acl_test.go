package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_ACL(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	params, err := input.IBCHooksKeeper.Params.Get(ctx)
	require.NoError(t, err)

	allowed, err := input.IBCHooksKeeper.GetAllowed(ctx, addrs[0])
	require.NoError(t, err)
	require.Equal(t, params.DefaultAllowed, allowed)

	err = input.IBCHooksKeeper.SetAllowed(ctx, addrs[0], true)
	require.NoError(t, err)

	allowed, err = input.IBCHooksKeeper.GetAllowed(ctx, addrs[0])
	require.NoError(t, err)
	require.True(t, allowed)

	err = input.IBCHooksKeeper.SetAllowed(ctx, addrs[0], false)
	require.NoError(t, err)
}
