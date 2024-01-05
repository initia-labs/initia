package reward_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/initia-labs/initia/x/reward/types"
)

func Test_BeginBlocker(t *testing.T) {
	app := createApp(t)

	// new block
	_, err := app.FinalizeBlock(&abci.RequestFinalizeBlock{Height: app.LastBlockHeight() + 1})
	require.NoError(t, err)

	ctx := app.BaseApp.NewContext(false)

	// update params & mint coins for reward distribution
	params, err := app.RewardKeeper.GetParams(ctx)
	require.NoError(t, err)

	rewardDenom := params.RewardDenom
	rewardAmount := math.NewInt(10_000_000)
	rewardCoins := sdk.NewCoins(sdk.NewCoin(rewardDenom, rewardAmount))
	err = app.BankKeeper.MintCoins(ctx, authtypes.Minter, rewardCoins)
	require.NoError(t, err)
	err = app.BankKeeper.SendCoinsFromModuleToModule(ctx, authtypes.Minter, types.ModuleName, rewardCoins)
	require.NoError(t, err)

	supply := app.BankKeeper.GetSupply(ctx, rewardDenom)

	params.ReleaseEnabled = true
	params.ReleaseRate = math.LegacyNewDecWithPrec(7, 2) // 7%
	params.DilutionPeriod = time.Hour * 24
	app.RewardKeeper.SetParams(ctx, params)

	lastReleaseTimestamp, err := app.RewardKeeper.GetLastReleaseTimestamp(ctx)
	require.NoError(t, err)

	// new block after
	_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{Height: app.LastBlockHeight() + 1, Time: lastReleaseTimestamp})
	require.NoError(t, err)

	// new block after 24 hours
	_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{Height: app.LastBlockHeight() + 1, Time: lastReleaseTimestamp.Add(time.Hour * 24).Add(time.Second)})
	require.NoError(t, err)

	// check supply
	expectedReleasedAmount := math.LegacyNewDec(7).QuoInt64(100).MulInt(supply.Amount).QuoInt64(365).TruncateInt()
	checkBalance(t, app, authtypes.NewModuleAddress(types.ModuleName), rewardCoins.Sub(sdk.NewCoin(rewardDenom, expectedReleasedAmount)))

	// release rate should be half
	releaseRate, err := app.RewardKeeper.GetReleaseRate(ctx)
	require.NoError(t, err)
	require.Equal(t, math.LegacyNewDecWithPrec(35, 3), releaseRate)

	lastReleaseTimestamp2, err := app.RewardKeeper.GetLastReleaseTimestamp(ctx)
	require.NoError(t, err)
	require.Equal(t, lastReleaseTimestamp.Add(time.Hour*24).Add(time.Second), lastReleaseTimestamp2)
	lastDilutionTimestamp, err := app.RewardKeeper.GetLastDilutionTimestamp(ctx)
	require.NoError(t, err)
	require.Equal(t, lastReleaseTimestamp.Add(time.Hour*24).Add(time.Second), lastDilutionTimestamp)
}

func Test_BeginBlockerNotEnabled(t *testing.T) {
	app := createApp(t)

	// new block
	_, err := app.FinalizeBlock(&abci.RequestFinalizeBlock{Height: app.LastBlockHeight() + 1})
	require.NoError(t, err)

	ctx := app.BaseApp.NewContext(false)

	// update params & mint coins for reward distribution
	params, err := app.RewardKeeper.GetParams(ctx)
	require.NoError(t, err)

	rewardDenom := params.RewardDenom
	rewardAmount := math.NewInt(10_000_000)
	rewardCoins := sdk.NewCoins(sdk.NewCoin(rewardDenom, rewardAmount))
	err = app.BankKeeper.MintCoins(ctx, authtypes.Minter, rewardCoins)
	require.NoError(t, err)
	err = app.BankKeeper.SendCoinsFromModuleToModule(ctx, authtypes.Minter, types.ModuleName, rewardCoins)
	require.NoError(t, err)

	params.ReleaseEnabled = false
	params.ReleaseRate = math.LegacyNewDecWithPrec(7, 2) // 7%
	params.DilutionPeriod = time.Hour * 24
	app.RewardKeeper.SetParams(ctx, params)

	lastReleaseTimestamp, err := app.RewardKeeper.GetLastReleaseTimestamp(ctx)
	require.NoError(t, err)

	// new block after
	_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{Height: app.LastBlockHeight() + 1, Time: lastReleaseTimestamp})
	require.NoError(t, err)

	// new block after 24 hours
	_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{Height: app.LastBlockHeight() + 1, Time: lastReleaseTimestamp.Add(time.Hour * 24).Add(time.Second)})
	require.NoError(t, err)

	// check supply
	expectedReleasedAmount := math.ZeroInt()
	checkBalance(t, app, authtypes.NewModuleAddress(types.ModuleName), rewardCoins.Sub(sdk.NewCoin(rewardDenom, expectedReleasedAmount)))

	// only timestamps updated
	lastReleaseTimestamp2, err := app.RewardKeeper.GetLastReleaseTimestamp(ctx)
	require.NoError(t, err)
	require.Equal(t, lastReleaseTimestamp.Add(time.Hour*24).Add(time.Second), lastReleaseTimestamp2)
	lastDilutionTimestamp, err := app.RewardKeeper.GetLastDilutionTimestamp(ctx)
	require.NoError(t, err)
	require.Equal(t, lastReleaseTimestamp.Add(time.Hour*24).Add(time.Second), lastDilutionTimestamp)
}
