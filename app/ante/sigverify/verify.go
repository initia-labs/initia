package sigverify

import (
	"context"
	"fmt"

	signingv1beta1 "cosmossdk.io/api/cosmos/tx/signing/v1beta1"
	txsigning "cosmossdk.io/x/tx/signing"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/crypto/types/multisig"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"

	"github.com/initia-labs/initia/crypto/ethsecp256k1"
)

// internalSignModeToAPI converts a signing.SignMode to a protobuf SignMode.
func internalSignModeToAPI(mode signing.SignMode) (signingv1beta1.SignMode, error) {
	switch mode {
	case signing.SignMode_SIGN_MODE_DIRECT:
		return signingv1beta1.SignMode_SIGN_MODE_DIRECT, nil
	case signing.SignMode_SIGN_MODE_LEGACY_AMINO_JSON:
		return signingv1beta1.SignMode_SIGN_MODE_LEGACY_AMINO_JSON, nil
	case signing.SignMode_SIGN_MODE_TEXTUAL:
		return signingv1beta1.SignMode_SIGN_MODE_TEXTUAL, nil
	case signing.SignMode_SIGN_MODE_DIRECT_AUX:
		return signingv1beta1.SignMode_SIGN_MODE_DIRECT_AUX, nil
	case signing.SignMode_SIGN_MODE_EIP_191:
		return signingv1beta1.SignMode_SIGN_MODE_EIP_191, nil //nolint
	default:
		return signingv1beta1.SignMode_SIGN_MODE_UNSPECIFIED, fmt.Errorf("unsupported sign mode %s", mode)
	}
}

// verifySignature verifies a transaction signature contained in SignatureData abstracting over different signing
// modes. It differs from verifySignature in that it uses the new txsigning.TxData interface in x/tx.
func verifySignature(
	ctx context.Context,
	pubKey cryptotypes.PubKey,
	signerData txsigning.SignerData,
	signatureData signing.SignatureData,
	handler *txsigning.HandlerMap,
	txData txsigning.TxData,
) error {
	switch data := signatureData.(type) {
	case *signing.SingleSignatureData:
		signMode, err := internalSignModeToAPI(data.SignMode)
		if err != nil {
			return err
		}
		signBytes, err := handler.GetSignBytes(ctx, signMode, signerData, txData)
		if err != nil {
			return err
		}

		// TODO - remove this once `interwoven-1` has been upgraded to initia@v1.1.0 via software upgrade
		// interwoven-1 eth pubkey verification has been changed to use EIP-191 prefix at block height 39652
		if sdkCtx := sdk.UnwrapSDKContext(ctx); sdkCtx.ChainID() == "interwoven-1" && sdkCtx.BlockHeight() < 39652 {
			if pubkey, ok := pubKey.(*ethsecp256k1.PubKey); ok {
				if !pubkey.VerifySignatureWithoutEIP191(signBytes, data.Signature) {
					return fmt.Errorf("unable to verify single signer signature")
				}

				return nil
			}
		}

		if !pubKey.VerifySignature(signBytes, data.Signature) {
			return fmt.Errorf("unable to verify single signer signature")
		}

		return nil

	case *signing.MultiSignatureData:
		multiPK, ok := pubKey.(multisig.PubKey)
		if !ok {
			return fmt.Errorf("expected %T, got %T", (multisig.PubKey)(nil), pubKey)
		}
		err := multiPK.VerifyMultisignature(func(mode signing.SignMode) ([]byte, error) {
			signMode, err := internalSignModeToAPI(mode)
			if err != nil {
				return nil, err
			}
			return handler.GetSignBytes(ctx, signMode, signerData, txData)
		}, data)
		if err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("unexpected SignatureData %T", signatureData)
	}
}
