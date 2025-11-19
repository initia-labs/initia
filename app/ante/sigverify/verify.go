package sigverify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"golang.org/x/crypto/sha3"

	signingv1beta1 "cosmossdk.io/api/cosmos/tx/signing/v1beta1"
	txsigning "cosmossdk.io/x/tx/signing"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/crypto/types/multisig"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"

	"github.com/initia-labs/initia/crypto/derivable"
	initiatx "github.com/initia-labs/initia/tx"
	vmtypes "github.com/initia-labs/movevm/types"
)

// internalSignModeToAPI converts a signing.SignMode to a protobuf SignMode.
func InternalSignModeToAPI(mode signing.SignMode) (signingv1beta1.SignMode, error) {
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
	case initiatx.Signing_SignMode_ACCOUNT_ABSTRACTION:
		return initiatx.Signingv1beta1_SignMode_ACCOUNT_ABSTRACTION, nil
	default:
		return signingv1beta1.SignMode_SIGN_MODE_UNSPECIFIED, fmt.Errorf("unsupported sign mode %s", mode)
	}
}

// verifySignature verifies a transaction signature contained in SignatureData abstracting over different signing
// modes. It differs from verifySignature in that it uses the new txsigning.TxData interface in x/tx.
func verifySignature(
	ctx context.Context,
	accountAbstractionKeeper AccountAbstractionKeeper,
	pubKey cryptotypes.PubKey,
	signerData txsigning.SignerData,
	signatureData signing.SignatureData,
	handler *txsigning.HandlerMap,
	txData txsigning.TxData,
) error {
	switch data := signatureData.(type) {
	case *signing.SingleSignatureData:
		signMode, err := InternalSignModeToAPI(data.SignMode)
		if err != nil {
			return err
		}
		signBytes, err := handler.GetSignBytes(ctx, signMode, signerData, txData)
		if err != nil {
			return err
		}

		// conduct account abstraction signature verification
		if accountAbstractionKeeper != nil && data.SignMode == initiatx.Signing_SignMode_ACCOUNT_ABSTRACTION {
			abstractionData := vmtypes.AbstractionData{}
			err = json.Unmarshal(data.Signature, &abstractionData)
			if err != nil {
				return err
			}

			if err := abstractionData.Validate(); err != nil {
				return err
			}

			// check if the given pubkey is derivable to prevent address collision
			if _, ok := abstractionData.AuthData.(*vmtypes.AbstractionAuthData__DerivableV1); ok {
				if _, ok := pubKey.(*derivable.PubKey); !ok {
					return fmt.Errorf("non-derivable pubkey account is not allowed to use derivable authentication")
				}
			}

			digest := sha3.Sum256(signBytes)
			digestBytes := digest[:]
			digestInSignature := abstractionData.SigningMessageDigest()
			if !bytes.Equal(digestBytes, digestInSignature) {
				return fmt.Errorf("signing message digest mismatch: expected %x (calculated from sign bytes), got %x (from signature)", digestBytes, digestInSignature)
			}

			_, err = accountAbstractionKeeper.VerifyAccountAbstractionSignature(ctx, signerData.Address, abstractionData)
			if err != nil {
				return fmt.Errorf("failed to verify account abstraction signature: %w", err)
			}

			return nil
		}

		// conduct normal signature verification
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
			signMode, err := InternalSignModeToAPI(mode)
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
