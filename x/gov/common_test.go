package gov_test

import (
	"encoding/json"
	"testing"
	"time"

	initiaapp "github.com/initia-labs/initia/app"
	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"
	"github.com/cometbft/cometbft/crypto"
	"github.com/cometbft/cometbft/crypto/secp256k1"
	cmtypes "github.com/cometbft/cometbft/types"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	v1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
)

// Bond denom should be set for staking test
const bondDenom = initiaapp.BondDenom

var (
	pubKeys = []crypto.PubKey{
		secp256k1.GenPrivKey().PubKey(),
		secp256k1.GenPrivKey().PubKey(),
		secp256k1.GenPrivKey().PubKey(),
		secp256k1.GenPrivKey().PubKey(),
		secp256k1.GenPrivKey().PubKey(),
	}

	addrs = []sdk.AccAddress{
		sdk.AccAddress(pubKeys[0].Address()),
		sdk.AccAddress(pubKeys[1].Address()),
		sdk.AccAddress(pubKeys[2].Address()),
		sdk.AccAddress(pubKeys[3].Address()),
		sdk.AccAddress(pubKeys[4].Address()),
	}

	validators = []*cmtypes.Validator{
		cmtypes.NewValidator(pubKeys[0], 1000000),
		cmtypes.NewValidator(pubKeys[1], 1000000),
		cmtypes.NewValidator(pubKeys[2], 1000000),
		cmtypes.NewValidator(pubKeys[3], 1000000),
		cmtypes.NewValidator(pubKeys[4], 1000000),
	}

	genCoins = sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(10000000))).Sort()

	minDepositRatio        = "0.01"
	minDeposit             = []sdk.Coin{sdk.NewCoin(bondDenom, math.NewInt(10000))}
	emergencyMinDeposit    = []sdk.Coin{sdk.NewCoin(bondDenom, math.NewInt(100000))}
	depositPeriod          = time.Hour
	votingPeriod           = time.Hour
	emergencyTallyInterval = time.Minute * 10
)

func createDefaultApp(t *testing.T) *initiaapp.InitiaApp {
	app := initiaapp.SetupWithGenesisAccounts(nil, authtypes.GenesisAccounts{
		&authtypes.BaseAccount{Address: addrs[0].String()},
	},
		banktypes.Balance{Address: addrs[0].String(), Coins: genCoins},
	)
	return app
}

func createAppWithSimpleValidators(t *testing.T) *initiaapp.InitiaApp {
	valSet := cmtypes.NewValidatorSet(validators)
	app := initiaapp.SetupWithGenesisAccounts(valSet, authtypes.GenesisAccounts{
		&authtypes.BaseAccount{Address: addrs[0].String()},
		&authtypes.BaseAccount{Address: addrs[1].String()},
		&authtypes.BaseAccount{Address: addrs[2].String()},
		&authtypes.BaseAccount{Address: addrs[3].String()},
		&authtypes.BaseAccount{Address: addrs[4].String()},
	},
		banktypes.Balance{Address: addrs[0].String(), Coins: genCoins},
		banktypes.Balance{Address: addrs[1].String(), Coins: genCoins},
		banktypes.Balance{Address: addrs[2].String(), Coins: genCoins},
		banktypes.Balance{Address: addrs[3].String(), Coins: genCoins},
		banktypes.Balance{Address: addrs[4].String(), Coins: genCoins},
	)
	_, err := app.FinalizeBlock(&abci.RequestFinalizeBlock{Height: app.LastBlockHeight() + 1})
	require.NoError(t, err)
	ctx := app.BaseApp.NewContext(false)
	params, err := app.GovKeeper.Params.Get(ctx)
	require.NoError(t, err)
	params.MinDepositRatio = minDepositRatio
	params.MinDeposit = minDeposit
	params.EmergencyMinDeposit = emergencyMinDeposit
	params.MaxDepositPeriod = depositPeriod
	params.VotingPeriod = votingPeriod
	params.EmergencyTallyInterval = emergencyTallyInterval
	err = app.GovKeeper.Params.Set(ctx, params)
	require.NoError(t, err)
	_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{Height: app.LastBlockHeight() + 1})
	require.NoError(t, err)
	return app
}

func createTextProposalMsg(t *testing.T, initialTokenAmount int64, expedited bool) *v1.MsgSubmitProposal {
	title := "Proposal"
	summary := "description"
	proposalMetadata := govtypes.ProposalMetadata{
		Title:   title,
		Summary: summary,
	}
	metadata, err := json.Marshal(proposalMetadata)
	require.NoError(t, err)

	newProposalMsg, err := v1.NewMsgSubmitProposal(
		[]sdk.Msg{},
		sdk.Coins{sdk.NewInt64Coin(bondDenom, initialTokenAmount)},
		addrs[0].String(),
		string(metadata),
		title,
		summary,
		expedited,
	)
	require.NoError(t, err)
	return newProposalMsg
}

func createDepositMsg(t *testing.T, depositor sdk.AccAddress, proposalID uint64, amount sdk.Coins) *v1.MsgDeposit {
	newDepositMsg := v1.NewMsgDeposit(
		depositor,
		proposalID,
		amount,
	)
	return newDepositMsg
}

func createVoteMsg(t *testing.T, voter sdk.AccAddress, proposalID uint64, option v1.VoteOption) *v1.MsgVote {
	newVoteMsg := v1.NewMsgVote(
		voter,
		proposalID,
		option,
		"",
	)
	return newVoteMsg
}
