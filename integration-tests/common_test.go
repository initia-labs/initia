package integration_tests

import (
	"fmt"
	"os"
	"slices"
	"testing"

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

	initiaapp "github.com/initia-labs/initia/app"
	customdistrtypes "github.com/initia-labs/initia/x/distribution/types"
	movetypes "github.com/initia-labs/initia/x/move/types"
	stakingtypes "github.com/initia-labs/initia/x/mstaking/types"
	vmtypes "github.com/initia-labs/movevm/types"
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

func dispatchableTokenDenom(t *testing.T) string {
	deployer, err := vmtypes.NewAccountAddress("0xcafe")
	require.NoError(t, err)

	metadata := movetypes.NamedObjectAddress(deployer, "test_token")
	return movetypes.DenomTraceDenomPrefixMove + metadata.CanonicalString()
}

func dispatchableTokenCoin(t *testing.T) sdk.Coin {
	denom := dispatchableTokenDenom(t)
	// Note: test token get balance will return 10x more than the actual balance
	return sdk.NewCoin(denom, math.NewInt(10000000))
}

func checkBalance(t *testing.T, app *initiaapp.InitiaApp, addr sdk.AccAddress, balances sdk.Coins) {
	ctxCheck := app.NewContext(true)
	require.True(t, balances.Equal(app.BankKeeper.GetAllBalances(ctxCheck, addr)))
}

func CreateApp(t *testing.T) (*initiaapp.InitiaApp, []sdk.AccAddress, []*secp256k1.PrivKey) {
	baseCoin := sdk.NewCoin(bondDenom, math.NewInt(1_000_000_000_000))
	quoteCoin := sdk.NewCoin("uusdc", math.NewInt(2_500_000_000_000))
	dexCoins := sdk.NewCoins(baseCoin, quoteCoin)

	app := initiaapp.SetupWithGenesisAccounts(nil, authtypes.GenesisAccounts{
		&authtypes.BaseAccount{Address: addr1.String()},
		&authtypes.BaseAccount{Address: addr2.String()},
		&authtypes.BaseAccount{Address: movetypes.StdAddr.String()},
	},
		banktypes.Balance{Address: addr1.String(), Coins: genCoins},
		banktypes.Balance{Address: addr2.String(), Coins: genCoins},
		banktypes.Balance{Address: movetypes.StdAddr.String(), Coins: dexCoins},
	)

	checkBalance(t, app, addr1, genCoins)
	checkBalance(t, app, addr2, genCoins)
	checkBalance(t, app, movetypes.StdAddr, dexCoins)

	_, err := app.FinalizeBlock(&abci.RequestFinalizeBlock{Height: app.LastBlockHeight() + 1})
	require.NoError(t, err)

	ctx := app.NewUncachedContext(false, tmproto.Header{})
	createDexPool(t, ctx, app, baseCoin, quoteCoin, math.LegacyNewDecWithPrec(8, 1), math.LegacyNewDecWithPrec(2, 1))

	// create dispatchable token
	createDispatchableToken(t, ctx, app, []sdk.AccAddress{addr1, addr2})

	// set reward weight
	distrParams := customdistrtypes.DefaultParams()
	distrParams.RewardWeights = []customdistrtypes.RewardWeight{
		{Denom: bondDenom, Weight: math.LegacyOneDec()},
	}
	require.NoError(t, app.DistrKeeper.Params.Set(ctx, distrParams))
	app.StakingKeeper.SetBondDenoms(ctx, []string{bondDenom, secondBondDenom})

	// fund second bond coin
	app.BankKeeper.SendCoins(ctx, movetypes.StdAddr, addr1, sdk.NewCoins(secondBondCoin))

	_, err = app.Commit()
	require.NoError(t, err)

	// create validator
	description := stakingtypes.NewDescription("foo_moniker", "", "", "", "")
	createValidatorMsg, err := stakingtypes.NewMsgCreateValidator(
		sdk.ValAddress(addr1).String(), valKey.PubKey(), sdk.NewCoins(bondCoin, secondBondCoin), description, commissionRates,
	)
	require.NoError(t, err)

	err = executeMsgs(t, app, []sdk.Msg{createValidatorMsg}, []uint64{0}, []uint64{0}, true, true, priv1)
	require.NoError(t, err)

	checkBalance(t, app, addr1, genCoins.Sub(bondCoin).Add(dispatchableTokenCoin(t)))

	return app, []sdk.AccAddress{addr1, addr2}, []*secp256k1.PrivKey{priv1, priv2}
}

func executeMsgs(t *testing.T, app *initiaapp.InitiaApp, msgs []sdk.Msg, accountNum []uint64, sequenceNum []uint64, expectSimPass bool, expectPass bool, priv ...cryptotypes.PrivKey) error {
	txGen := initiaapp.MakeEncodingConfig().TxConfig
	header := tmproto.Header{Height: app.LastBlockHeight() + 1}
	_, _, err := testutilsims.SignCheckDeliver(t, txGen, app.BaseApp, header, msgs, "", accountNum, sequenceNum, expectSimPass, expectPass, priv...)
	return err
}

// func executeMsgsWithGasInfo(t *testing.T, app *initiaapp.InitiaApp, msgs []sdk.Msg, accountNum []uint64, sequenceNum []uint64, priv ...cryptotypes.PrivKey) (sdk.GasInfo, error) {
// 	txGen := initiaapp.MakeEncodingConfig().TxConfig
// 	header := tmproto.Header{Height: app.LastBlockHeight() + 1}
// 	gas, _, err := testutilsims.SignCheckDeliver(t, txGen, app.BaseApp, header, msgs, "", accountNum, sequenceNum, true, true, priv...)
// 	return gas, err
// }

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
	metadataBase, err := movetypes.MetadataAddressFromDenom(baseCoin.Denom)
	require.NoError(t, err)

	metadataQuote, err := movetypes.MetadataAddressFromDenom(quoteCoin.Denom)
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

	return movetypes.NamedObjectAddress(vmtypes.StdAddress, denomLP)
}

func createDispatchableToken(t *testing.T, ctx sdk.Context, app *initiaapp.InitiaApp, receivers []sdk.AccAddress) {
	deployer, err := vmtypes.NewAccountAddress("0xcafe")
	require.NoError(t, err)
	dispatchableTokenModule := readMoveFile("test_dispatchable_token")
	err = app.MoveKeeper.PublishModuleBundle(ctx, deployer, vmtypes.NewModuleBundle(vmtypes.NewModule(dispatchableTokenModule)), movetypes.UpgradePolicy_COMPATIBLE)
	require.NoError(t, err)

	err = app.MoveKeeper.ExecuteEntryFunctionJSON(ctx, deployer, deployer, "test_dispatchable_token", "initialize", []vmtypes.TypeTag{}, []string{})
	require.NoError(t, err)

	for _, receiver := range receivers {
		receiverAddr, err := vmtypes.NewAccountAddressFromBytes(receiver.Bytes())
		require.NoError(t, err)
		// Note: test token get balance will return 10x more than the actual balance
		err = app.MoveKeeper.ExecuteEntryFunctionJSON(ctx, deployer, deployer, "test_dispatchable_token", "mint", []vmtypes.TypeTag{}, []string{fmt.Sprintf("\"%s\"", receiverAddr.String()), `"1000000"`})
		require.NoError(t, err)
	}
}

// readMoveFile reads a Move file from the x/move/binaries directory
func readMoveFile(filename string) []byte {
	path := "../x/move/keeper/binaries/" + filename + ".mv"
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return b
}
