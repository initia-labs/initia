package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/initia-labs/initia/x/reward/types"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

func Test_LastReleaseTimestamp(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	now := time.Now().UTC()
	input.RewardKeeper.SetLastReleaseTimestamp(ctx, now)

	lastReleaseTimestamp, err := input.RewardKeeper.GetLastReleaseTimestamp(ctx)
	require.NoError(t, err)
	require.Equal(t, now, lastReleaseTimestamp)
}

func Test_LastDilutionTimestamp(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	now := time.Now().UTC()
	input.RewardKeeper.SetLastDilutionTimestamp(ctx, now)
	lastDilutionTimestamp, err := input.RewardKeeper.GetLastDilutionTimestamp(ctx)
	require.NoError(t, err)
	require.Equal(t, now, lastDilutionTimestamp)
}

func Test_Params(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	params, err := input.RewardKeeper.GetParams(ctx)
	require.NoError(t, err)
	params.ReleaseRate = math.LegacyNewDecWithPrec(3, 2)
	err = input.RewardKeeper.SetParams(ctx, params)
	require.NoError(t, err)
	_params, err := input.RewardKeeper.GetParams(ctx)
	require.NoError(t, err)
	require.Equal(t, params, _params)
}

func Test_AnnualProvisions(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	params, err := input.RewardKeeper.GetParams(ctx)
	require.NoError(t, err)
	expectedAnnualProvisions := params.ReleaseRate.MulInt(input.BankKeeper.GetSupply(ctx, params.RewardDenom).Amount)
	annualProvisions, err := input.RewardKeeper.GetAnnualProvisions(ctx)
	require.NoError(t, err)
	require.Equal(t, expectedAnnualProvisions, annualProvisions)
}

func Test_GetRemainRewardAmount(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	params, err := input.RewardKeeper.GetParams(ctx)
	require.NoError(t, err)
	rewardCoin := sdk.NewCoin(params.RewardDenom, math.NewInt(100))

	rewardModuleAddr := input.AccountKeeper.GetModuleAddress(types.ModuleName)
	input.Faucet.Fund(ctx, rewardModuleAddr, rewardCoin)

	amount := input.RewardKeeper.GetRemainRewardAmount(ctx, params.RewardDenom)
	require.Equal(t, rewardCoin.Amount, amount)
}

func Test_AddCollectedFees(t *testing.T) {
	ctx, input := createDefaultTestInput(t)

	params, err := input.RewardKeeper.GetParams(ctx)
	require.NoError(t, err)
	rewardCoin := sdk.NewCoin(params.RewardDenom, math.NewInt(100))

	rewardModuleAddr := input.AccountKeeper.GetModuleAddress(types.ModuleName)
	input.Faucet.Fund(ctx, rewardModuleAddr, rewardCoin)

	err = input.RewardKeeper.AddCollectedFees(ctx, sdk.NewCoins(rewardCoin))
	require.NoError(t, err)

	balance := input.BankKeeper.GetBalance(ctx, authtypes.NewModuleAddress(authtypes.FeeCollectorName), params.RewardDenom)
	require.Equal(t, rewardCoin, balance)
}
