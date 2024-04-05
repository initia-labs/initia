package keeper_test

import (
	"testing"

	vmtypes "github.com/initia-labs/movevm/types"
	"github.com/stretchr/testify/require"
)

func Test_SetAllowedPublishers(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	allowedPublishers, err := input.MoveKeeper.AllowedPublishers(ctx)
	require.NoError(t, err)
	require.Empty(t, allowedPublishers)

	err = input.MoveKeeper.SetAllowedPublishers(ctx, []vmtypes.AccountAddress{vmtypes.StdAddress, vmtypes.TestAddress})
	require.NoError(t, err)

	allowedPublishers, err = input.MoveKeeper.AllowedPublishers(ctx)
	require.NoError(t, err)
	require.Equal(t, []vmtypes.AccountAddress{vmtypes.StdAddress, vmtypes.TestAddress}, allowedPublishers)
}
