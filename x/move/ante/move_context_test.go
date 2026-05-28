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

	vmtypes "github.com/initia-labs/movevm/types"
)

const baseDenom = initiaapp.BondDenom

type TestBlockGasMeter struct {
	gasUsed uint64
}

func (t *TestBlockGasMeter) AccumulateGas(ctx context.Context, gas uint64) error {
	t.gasUsed += gas
	return nil
}

func (suite *AnteTestSuite) requireFeePayer(ctx sdk.Context, expected sdk.AccAddress) {
	expectedFeePayer, err := vmtypes.NewAccountAddressFromBytes(expected)
	suite.Require().NoError(err)

	actualFeePayer := movetypes.FeePayer(ctx)
	suite.Require().NotNil(actualFeePayer)
	suite.Require().True(expectedFeePayer.Equals(*actualFeePayer))
}

func (suite *AnteTestSuite) TestMoveContextDecorator() {
	suite.SetupTest() // setup
	suite.txBuilder = suite.clientCtx.TxConfig.NewTxBuilder()

	// keys and addresses
	priv1, _, addr1 := testdata.KeyTestPubAddr()
	priv2, _, feePayer := testdata.KeyTestPubAddr()
	err := suite.txBuilder.SetMsgs(testdata.NewTestMsg(addr1))
	suite.Require().NoError(err)

	feeAmount := sdk.NewCoins(sdk.NewCoin(baseDenom, math.NewInt(100)))
	gasLimit := uint64(200_000)
	suite.txBuilder.SetFeeAmount(feeAmount)
	suite.txBuilder.SetGasLimit(gasLimit)
	suite.txBuilder.SetFeePayer(feePayer)

	privs, accNums, accSeqs := []cryptotypes.PrivKey{priv1, priv2}, []uint64{0, 0}, []uint64{0, 0}
	tx, err := suite.CreateTestTx(privs, accNums, accSeqs, suite.ctx.ChainID())
	suite.Require().NoError(err)

	decorator := ante.NewMoveContextDecorator()

	// in normal mode
	ctx, err := decorator.AnteHandle(suite.ctx.WithIsCheckTx(false), tx, false, nil)
	suite.Require().NoError(err)
	suite.Require().Equal(sdk.NewDecCoinsFromCoins(feeAmount...).QuoDec(math.LegacyNewDec(int64(gasLimit))), ctx.Value(movetypes.GasPricesContextKey).(sdk.DecCoins))
	suite.requireFeePayer(ctx, feePayer)

	// in simulation mode
	ctx, err = decorator.AnteHandle(suite.ctx, tx, true, nil)
	suite.Require().NoError(err)
	suite.Require().Nil(ctx.Value(movetypes.GasPricesContextKey))
	suite.requireFeePayer(ctx, feePayer)
}
