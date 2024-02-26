package keeper_test

import (
	"testing"

	"github.com/initia-labs/initia/x/ibc-hooks/keeper"
	"github.com/initia-labs/initia/x/ibc-hooks/types"
	"github.com/stretchr/testify/require"
)

func Test_QueryACL(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	err := input.IBCHooksKeeper.SetAllowed(ctx, addrs[0], true)
	require.NoError(t, err)

	qs := keeper.NewQueryServerImpl(&input.IBCHooksKeeper)
	res, err := qs.ACL(ctx, &types.QueryACLRequest{
		Address: addrs[0].String(),
	})
	require.NoError(t, err)
	require.True(t, res.Acl.Allowed)

	err = input.IBCHooksKeeper.SetAllowed(ctx, addrs[0], false)
	require.NoError(t, err)

	res, err = qs.ACL(ctx, &types.QueryACLRequest{
		Address: addrs[0].String(),
	})
	require.NoError(t, err)
	require.False(t, res.Acl.Allowed)
}

func Test_QueryACLs(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	err := input.IBCHooksKeeper.SetAllowed(ctx, addrs[0], true)
	require.NoError(t, err)

	err = input.IBCHooksKeeper.SetAllowed(ctx, addrs[1], false)
	require.NoError(t, err)

	qs := keeper.NewQueryServerImpl(&input.IBCHooksKeeper)
	res, err := qs.ACLs(ctx, &types.QueryACLsRequest{})
	require.NoError(t, err)
	if res.Acls[0].Address == addrs[0].String() {
		require.True(t, res.Acls[0].Allowed)
		require.False(t, res.Acls[1].Allowed)
	} else {
		require.True(t, res.Acls[1].Allowed)
		require.False(t, res.Acls[0].Allowed)
	}

}

func Test_QueryParams(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	err := input.IBCHooksKeeper.Params.Set(ctx, types.Params{DefaultAllowed: true})
	require.NoError(t, err)

	qs := keeper.NewQueryServerImpl(&input.IBCHooksKeeper)
	res, err := qs.Params(ctx, &types.QueryParamsRequest{})
	require.NoError(t, err)
	require.True(t, res.Params.DefaultAllowed)
}
