package sigverify_test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/sha3"
	"google.golang.org/protobuf/types/known/anypb"

	"cosmossdk.io/math"
	txsigning "cosmossdk.io/x/tx/signing"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"

	"github.com/cosmos/cosmos-sdk/client"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsign "github.com/cosmos/cosmos-sdk/x/auth/signing"

	initiaapp "github.com/initia-labs/initia/app"
	"github.com/initia-labs/initia/app/ante/sigverify"
	initiatx "github.com/initia-labs/initia/tx"
	movetypes "github.com/initia-labs/initia/x/move/types"
	vmtypes "github.com/initia-labs/movevm/types"

	"github.com/initia-labs/initia/crypto/derivable"
	ethsecp256k1 "github.com/initia-labs/initia/crypto/ethsecp256k1"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

func createAbstractPublicKey(
	ethereumAddress []byte,
	domain string,
) ([]byte, error) {
	encodedEthereumAddress, err := vmtypes.SerializeBytes(ethereumAddress)
	if err != nil {
		return nil, err
	}

	encodedDomain, err := vmtypes.SerializeString(domain)
	if err != nil {
		return nil, err
	}

	var res []byte
	res = append(res, encodedEthereumAddress...)
	res = append(res, encodedDomain...)
	return res, nil
}

func createAbstractSignature(
	scheme string,
	issuedAt string,
	signature []byte,
) ([]byte, error) {
	var res []byte
	/*
		enum SIWEAbstractSignature has drop {
			/// Deprecated, use MessageV2 instead
			MessageV1 {
				/// The date and time when the signature was issued
				issued_at: String,
				/// The signature of the message
				signature: vector<u8>
			},
			MessageV2 {
				/// The scheme in the URI of the message, e.g. the scheme of the website that requested the signature (http, https, etc.)
				scheme: String,
				/// The date and time when the signature was issued
				issued_at: String,
				/// The signature of the message
				signature: vector<u8>
			}
		}
	*/

	encodedType, err := vmtypes.SerializeUint8(0x01)
	if err != nil {
		return nil, err
	}

	encodedScheme, err := vmtypes.SerializeString(scheme)
	if err != nil {
		return nil, err
	}

	encodedIssuedAt, err := vmtypes.SerializeString(issuedAt)
	if err != nil {
		return nil, err
	}

	encodedSignature, err := vmtypes.SerializeBytes(signature)
	if err != nil {
		return nil, err
	}

	res = append(res, encodedType...) // MessageV2 type
	res = append(res, encodedScheme...)
	res = append(res, encodedIssuedAt...)
	res = append(res, encodedSignature...)
	return res, nil
}

func constructMessage(
	ethereumAddress string,
	domain string,
	digestUtf8 string,
	issuedAt string,
	scheme string,
	chainId string,
) ([]byte, error) {
	message := fmt.Sprintf("%s wants you to sign in with your Ethereum account:\n%s\n\nPlease confirm you explicitly initiated this request from %s. You are approving to execute transaction on Initia blockchain (%s).\n\nURI: %s://%s\nVersion: 1\nChain ID: %s\nNonce: %s\nIssued At: %s", domain, ethereumAddress, domain, chainId, scheme, domain, chainId, digestUtf8, issuedAt)
	msgLen := len(message)

	prefix := []byte("\x19Ethereum Signed Message:\n")
	msgLenBytes := []byte(strconv.Itoa(msgLen))

	var fullMessage []byte
	fullMessage = append(fullMessage, prefix...)
	fullMessage = append(fullMessage, msgLenBytes...)
	fullMessage = append(fullMessage, []byte(message)...)

	return fullMessage, nil
}

