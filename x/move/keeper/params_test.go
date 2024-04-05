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

func Test_SetParams(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	allowedPublishers, err := input.MoveKeeper.AllowedPublishers(ctx)
	require.NoError(t, err)
	require.Empty(t, allowedPublishers)

	params, err := input.MoveKeeper.GetParams(ctx)
	require.NoError(t, err)
	params.AllowedPublishers = []string{vmtypes.StdAddress.String(), vmtypes.TestAddress.String()}

	err = input.MoveKeeper.SetParams(ctx, params)
	require.NoError(t, err)

	allowedPublishers, err = input.MoveKeeper.AllowedPublishers(ctx)
	require.NoError(t, err)
	require.Equal(t, []vmtypes.AccountAddress{vmtypes.StdAddress, vmtypes.TestAddress}, allowedPublishers)
}
