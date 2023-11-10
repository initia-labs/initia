package ante_test

import (
	"fmt"

	"cosmossdk.io/math"
	"github.com/initia-labs/initia/app"
	"github.com/initia-labs/initia/x/move/ante"
	"github.com/initia-labs/initia/x/move/types"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const baseDenom = app.BondDenom

type TestAnteKeeper struct {
	pools           map[string][]math.Int
	weights         map[string][]sdk.Dec
	baseDenom       string
	baseMinGasPrice sdk.Dec
}

func (k TestAnteKeeper) HasDexPair(_ sdk.Context, denomQuote string) (bool, error) {
	_, found := k.pools[denomQuote]
	if !found {
		return false, nil
	}

	_, found = k.weights[denomQuote]
	if !found {
		return false, nil
	}

	return true, nil
}

func (k TestAnteKeeper) GetPoolSpotPrice(_ sdk.Context, denomQuote string) (quotePrice sdk.Dec, err error) {
	balances, found := k.pools[denomQuote]
	if !found {
		return math.LegacyZeroDec(), fmt.Errorf("not found")
	}

	weights, found := k.weights[denomQuote]
	if !found {
		return math.LegacyZeroDec(), fmt.Errorf("not found")
	}

	return types.GetPoolSpotPrice(balances[0], balances[1], weights[0], weights[1]), nil
}

func (k TestAnteKeeper) BaseDenom(_ sdk.Context) (res string) {
	return k.baseDenom
}

func (k TestAnteKeeper) BaseMinGasPrice(ctx sdk.Context) sdk.Dec {
	return k.baseMinGasPrice
}

func (suite *AnteTestSuite) TestEnsureMempoolFees() {
	suite.SetupTest(true) // setup
	suite.txBuilder = suite.clientCtx.TxConfig.NewTxBuilder()

	dexPools := make(map[string][]math.Int)
	dexPools["atom"] = []math.Int{
		sdk.NewInt(1), // base
		sdk.NewInt(2), // quote
	}

	dexWeights := make(map[string][]sdk.Dec)
	dexWeights["atom"] = []sdk.Dec{
		sdk.NewDecWithPrec(2, 1), // base
		sdk.NewDecWithPrec(8, 1), // quote
	}

	// set price 0.5 base == 1 quote
	fc := ante.NewMempoolFeeChecker(TestAnteKeeper{
		pools:           dexPools,
		weights:         dexWeights,
		baseDenom:       baseDenom,
		baseMinGasPrice: math.LegacyZeroDec(),
	})

	// keys and addresses
	priv1, _, addr1 := testdata.KeyTestPubAddr()

	// msg and signatures
	// gas price 0.0005
	msg := testdata.NewTestMsg(addr1)
	feeAmount := sdk.NewCoins(sdk.NewCoin(baseDenom, sdk.NewInt(100)))
	gasLimit := uint64(200_000)
	atomFeeAmount := sdk.NewCoins(sdk.NewCoin("atom", sdk.NewInt(200)))

	suite.Require().NoError(suite.txBuilder.SetMsgs(msg))
	suite.txBuilder.SetFeeAmount(feeAmount)
	suite.txBuilder.SetGasLimit(gasLimit)

	privs, accNums, accSeqs := []cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}
	tx, err := suite.CreateTestTx(privs, accNums, accSeqs, suite.ctx.ChainID())
	suite.Require().NoError(err)

	suite.txBuilder.SetFeeAmount(atomFeeAmount)
	tx2, err := suite.CreateTestTx(privs, accNums, accSeqs, suite.ctx.ChainID())
	suite.Require().NoError(err)

	// Set high gas price so standard test fee fails
	// gas price = 0.004
	basePrice := sdk.NewDecCoinFromDec(baseDenom, sdk.NewDecWithPrec(4, 3))
	highGasPrice := []sdk.DecCoin{basePrice}
	suite.ctx = suite.ctx.WithMinGasPrices(highGasPrice)

	// Set IsCheckTx to true
	suite.ctx = suite.ctx.WithIsCheckTx(true)

	// antehandler errors with insufficient fees
	_, _, err = fc.CheckTxFeeWithMinGasPrices(suite.ctx, tx)
	suite.Require().NotNil(err, "Decorator should have errored on too low fee for local gasPrice")

	// Set IsCheckTx to false
	suite.ctx = suite.ctx.WithIsCheckTx(false)

	// antehandler should not error since we do not check minGasPrice in DeliverTx
	_, _, err = fc.CheckTxFeeWithMinGasPrices(suite.ctx, tx)
	suite.Require().Nil(err, "MempoolFeeDecorator returned error in DeliverTx")

	// Set IsCheckTx back to true for testing sufficient mempool fee
	suite.ctx = suite.ctx.WithIsCheckTx(true)

	// gas price = 0.0005
	basePrice = sdk.NewDecCoinFromDec(baseDenom, sdk.NewDecWithPrec(5, 4))
	lowGasPrice := []sdk.DecCoin{basePrice}
	suite.ctx = suite.ctx.WithMinGasPrices(lowGasPrice)

	_, _, err = fc.CheckTxFeeWithMinGasPrices(suite.ctx, tx)
	suite.Require().Nil(err, "Decorator should not have errored on fee higher than local gasPrice")

	_, _, err = fc.CheckTxFeeWithMinGasPrices(suite.ctx, tx2)
	suite.Require().Nil(err, "Decorator should not have errored on fee higher than local gasPrice")

	// set high base_min_gas_price to test should be failed
	fc = ante.NewMempoolFeeChecker(TestAnteKeeper{
		pools:           dexPools,
		weights:         dexWeights,
		baseDenom:       baseDenom,
		baseMinGasPrice: sdk.NewDecWithPrec(4, 3),
	})

	_, _, err = fc.CheckTxFeeWithMinGasPrices(suite.ctx, tx)
	suite.Require().NotNil(err, "Decorator should have errored on too low fee for local gasPrice")
}
