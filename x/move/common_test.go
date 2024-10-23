package move_test

import (
	"slices"
	"testing"

	initiaapp "github.com/initia-labs/initia/app"
	customdistrtypes "github.com/initia-labs/initia/x/distribution/types"
	"github.com/initia-labs/initia/x/move/types"
	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"
	vmtypes "github.com/initia-labs/movevm/types"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	testutilsims "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
)

// Bond denom should be set for staking test
const bondDenom = initiaapp.BondDenom
const secondBondDenom = "ulp"

var (
	priv1 = secp256k1.GenPrivKey()
	addr1 = sdk.AccAddress(priv1.PubKey().Address())
	priv2 = secp256k1.GenPrivKey()
	addr2 = sdk.AccAddress(priv2.PubKey().Address())

	valKey = ed25519.GenPrivKey()

	commissionRates = stakingtypes.NewCommissionRates(math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec())

	genCoins       = sdk.NewCoins(sdk.NewCoin(bondDenom, math.NewInt(5000000))).Sort()
	bondCoin       = sdk.NewCoin(bondDenom, math.NewInt(1_000_000))
	secondBondCoin = sdk.NewCoin(secondBondDenom, math.NewInt(1_000_000))
)

func checkBalance(t *testing.T, app *initiaapp.InitiaApp, addr sdk.AccAddress, balances sdk.Coins) {
	ctxCheck := app.BaseApp.NewContext(true)
	require.True(t, balances.Equal(app.BankKeeper.GetAllBalances(ctxCheck, addr)))
}

func createApp(t *testing.T) *initiaapp.InitiaApp {
	baseCoin := sdk.NewCoin(bondDenom, math.NewInt(1_000_000_000_000))
	quoteCoin := sdk.NewCoin("uusdc", math.NewInt(2_500_000_000_000))
	dexCoins := sdk.NewCoins(baseCoin, quoteCoin)

	app := initiaapp.SetupWithGenesisAccounts(nil, authtypes.GenesisAccounts{
		&authtypes.BaseAccount{Address: addr1.String()},
		&authtypes.BaseAccount{Address: addr2.String()},
		&authtypes.BaseAccount{Address: types.StdAddr.String()},
	},
		banktypes.Balance{Address: addr1.String(), Coins: genCoins},
		banktypes.Balance{Address: addr2.String(), Coins: genCoins},
		banktypes.Balance{Address: types.StdAddr.String(), Coins: dexCoins},
	)

	checkBalance(t, app, addr1, genCoins)
	checkBalance(t, app, addr2, genCoins)
	checkBalance(t, app, types.StdAddr, dexCoins)

	_, err := app.FinalizeBlock(&abci.RequestFinalizeBlock{Height: app.LastBlockHeight() + 1})
	require.NoError(t, err)

	ctx := app.BaseApp.NewUncachedContext(false, tmproto.Header{})
	createDexPool(t, ctx, app, baseCoin, quoteCoin, math.LegacyNewDecWithPrec(8, 1), math.LegacyNewDecWithPrec(2, 1))

	// set reward weight
	distrParams := customdistrtypes.DefaultParams()
	distrParams.RewardWeights = []customdistrtypes.RewardWeight{
		{Denom: bondDenom, Weight: math.LegacyOneDec()},
	}
	require.NoError(t, app.DistrKeeper.Params.Set(ctx, distrParams))
	app.StakingKeeper.SetBondDenoms(ctx, []string{bondDenom, secondBondDenom})

	// fund second bond coin
	app.BankKeeper.SendCoins(ctx, types.StdAddr, addr1, sdk.NewCoins(secondBondCoin))

	_, err = app.Commit()
	require.NoError(t, err)

	// create validator
	description := stakingtypes.NewDescription("foo_moniker", "", "", "", "")
	createValidatorMsg, err := stakingtypes.NewMsgCreateValidator(
		sdk.ValAddress(addr1).String(), valKey.PubKey(), sdk.NewCoins(bondCoin, secondBondCoin), description, commissionRates,
	)
	require.NoError(t, err)

	err = executeMsgs(t, app, []sdk.Msg{createValidatorMsg}, []uint64{0}, []uint64{0}, priv1)
	require.NoError(t, err)

	checkBalance(t, app, addr1, genCoins.Sub(bondCoin))

	return app
}

func executeMsgs(t *testing.T, app *initiaapp.InitiaApp, msgs []sdk.Msg, accountNum []uint64, sequenceNum []uint64, priv ...cryptotypes.PrivKey) error {
	txGen := initiaapp.MakeEncodingConfig().TxConfig
	header := tmproto.Header{Height: app.LastBlockHeight() + 1}
	_, _, err := testutilsims.SignCheckDeliver(t, txGen, app.BaseApp, header, msgs, "", accountNum, sequenceNum, true, true, priv...)
	return err
}

func decToVmArgument(t *testing.T, val math.LegacyDec) []byte {
	// big-endian bytes (bytes are cloned)
	bz := val.BigInt().Bytes()

	// reverse bytes to little-endian
	slices.Reverse(bz)

	// serialize bytes
	bz, err := vmtypes.SerializeBytes(bz)
	require.NoError(t, err)

	return bz
}

func createDexPool(
	t *testing.T, ctx sdk.Context, app *initiaapp.InitiaApp,
	baseCoin sdk.Coin, quoteCoin sdk.Coin,
	weightBase math.LegacyDec, weightQuote math.LegacyDec,
) (metadataLP vmtypes.AccountAddress) {
	metadataBase, err := types.MetadataAddressFromDenom(baseCoin.Denom)
	require.NoError(t, err)

	metadataQuote, err := types.MetadataAddressFromDenom(quoteCoin.Denom)
	require.NoError(t, err)

	denomLP := "ulp"

	//
	// prepare arguments
	//

	name, err := vmtypes.SerializeString("LP Coin")
	require.NoError(t, err)

	symbol, err := vmtypes.SerializeString(denomLP)
	require.NoError(t, err)

	// 0.003 == 0.3%
	swapFeeBz := decToVmArgument(t, math.LegacyNewDecWithPrec(3, 3))
	weightBaseBz := decToVmArgument(t, weightBase)
	weightQuoteBz := decToVmArgument(t, weightQuote)

	baseAmount, err := vmtypes.SerializeUint64(baseCoin.Amount.Uint64())
	require.NoError(t, err)

	quoteAmount, err := vmtypes.SerializeUint64(quoteCoin.Amount.Uint64())
	require.NoError(t, err)

	err = app.MoveKeeper.ExecuteEntryFunction(
		ctx,
		vmtypes.StdAddress,
		vmtypes.StdAddress,
		"dex",
		"create_pair_script",
		[]vmtypes.TypeTag{},
		[][]byte{
			name,
			symbol,
			swapFeeBz,
			weightBaseBz,
			weightQuoteBz,
			metadataBase[:],
			metadataQuote[:],
			baseAmount,
			quoteAmount,
		},
	)
	require.NoError(t, err)

	return types.NamedObjectAddress(vmtypes.StdAddress, denomLP)
}
