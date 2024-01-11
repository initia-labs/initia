package ante_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"cosmossdk.io/log"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/server"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsign "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	simcli "github.com/cosmos/cosmos-sdk/x/simulation/client/cli"

	initiaapp "github.com/initia-labs/initia/app"
	initiaapporacle "github.com/initia-labs/initia/app/oracle"
	moveconfig "github.com/initia-labs/initia/x/move/config"
	movetypes "github.com/initia-labs/initia/x/move/types"
)

// AnteTestSuite is a test suite to be used with ante handler tests.
type AnteTestSuite struct {
	suite.Suite

	app       *initiaapp.InitiaApp
	ctx       sdk.Context
	clientCtx client.Context
	txBuilder client.TxBuilder
}

// returns context and app with params set on account keeper
func (suite *AnteTestSuite) createTestApp(isCheckTx bool, tempDir string) (*initiaapp.InitiaApp, sdk.Context) {
	appOptions := make(simtestutil.AppOptionsMap, 0)
	appOptions[flags.FlagHome] = tempDir
	appOptions[server.FlagInvCheckPeriod] = simcli.FlagPeriodValue

	app := initiaapp.NewInitiaApp(
		log.NewNopLogger(), dbm.NewMemDB(), nil, true, moveconfig.DefaultMoveConfig(), initiaapporacle.DefaultConfig(), appOptions,
	)
	ctx := app.BaseApp.NewContext(isCheckTx)
	err := app.AccountKeeper.Params.Set(ctx, authtypes.DefaultParams())
	suite.NoError(err)

	app.MoveKeeper.SetParams(ctx, movetypes.DefaultParams())

	return app, ctx
}

// SetupTest setups a new test, with new app, context, and anteHandler.
func (suite *AnteTestSuite) SetupTest(isCheckTx bool) {
	tempDir := suite.T().TempDir()
	suite.app, suite.ctx = suite.createTestApp(isCheckTx, tempDir)
	suite.ctx = suite.ctx.WithBlockHeight(1)

	// Set up TxConfig.
	encodingConfig := initiaapp.MakeEncodingConfig()

	// We're using TestMsg encoding in some tests, so register it here.
	encodingConfig.Amino.RegisterConcrete(&testdata.TestMsg{}, "testdata.TestMsg", nil)
	testdata.RegisterInterfaces(encodingConfig.InterfaceRegistry)

	suite.clientCtx = client.Context{}.
		WithTxConfig(encodingConfig.TxConfig)
}

// CreateTestTx is a helper function to create a tx given multiple inputs.
func (suite *AnteTestSuite) CreateTestTx(privs []cryptotypes.PrivKey, accNums []uint64, accSeqs []uint64, chainID string) (authsign.Tx, error) {
	defaultSignMode, err := authsign.APISignModeToInternal(suite.clientCtx.TxConfig.SignModeHandler().DefaultMode())
	suite.NoError(err)

	// First round: we gather all the signer infos. We use the "set empty
	// signature" hack to do that.
	var sigsV2 []signing.SignatureV2
	for i, priv := range privs {

		sigV2 := signing.SignatureV2{
			PubKey: priv.PubKey(),
			Data: &signing.SingleSignatureData{
				SignMode:  defaultSignMode,
				Signature: nil,
			},
			Sequence: accSeqs[i],
		}

		sigsV2 = append(sigsV2, sigV2)
	}
	err = suite.txBuilder.SetSignatures(sigsV2...)
	if err != nil {
		return nil, err
	}

	// Second round: all signer infos are set, so each signer can sign.
	sigsV2 = []signing.SignatureV2{}
	for i, priv := range privs {
		signerData := authsign.SignerData{
			Address:       sdk.AccAddress(priv.PubKey().Address()).String(),
			ChainID:       chainID,
			AccountNumber: accNums[i],
			Sequence:      accSeqs[i],
			PubKey:        priv.PubKey(),
		}
		sigV2, err := tx.SignWithPrivKey(
			context.TODO(), defaultSignMode, signerData,
			suite.txBuilder, priv, suite.clientCtx.TxConfig, accSeqs[i])
		if err != nil {
			return nil, err
		}

		sigsV2 = append(sigsV2, sigV2)
	}
	err = suite.txBuilder.SetSignatures(sigsV2...)
	if err != nil {
		return nil, err
	}

	return suite.txBuilder.GetTx(), nil
}

func TestAnteTestSuite(t *testing.T) {
	suite.Run(t, new(AnteTestSuite))
}