func (suite *AnteTestSuite) TestEthereumDerivableAccountAbstraction() {
	suite.SetupTest() // setup
	suite.txBuilder = suite.clientCtx.TxConfig.NewTxBuilder()

	// keys and addresses

	signerPriv := ethsecp256k1.GenerateKey()
	signerPub := signerPriv.PubKey()

	_, _, acc2 := testdata.KeyTestPubAddr()

	signerEthereumAddress := "0x" + strings.ToLower(hex.EncodeToString(signerPub.Address().Bytes()))
	abstractPublicKey, err := createAbstractPublicKey([]byte(signerEthereumAddress), "localhost:3001")
	suite.Require().NoError(err)

	res, _, err := suite.app.MoveKeeper.ExecuteViewFunctionJSON(
		suite.ctx,
		vmtypes.StdAddress,
		"account_abstraction",
		"derive_account_address_view",
		[]vmtypes.TypeTag{},
		[]string{
			"\"0x1\"",
			"\"ethereum_derivable_account\"",
			"\"authenticate\"",
			fmt.Sprintf("\"%s\"", hex.EncodeToString(abstractPublicKey)),
		},
	)
	suite.Require().NoError(err)

	daaAddress, err := movetypes.AccAddressFromString(suite.app.AccountKeeper.AddressCodec(), res.Ret[1:len(res.Ret)-1])
	suite.Require().NoError(err)

	daaAccAddress := movetypes.ConvertVMAddressToSDKAddress(daaAddress)

	err = suite.app.MoveKeeper.MoveBankKeeper().MintCoins(suite.ctx, daaAccAddress, sdk.NewCoins(sdk.NewCoin(initiaapp.BondDenom, math.NewInt(100))))
	suite.Require().NoError(err)

	feeAmount := sdk.NewCoins(sdk.NewCoin(initiaapp.BondDenom, math.NewInt(100)))
	gasLimit := uint64(200_000)
	suite.txBuilder.SetFeeAmount(feeAmount)
	suite.txBuilder.SetGasLimit(gasLimit)

	acc := suite.app.AccountKeeper.GetAccount(suite.ctx, daaAccAddress)
	pubkey := derivable.NewPubKey("0x1", "ethereum_derivable_account", "authenticate", abstractPublicKey)
	suite.Require().Equal(pubkey.Address().Bytes(), daaAddress.Bytes())
	err = acc.SetPubKey(pubkey)
	suite.Require().NoError(err)
	suite.app.AccountKeeper.SetAccount(suite.ctx, acc)

	tx, err := suite.CreateEthereumDerivableAccountAbstractionTransferTx(daaAccAddress, acc2, acc.GetAccountNumber(), acc.GetSequence(), suite.ctx.ChainID(), signerPriv, signerPub, signerEthereumAddress, abstractPublicKey)
	suite.Require().NoError(err)

	decorator := sigverify.NewSigVerificationDecorator(suite.app.AccountKeeper, suite.clientCtx.TxConfig.SignModeHandler(), suite.app.MoveKeeper)

	_, err = decorator.AnteHandle(suite.ctx.WithIsCheckTx(false), tx, false, nil)
	suite.Require().NoError(err)
}

func (suite *AnteTestSuite) CreateEthereumDerivableAccountAbstractionTransferTx(daaAccAddress sdk.AccAddress, receiverAddr sdk.AccAddress, accNum uint64, accSeq uint64, chainID string, signerPriv cryptotypes.PrivKey, signerPub cryptotypes.PubKey, ethereumAddress string, abstractPublicKey []byte) (authsign.Tx, error) {
	err := suite.txBuilder.SetMsgs(banktypes.NewMsgSend(
		daaAccAddress,
		receiverAddr,
		sdk.NewCoins(sdk.NewCoin(initiaapp.BondDenom, math.NewInt(10))),
	))
	if err != nil {
		return nil, err
	}

	// First round: we gather all the signer infos. We use the "set empty
	// signature" hack to do that.
	sigV2 := signing.SignatureV2{
		PubKey: derivable.NewPubKey("0x1", "ethereum_derivable_account", "authenticate", abstractPublicKey),
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
		Address:       daaAccAddress.String(),
		ChainID:       chainID,
		AccountNumber: accNum,
		Sequence:      accSeq,
		PubKey:        derivable.NewPubKey("0x1", "ethereum_derivable_account", "authenticate", abstractPublicKey),
	}
	sigV2, err = DAAEthereumSign(
		context.TODO(), initiatx.Signing_SignMode_ACCOUNT_ABSTRACTION, signerData,
		suite.txBuilder, signerPriv, signerPub, suite.clientCtx.TxConfig, accSeq, ethereumAddress, abstractPublicKey, chainID)
	if err != nil {
		return nil, err
	}
	err = suite.txBuilder.SetSignatures(sigV2)
	if err != nil {
		return nil, err
	}

	return suite.txBuilder.GetTx(), nil
}

func DAAEthereumSign(
	ctx context.Context,
	signMode signing.SignMode, signerData authsign.SignerData,
	txBuilder client.TxBuilder, signerPriv cryptotypes.PrivKey, signerPub cryptotypes.PubKey, txConfig client.TxConfig,
	accSeq uint64, ethereumAddress string, abstractPublicKey []byte, chainID string,
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
	digestHex := "0x" + hex.EncodeToString(digestBytes)

	message, err := constructMessage(
		ethereumAddress,
		"localhost:3001",
		digestHex,
		"2025-01-01T00:00:00.000Z",
		"http",
		chainID,
	)
	if err != nil {
		return signing.SignatureV2{}, err
	}

	signature, err := signerPriv.Sign(message)
	if err != nil {
		return signing.SignatureV2{}, err
	}

	abstractSignature, err := createAbstractSignature("http", "2025-01-01T00:00:00.000Z", signature)
	if err != nil {
		return signing.SignatureV2{}, err
	}

	abstractionData := &vmtypes.AbstractionData{
		FunctionInfo: vmtypes.FunctionInfo{
			ModuleAddress: vmtypes.StdAddress,
			ModuleName:    "ethereum_derivable_account",
			FunctionName:  "authenticate",
		},
		AuthData: &vmtypes.AbstractionAuthData__DerivableV1{
			SigningMessageDigest: digestBytes,
			AbstractSignature:    abstractSignature,
			AbstractPublicKey:    abstractPublicKey,
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
