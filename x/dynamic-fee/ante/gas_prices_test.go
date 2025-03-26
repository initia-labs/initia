package ante_test

import (
	"context"

	"cosmossdk.io/math"
	"github.com/initia-labs/initia/x/dynamic-fee/ante"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type TestBlockGasMeter struct {
	gasUsed uint64
}

func (t *TestBlockGasMeter) AccumulateGas(ctx context.Context, gas uint64) error {
	t.gasUsed += gas
	return nil
}

func (suite *AnteTestSuite) TestGasPricesDecorator() {
	suite.SetupTest() // setup
	suite.txBuilder = suite.clientCtx.TxConfig.NewTxBuilder()

	// keys and addresses
	priv1, _, _ := testdata.KeyTestPubAddr()

	feeAmount := sdk.NewCoins(sdk.NewCoin(baseDenom, math.NewInt(100)))
	gasLimit := uint64(200_000)
	suite.txBuilder.SetFeeAmount(feeAmount)
	suite.txBuilder.SetGasLimit(gasLimit)

	privs, accNums, accSeqs := []cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}
	tx, err := suite.CreateTestTx(privs, accNums, accSeqs, suite.ctx.ChainID())
	suite.Require().NoError(err)

	blockGasMeter := &TestBlockGasMeter{}
	decorator := ante.NewGasPricesDecorator(blockGasMeter)

	// in normal mode
	ctx, err := decorator.AnteHandle(suite.ctx.WithIsCheckTx(false), tx, false, nil)
	suite.Require().NoError(err)
	suite.Require().Equal(sdk.NewDecCoinsFromCoins(feeAmount...).QuoDec(math.LegacyNewDec(int64(gasLimit))), ctx.Value(ante.GasPricesContextKey).(sdk.DecCoins))

	// incremented in normal mode
	suite.Require().Equal(gasLimit, blockGasMeter.gasUsed)

	// in simulation mode
	ctx, err = decorator.AnteHandle(suite.ctx, tx, true, nil)
	suite.Require().NoError(err)
	suite.Require().Nil(ctx.Value(ante.GasPricesContextKey))

	// not incremented in simulation mode
	suite.Require().Equal(gasLimit, blockGasMeter.gasUsed)
}
