package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/initia-labs/initia/x/reward/keeper"
	"github.com/initia-labs/initia/x/reward/types"
	"github.com/stretchr/testify/require"
)

func Test_UpdateParams(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	params, err := input.RewardKeeper.GetParams(ctx)
	require.NoError(t, err)
	params.ReleaseRate = math.LegacyNewDecWithPrec(3, 2)
	msgServer := keeper.NewMsgServerImpl(&input.RewardKeeper)
	_, err = msgServer.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: input.RewardKeeper.GetAuthority(),
		Params:    params,
	})
	require.NoError(t, err)
	_params, err := input.RewardKeeper.GetParams(ctx)
	require.NoError(t, err)
	require.Equal(t, params, _params)
}

func Test_FundCommunityPool(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	moduleAddr := input.AccountKeeper.GetModuleAddress(types.ModuleName)
	input.Faucet.Fund(ctx, moduleAddr, sdk.NewCoin("ureward", math.NewInt(1000000)))

	msgServer := keeper.NewMsgServerImpl(&input.RewardKeeper)
	_, err := msgServer.FundCommunityPool(ctx, &types.MsgFundCommunityPool{
		Authority: input.RewardKeeper.GetAuthority(),
		Amount:    sdk.NewCoins(sdk.NewCoin("ureward", math.NewInt(500000))),
	})
	require.NoError(t, err)

	balance := input.BankKeeper.GetBalance(ctx, moduleAddr, "ureward")
	require.Equal(t, math.NewInt(500000), balance.Amount)

	feePool, err := input.DistKeeper.FeePool.Get(ctx)
	require.NoError(t, err)
	communityPool := feePool.GetCommunityPool()
	require.Equal(t, math.NewInt(500000), communityPool.AmountOf("ureward").TruncateInt())
}
