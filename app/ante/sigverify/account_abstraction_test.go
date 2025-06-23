package sigverify_test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"crypto/ed25519"

	"github.com/stretchr/testify/suite"
	"golang.org/x/crypto/sha3"
	"google.golang.org/protobuf/types/known/anypb"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	txsigning "cosmossdk.io/x/tx/signing"
	dbm "github.com/cosmos/cosmos-db"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
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
	"github.com/initia-labs/initia/app/ante/sigverify"
	initiatx "github.com/initia-labs/initia/tx"
	moveconfig "github.com/initia-labs/initia/x/move/config"
	movetypes "github.com/initia-labs/initia/x/move/types"
	"github.com/initia-labs/movevm/precompile"
	vmtypes "github.com/initia-labs/movevm/types"

	oracleconfig "github.com/skip-mev/connect/v2/oracle/config"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

var public_key_authenticator = ReadMoveFile("public_key_authenticator")

func ReadMoveFile(filename string) []byte {
	path := "../../../x/move/keeper/binaries/" + filename + ".mv"
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return b
}

// AnteTestSuite is a test suite to be used with ante handler tests.
type AnteTestSuite struct {
	suite.Suite

	app       *initiaapp.InitiaApp
	ctx       sdk.Context
	clientCtx client.Context
	txBuilder client.TxBuilder
}

// returns context and app with params set on account keeper
func (suite *AnteTestSuite) createTestApp(tempDir string) (*initiaapp.InitiaApp, sdk.Context) {
	appOptions := make(simtestutil.AppOptionsMap, 0)
	appOptions[flags.FlagHome] = tempDir
	appOptions[server.FlagInvCheckPeriod] = simcli.FlagPeriodValue

	app := initiaapp.NewInitiaApp(
		log.NewNopLogger(), dbm.NewMemDB(), nil, true, moveconfig.DefaultMoveConfig(), oracleconfig.NewDefaultAppConfig(), appOptions,
	)
	ctx := app.BaseApp.NewUncachedContext(false, tmproto.Header{})
	err := app.AccountKeeper.Params.Set(ctx, authtypes.DefaultParams())
	suite.NoError(err)

	moveParams := movetypes.DefaultParams()
	app.MoveKeeper.SetParams(ctx, moveParams)

	// load stdlib module bytes
	moduleBytes, err := precompile.ReadStdlib()
	suite.Require().NoError(err)

	err = app.MoveKeeper.Initialize(ctx, moduleBytes, moveParams.AllowedPublishers, initiaapp.BondDenom)
	suite.Require().NoError(err)

	return app, ctx
}

// SetupTest setups a new test, with new app, context, and anteHandler.
func (suite *AnteTestSuite) SetupTest() {
	tempDir := suite.T().TempDir()
	suite.app, suite.ctx = suite.createTestApp(tempDir)
	suite.ctx = suite.ctx.WithBlockHeight(1)
	suite.ctx = suite.ctx.WithChainID("interwoven-1")

	// Set up TxConfig.
	encodingConfig := initiaapp.MakeEncodingConfig()

	suite.clientCtx = client.Context{}.
		WithTxConfig(encodingConfig.TxConfig)
}

func (suite *AnteTestSuite) CreateAccountAbstractionTransferTx(priv cryptotypes.PrivKey, receiverAddr sdk.AccAddress, accNum uint64, accSeq uint64, chainID string, signerPriv ed25519.PrivateKey, signerPub ed25519.PublicKey) (authsign.Tx, error) {
	err := suite.txBuilder.SetMsgs(banktypes.NewMsgSend(
		sdk.AccAddress(priv.PubKey().Address()),
		receiverAddr,
		sdk.NewCoins(sdk.NewCoin(initiaapp.BondDenom, math.NewInt(10))),
	))
	if err != nil {
		return nil, err
	}

	// First round: we gather all the signer infos. We use the "set empty
	// signature" hack to do that.
	sigV2 := signing.SignatureV2{
		PubKey: priv.PubKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  initiatx.Signing_SignMode_ACCOUNT_ABSTRACTION,
			Signature: nil,
		},
		Sequence: accSeq,
	}
	err = suite.txBuilder.SetSignatures(sigV2)
	if err != nil {
		return nil, err
	}

	// Second round: all signer infos are set, so each signer can sign.
	signerData := authsign.SignerData{
		Address:       sdk.AccAddress(priv.PubKey().Address()).String(),
		ChainID:       chainID,
		AccountNumber: accNum,
		Sequence:      accSeq,
		PubKey:        priv.PubKey(),
	}
	sigV2, err = AASignWithEd25519PrivKey(
		context.TODO(), initiatx.Signing_SignMode_ACCOUNT_ABSTRACTION, signerData,
		suite.txBuilder, signerPriv, signerPub, suite.clientCtx.TxConfig, accSeq)
	if err != nil {
		return nil, err
	}
	err = suite.txBuilder.SetSignatures(sigV2)
	if err != nil {
		return nil, err
	}

	return suite.txBuilder.GetTx(), nil
}

