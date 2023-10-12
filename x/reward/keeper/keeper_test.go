package keeper_test

import (
	"testing"
	"time"

	"github.com/initia-labs/initia/x/reward/types"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

func Test_LastReleaseTimestamp(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	now := time.Now().UTC()
	input.RewardKeeper.SetLastReleaseTimestamp(ctx, now)
	require.Equal(t, now, input.RewardKeeper.GetLastReleaseTimestamp(ctx))
}

func Test_LastDilutionTimestamp(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	now := time.Now().UTC()
	input.RewardKeeper.SetLastDilutionTimestamp(ctx, now)
	require.Equal(t, now, input.RewardKeeper.GetLastDilutionTimestamp(ctx))
}

func Test_Params(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	params := input.RewardKeeper.GetParams(ctx)
	params.ReleaseRate = sdk.NewDecWithPrec(3, 2)
	input.RewardKeeper.SetParams(ctx, params)
	require.Equal(t, params, input.RewardKeeper.GetParams(ctx))
}

func Test_AnnualProvisions(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	params := input.RewardKeeper.GetParams(ctx)
	expectedAnnualProvisions := params.ReleaseRate.MulInt(input.BankKeeper.GetSupply(ctx, params.RewardDenom).Amount)
	require.Equal(t, expectedAnnualProvisions, input.RewardKeeper.GetAnnualProvisions(ctx))
}

func Test_GetRemainRewardAmount(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	rewardDenom := input.RewardKeeper.GetParams(ctx).RewardDenom
	rewardCoin := sdk.NewCoin(rewardDenom, sdk.NewInt(100))

	rewardModuleAddr := input.AccountKeeper.GetModuleAddress(types.ModuleName)
	input.Faucet.Fund(ctx, rewardModuleAddr, rewardCoin)

	amount := input.RewardKeeper.GetRemainRewardAmount(ctx, rewardDenom)
	require.Equal(t, rewardCoin.Amount, amount)
}

func Test_AddCollectedFees(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	rewardDenom := input.RewardKeeper.GetParams(ctx).RewardDenom
	rewardCoin := sdk.NewCoin(rewardDenom, sdk.NewInt(100))

	rewardModuleAddr := input.AccountKeeper.GetModuleAddress(types.ModuleName)
	input.Faucet.Fund(ctx, rewardModuleAddr, rewardCoin)

	err := input.RewardKeeper.AddCollectedFees(ctx, sdk.NewCoins(rewardCoin))
	require.NoError(t, err)

	balance := input.BankKeeper.GetBalance(ctx, authtypes.NewModuleAddress(authtypes.FeeCollectorName), rewardDenom)
	require.Equal(t, rewardCoin, balance)
}
