package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_Genesis(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	require.NoError(t, input.IBCHooksKeeper.SetAllowed(ctx, addrs[0], true))
	require.NoError(t, input.IBCHooksKeeper.SetAllowed(ctx, addrs[1], true))
	require.NoError(t, input.IBCHooksKeeper.SetAllowed(ctx, addrs[2], false))

	genState := input.IBCHooksKeeper.ExportGenesis(ctx)
	input.IBCHooksKeeper.InitGenesis(ctx, genState)

	allowed, err := input.IBCHooksKeeper.GetAllowed(ctx, addrs[0])
	require.NoError(t, err)
	require.True(t, allowed)

	allowed, err = input.IBCHooksKeeper.GetAllowed(ctx, addrs[1])
	require.NoError(t, err)
	require.True(t, allowed)

	allowed, err = input.IBCHooksKeeper.GetAllowed(ctx, addrs[2])
	require.NoError(t, err)
	require.False(t, allowed)
}