func AASignWithEd25519PrivKey(
	ctx context.Context,
	signMode signing.SignMode, signerData authsign.SignerData,
	txBuilder client.TxBuilder, priv ed25519.PrivateKey, pub ed25519.PublicKey, txConfig client.TxConfig,
	accSeq uint64,
) (signing.SignatureV2, error) {
	var sigV2 signing.SignatureV2

	tx := txBuilder.GetTx()
	adaptableTx, ok := tx.(authsign.V2AdaptableTx)
	if !ok {
		return signing.SignatureV2{}, fmt.Errorf("expected tx to be V2AdaptableTx, got %T", tx)
	}
	txData := adaptableTx.GetSigningTxData()

	txSignMode, err := sigverify.InternalSignModeToAPI(signMode)
	if err != nil {
		return signing.SignatureV2{}, err
	}

	var pubKey *anypb.Any
	if signerData.PubKey != nil {
		anyPk, err := codectypes.NewAnyWithValue(signerData.PubKey)
		if err != nil {
			return signing.SignatureV2{}, err
		}

		pubKey = &anypb.Any{
			TypeUrl: anyPk.TypeUrl,
			Value:   anyPk.Value,
		}
	}
	txSignerData := txsigning.SignerData{
		ChainID:       signerData.ChainID,
		AccountNumber: signerData.AccountNumber,
		Sequence:      signerData.Sequence,
		Address:       signerData.Address,
		PubKey:        pubKey,
	}
	signBytes, err := txConfig.SignModeHandler().GetSignBytes(ctx, txSignMode, txSignerData, txData)
	if err != nil {
		return signing.SignatureV2{}, err
	}

	digest := sha3.Sum256(signBytes)
	digestBytes := digest[:]

	signature := ed25519.Sign(priv, digestBytes)

	var authenticator []byte

	bcsEncodedPubBz, err := vmtypes.SerializeBytes(pub)
	if err != nil {
		return signing.SignatureV2{}, err
	}

	bcsEncodedSignatureBz, err := vmtypes.SerializeBytes(signature)
	if err != nil {
		return signing.SignatureV2{}, err
	}

	authenticator = append(authenticator, bcsEncodedPubBz...)
	authenticator = append(authenticator, bcsEncodedSignatureBz...)

	abstractionData := &movetypes.AbstractionData{
		FunctionInfo: movetypes.FunctionInfo{
			ModuleAddress: "0xcafe",
			ModuleName:    "public_key_authenticator",
			FunctionName:  "authenticate",
		},
		AuthData: movetypes.AbstractionAuthData{
			V1: &movetypes.V1AuthData{
				SigningMessageDigest: digestBytes,
				Authenticator:        authenticator,
			},
		},
	}

	abstractionDataBz, err := json.Marshal(abstractionData)
	if err != nil {
		return signing.SignatureV2{}, err
	}

	// Construct the SignatureV2 struct
	sigData := signing.SingleSignatureData{
		SignMode:  signMode,
		Signature: abstractionDataBz,
	}

	sigV2 = signing.SignatureV2{
		PubKey:   signerData.PubKey,
		Data:     &sigData,
		Sequence: accSeq,
	}

	return sigV2, nil
}

func (suite *AnteTestSuite) TestAccountAbstractionAuthentication() {
	suite.SetupTest() // setup
	suite.txBuilder = suite.clientCtx.TxConfig.NewTxBuilder()

	cafeAddr, err := vmtypes.NewAccountAddress("0xcafe")
	suite.Require().NoError(err)

	err = suite.app.MoveKeeper.PublishModuleBundle(suite.ctx, cafeAddr, vmtypes.NewModuleBundle([]vmtypes.Module{
		{Code: public_key_authenticator},
	}...), movetypes.UpgradePolicy_COMPATIBLE)
	suite.Require().NoError(err)

	// keys and addresses
	priv1, _, acc1 := testdata.KeyTestPubAddr()
	_, _, acc2 := testdata.KeyTestPubAddr()

	err = suite.app.MoveKeeper.MoveBankKeeper().MintCoins(suite.ctx, acc1, sdk.NewCoins(sdk.NewCoin(initiaapp.BondDenom, math.NewInt(100))))
	suite.Require().NoError(err)

	signerPub, signerPriv, err := ed25519.GenerateKey(nil)
	suite.Require().NoError(err)

	err = suite.app.MoveKeeper.ExecuteEntryFunctionJSON(
		suite.ctx,
		movetypes.ConvertSDKAddressToVMAddress(acc1),
		vmtypes.StdAddress,
		"account_abstraction",
		"add_authentication_function",
		[]vmtypes.TypeTag{},
		[]string{"\"0xcafe\"", "\"public_key_authenticator\"", "\"authenticate\""},
	)
	suite.Require().NoError(err)

	err = suite.app.MoveKeeper.ExecuteEntryFunctionJSON(
		suite.ctx,
		movetypes.ConvertSDKAddressToVMAddress(acc1),
		cafeAddr,
		"public_key_authenticator",
		"permit_public_key",
		[]vmtypes.TypeTag{},
		[]string{fmt.Sprintf("\"%s\"", hex.EncodeToString(signerPub))},
	)
	suite.Require().NoError(err)

	feeAmount := sdk.NewCoins(sdk.NewCoin(initiaapp.BondDenom, math.NewInt(100)))
	gasLimit := uint64(200_000)
	suite.txBuilder.SetFeeAmount(feeAmount)
	suite.txBuilder.SetGasLimit(gasLimit)

	acc := suite.app.AccountKeeper.GetAccount(suite.ctx, acc1)
	err = acc.SetPubKey(priv1.PubKey())
	suite.Require().NoError(err)
	suite.app.AccountKeeper.SetAccount(suite.ctx, acc)

	tx, err := suite.CreateAccountAbstractionTransferTx(priv1, acc2, acc.GetAccountNumber(), acc.GetSequence(), suite.ctx.ChainID(), signerPriv, signerPub)
	suite.Require().NoError(err)

	decorator := sigverify.NewSigVerificationDecorator(suite.app.AccountKeeper, suite.clientCtx.TxConfig.SignModeHandler(), suite.app.MoveKeeper)

	_, err = decorator.AnteHandle(suite.ctx.WithIsCheckTx(false), tx, false, nil)
	suite.Require().NoError(err)
}

func TestAnteTestSuite(t *testing.T) {
	suite.Run(t, new(AnteTestSuite))
}
