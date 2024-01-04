package distribution_test

import (
	"testing"

	"cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

func Test_BeginBlocker(t *testing.T) {
	app := createApp(t)

	// create rewards
	_, err := app.FinalizeBlock(&abci.RequestFinalizeBlock{Height: app.LastBlockHeight() + 1, ProposerAddress: valKey.PubKey().Address()})
	require.NoError(t, err)

	// initialize staking for bondDenom
	header := tmproto.Header{Height: app.LastBlockHeight() + 1, ProposerAddress: valKey.PubKey().Address()}

	ctx := app.BaseApp.NewContextLegacy(false, header)
	err = app.BankKeeper.MintCoins(ctx, authtypes.Minter, genCoins)
	require.NoError(t, err)
	err = app.BankKeeper.SendCoinsFromModuleToModule(ctx, authtypes.Minter, authtypes.FeeCollectorName, genCoins)
	require.NoError(t, err)

	coins := app.BankKeeper.GetAllBalances(ctx, authtypes.NewModuleAddress(authtypes.FeeCollectorName))
	require.NotEmpty(t, coins)

	_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height:          app.LastBlockHeight() + 1,
		ProposerAddress: valKey.PubKey().Address(),
		DecidedLastCommit: abci.CommitInfo{
			Votes: []abci.VoteInfo{
				{
					Validator: abci.Validator{
						Address: sdk.GetConsAddress(valKey.PubKey()),
						Power:   10,
					},
					BlockIdFlag: tmproto.BlockIDFlagCommit,
				},
			},
		},
	})
	require.NoError(t, err)

	header = tmproto.Header{Height: app.LastBlockHeight() + 1}
	ctx = app.BaseApp.NewContextLegacy(false, header)
	rewards, err := app.DistrKeeper.GetValidatorOutstandingRewards(ctx, sdk.ValAddress(addr1))
	require.NoError(t, err)

	// exclude community tax
	params, err := app.DistrKeeper.Params.Get(ctx)
	require.NoError(t, err)

	expectedRewards := sdk.NewDecCoinsFromCoins(genCoins...).MulDec(math.LegacyOneDec().Sub(params.CommunityTax))
	require.Equal(t, expectedRewards, rewards.Rewards.Sum())
}
