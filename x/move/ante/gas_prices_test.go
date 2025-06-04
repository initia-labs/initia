package ante_test

import (
	"context"

	"cosmossdk.io/math"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"

	initiaapp "github.com/initia-labs/initia/app"
	"github.com/initia-labs/initia/x/move/ante"
	movetypes "github.com/initia-labs/initia/x/move/types"
)

const baseDenom = initiaapp.BondDenom

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

	decorator := ante.NewGasPricesDecorator()

	// in normal mode
	ctx, err := decorator.AnteHandle(suite.ctx.WithIsCheckTx(false), tx, false, nil)
	suite.Require().NoError(err)
	suite.Require().Equal(sdk.NewDecCoinsFromCoins(feeAmount...).QuoDec(math.LegacyNewDec(int64(gasLimit))), ctx.Value(movetypes.GasPricesContextKey).(sdk.DecCoins))

	// in simulation mode
	ctx, err = decorator.AnteHandle(suite.ctx, tx, true, nil)
	suite.Require().NoError(err)
	suite.Require().Nil(ctx.Value(movetypes.GasPricesContextKey))
}
