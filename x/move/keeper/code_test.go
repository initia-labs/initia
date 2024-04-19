package keeper_test

import (
	"testing"

	"github.com/initia-labs/initia/x/move/keeper"
	vmtypes "github.com/initia-labs/movevm/types"

	"github.com/stretchr/testify/require"
)

func Test_CodeKeeper_GetParams(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	allowedPublishers, err := keeper.NewCodeKeeper(&input.MoveKeeper).GetParams(ctx)
	require.NoError(t, err)
	require.Empty(t, allowedPublishers)
}

func Test_CodeKeeper_SetParams(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	codeKeeper := keeper.NewCodeKeeper(&input.MoveKeeper)

	err := codeKeeper.SetAllowedPublishers(ctx, []vmtypes.AccountAddress{vmtypes.StdAddress, vmtypes.TestAddress})
	require.NoError(t, err)

	allowedPublishers, err := codeKeeper.GetAllowedPublishers(ctx)
	require.NoError(t, err)
	require.Equal(t, []vmtypes.AccountAddress{vmtypes.StdAddress, vmtypes.TestAddress}, allowedPublishers)
}

func Test_CodeKeeper_MustContains_StdAddr(t *testing.T) {
	ctx, input := createDefaultTestInput(t)
	codeKeeper := keeper.NewCodeKeeper(&input.MoveKeeper)

	err := codeKeeper.SetAllowedPublishers(ctx, []vmtypes.AccountAddress{vmtypes.TestAddress})
	require.Error(t, err)
}
