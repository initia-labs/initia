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

func (suite *AnteTestSuite) Test_BlockGasDecorator() {
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
	decorator := ante.NewBlockGasDecorator(blockGasMeter)

	// in normal mode
	_, err = decorator.AnteHandle(suite.ctx.WithIsCheckTx(false), tx, false, nil)
	suite.Require().NoError(err)

	// incremented in normal mode
	suite.Require().Equal(gasLimit, blockGasMeter.gasUsed)

	// in check tx mode
	_, err = decorator.AnteHandle(suite.ctx.WithIsCheckTx(true), tx, true, nil)
	suite.Require().NoError(err)

	// not incremented in check tx mode
	suite.Require().Equal(gasLimit, blockGasMeter.gasUsed)

	// in simulation mode
	_, err = decorator.AnteHandle(suite.ctx.WithIsCheckTx(false), tx, true, nil)
	suite.Require().NoError(err)

	// not incremented in simulation mode
	suite.Require().Equal(gasLimit, blockGasMeter.gasUsed)
}
