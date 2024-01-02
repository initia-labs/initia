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

	// initialize staking for bondDenom
	header := tmproto.Header{Height: app.LastBlockHeight() + 1, ProposerAddress: valKey.PubKey().Address()}

	// create rewards
	app.BeginBlock(abci.RequestBeginBlock{Header: header})
	ctx := app.BaseApp.NewContext(false, header)
	err := app.BankKeeper.MintCoins(ctx, authtypes.Minter, genCoins)
	require.NoError(t, err)
	err = app.BankKeeper.SendCoinsFromModuleToModule(ctx, authtypes.Minter, authtypes.FeeCollectorName, genCoins)
	require.NoError(t, err)
	app.EndBlock(abci.RequestEndBlock{})
	app.Commit()

	header = tmproto.Header{Height: app.LastBlockHeight() + 1, ProposerAddress: valKey.PubKey().Address()}
	ctx = app.BaseApp.NewContext(true, header)
	coins := app.BankKeeper.GetAllBalances(ctx, authtypes.NewModuleAddress(authtypes.FeeCollectorName))
	require.NotEmpty(t, coins)

	app.BeginBlock(abci.RequestBeginBlock{Header: header, LastCommitInfo: abci.CommitInfo{
		Votes: []abci.VoteInfo{
			{
				Validator: abci.Validator{
					Address: sdk.GetConsAddress(valKey.PubKey()),
					Power:   10,
				},
				SignedLastBlock: true,
			},
		},
	}})
	app.EndBlock(abci.RequestEndBlock{})
	app.Commit()

	header = tmproto.Header{Height: app.LastBlockHeight() + 1}
	ctx = app.BaseApp.NewContext(true, header)
	rewards := app.DistrKeeper.GetValidatorOutstandingRewards(ctx, sdk.ValAddress(addr1))

	// exclude community tax
	expectedRewards := sdk.NewDecCoinsFromCoins(genCoins...).MulDec(math.LegacyOneDec().Sub(app.DistrKeeper.GetCommunityTax(ctx)))
	require.Equal(t, expectedRewards, rewards.Rewards.Sum())
}
