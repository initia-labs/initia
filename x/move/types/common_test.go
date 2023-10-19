package types_test

import (
	"testing"

	initiaapp "github.com/initia-labs/initia/app"
	customdistrtypes "github.com/initia-labs/initia/x/distribution/types"
	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
)

func init() {
	sdkConfig := sdk.GetConfig()
	sdkConfig.SetCoinType(initiaapp.CoinType)

	accountPubKeyPrefix := initiaapp.AccountAddressPrefix + "pub"
	validatorAddressPrefix := initiaapp.AccountAddressPrefix + "valoper"
	validatorPubKeyPrefix := initiaapp.AccountAddressPrefix + "valoperpub"
	consNodeAddressPrefix := initiaapp.AccountAddressPrefix + "valcons"
	consNodePubKeyPrefix := initiaapp.AccountAddressPrefix + "valconspub"

	sdkConfig.SetBech32PrefixForAccount(initiaapp.AccountAddressPrefix, accountPubKeyPrefix)
	sdkConfig.SetBech32PrefixForValidator(validatorAddressPrefix, validatorPubKeyPrefix)
	sdkConfig.SetBech32PrefixForConsensusNode(consNodeAddressPrefix, consNodePubKeyPrefix)
	sdkConfig.SetAddressVerifier(initiaapp.VerifyAddressLen())
	sdkConfig.Seal()
}

// Bond denom should be set for staking test
const bondDenom = initiaapp.BondDenom

var (
	priv1 = secp256k1.GenPrivKey()
	addr1 = sdk.AccAddress(priv1.PubKey().Address())
	priv2 = secp256k1.GenPrivKey()
	addr2 = sdk.AccAddress(priv2.PubKey().Address())
	priv3 = secp256k1.GenPrivKey()
	addr3 = sdk.AccAddress(priv3.PubKey().Address())

	valKey = ed25519.GenPrivKey()

	commissionRates = stakingtypes.NewCommissionRates(math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec())

	genCoins = sdk.NewCoins(sdk.NewCoin(bondDenom, sdk.NewInt(5000000))).Sort()
	bondCoin = sdk.NewCoin(bondDenom, sdk.NewInt(1000000))
)

func checkBalance(t *testing.T, app *initiaapp.InitiaApp, addr sdk.AccAddress, balances sdk.Coins) {
	ctxCheck := app.BaseApp.NewContext(true, tmproto.Header{})
	require.True(t, balances.IsEqual(app.BankKeeper.GetAllBalances(ctxCheck, addr)))
}

func createApp(t *testing.T) *initiaapp.InitiaApp {
	app := initiaapp.SetupWithGenesisAccounts(nil, authtypes.GenesisAccounts{
		&authtypes.BaseAccount{Address: addr1.String()},
		&authtypes.BaseAccount{Address: addr2.String()},
	},
		banktypes.Balance{Address: addr1.String(), Coins: genCoins},
		banktypes.Balance{Address: addr2.String(), Coins: genCoins},
	)

	checkBalance(t, app, addr1, genCoins)
	checkBalance(t, app, addr2, genCoins)

	// set reward weight
	distrParams := customdistrtypes.DefaultParams()
	distrParams.RewardWeights = []customdistrtypes.RewardWeight{
		{Denom: bondDenom, Weight: sdk.OneDec()},
	}
	app.DistrKeeper.SetParams(app.BaseApp.NewContext(false, tmproto.Header{}), distrParams)

	// create validator
	description := stakingtypes.NewDescription("foo_moniker", "", "", "", "")
	createValidatorMsg, err := stakingtypes.NewMsgCreateValidator(
		sdk.ValAddress(addr1), valKey.PubKey(), sdk.NewCoins(bondCoin), description, commissionRates,
	)
	require.NoError(t, err)

	err = executeMsgs(t, app, []sdk.Msg{createValidatorMsg}, []uint64{0}, []uint64{0}, priv1)
	require.NoError(t, err)

	checkBalance(t, app, addr1, genCoins.Sub(bondCoin))

	header := tmproto.Header{Height: app.LastBlockHeight() + 1}
	app.BeginBlock(abci.RequestBeginBlock{Header: header})

	return app
}

func executeMsgs(t *testing.T, app *initiaapp.InitiaApp, msgs []sdk.Msg, accountNum []uint64, sequenceNum []uint64, priv ...cryptotypes.PrivKey) error {
	txGen := initiaapp.MakeEncodingConfig().TxConfig
	header := tmproto.Header{Height: app.LastBlockHeight() + 1}
	_, _, err := simtestutil.SignCheckDeliver(t, txGen, app.BaseApp, header, msgs, "", accountNum, sequenceNum, true, true, priv...)
	return err
}
