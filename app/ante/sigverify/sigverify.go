package sigverify

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"golang.org/x/crypto/sha3"

	"google.golang.org/protobuf/types/known/anypb"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	txsigning "cosmossdk.io/x/tx/signing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256r1"
	"github.com/cosmos/cosmos-sdk/crypto/types/multisig"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/initia-labs/initia/crypto/ethsecp256k1"

	forwardingtypes "github.com/noble-assets/forwarding/v2/types"

	initiatx "github.com/initia-labs/initia/tx"
	vmtypes "github.com/initia-labs/movevm/types"
)

var ZeroPubKey = secp256k1.GenPrivKeyFromSecret(bytes.Repeat([]byte{0x00}, 32)).PubKey()

type MoveKeeper interface {
	VerifyAccountAbstractionSignature(ctx context.Context, sender string, abstractionData vmtypes.AbstractionData) (string, error)
}

// SigVerificationDecorator verifies all signatures for a tx and return an error if any are invalid. Note,
// the SigVerificationDecorator will not check signatures on ReCheck.
//
// CONTRACT: Pubkeys are set in context for all signers before this decorator runs
// CONTRACT: Tx must implement SigVerifiableTx interface
type SigVerificationDecorator struct {
	ak              authante.AccountKeeper
	signModeHandler *txsigning.HandlerMap
	moveKeeper      MoveKeeper
}

func NewSigVerificationDecorator(ak authante.AccountKeeper, signModeHandler *txsigning.HandlerMap, moveKeeper MoveKeeper) SigVerificationDecorator {
	return SigVerificationDecorator{
		ak:              ak,
		signModeHandler: signModeHandler,
		moveKeeper:      moveKeeper,
	}
}

func (svd SigVerificationDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	sigTx, ok := tx.(authsigning.Tx)
	if !ok {
		return ctx, errorsmod.Wrap(sdkerrors.ErrTxDecode, "invalid transaction type")
	}

	// stdSigs contains the sequence number, account number, and signatures.
	// When simulating, this would just be a 0-length slice.
	sigs, err := sigTx.GetSignaturesV2()
	if err != nil {
		return ctx, err
	}

	signers, err := sigTx.GetSigners()
	if err != nil {
		return ctx, err
	}

	// check that signer length and signature length are the same
	if len(sigs) != len(signers) {
		return ctx, errorsmod.Wrapf(sdkerrors.ErrUnauthorized, "invalid number of signer;  expected: %d, got %d", len(signers), len(sigs))
	}

	for i, sig := range sigs {
		acc, err := authante.GetSignerAcc(ctx, svd.ak, signers[i])
		if err != nil {
			return ctx, err
		}

		// retrieve pubkey
		pubKey := acc.GetPubKey()
		if !simulate && pubKey == nil {
			return ctx, errorsmod.Wrap(sdkerrors.ErrInvalidPubKey, "pubkey on account is not set")
		}

		// Check account sequence number.
		if sig.Sequence != acc.GetSequence() {
			return ctx, errorsmod.Wrapf(
				sdkerrors.ErrWrongSequence,
				"account sequence mismatch, expected %d, got %d", acc.GetSequence(), sig.Sequence,
			)
		}

		// retrieve signer data
		genesis := ctx.BlockHeight() == 0
		chainID := ctx.ChainID()
		var accNum uint64
		if !genesis {
			accNum = acc.GetAccountNumber()
		}

		// no need to verify signatures on recheck tx
		if !simulate && !ctx.IsReCheckTx() {
			anyPk, _ := codectypes.NewAnyWithValue(pubKey)

			signerData := txsigning.SignerData{
				Address:       acc.GetAddress().String(),
				ChainID:       chainID,
				AccountNumber: accNum,
				Sequence:      acc.GetSequence(),
				PubKey: &anypb.Any{
					TypeUrl: anyPk.TypeUrl,
					Value:   anyPk.Value,
				},
			}
			adaptableTx, ok := tx.(authsigning.V2AdaptableTx)
			if !ok {
				return ctx, fmt.Errorf("expected tx to implement V2AdaptableTx, got %T", tx)
			}
			txData := adaptableTx.GetSigningTxData()

			if data, ok := sig.Data.(*signing.SingleSignatureData); ok && data.SignMode == initiatx.Signing_SignMode_ACCOUNT_ABSTRACTION {
				abstractionData := vmtypes.AbstractionData{}
				err = json.Unmarshal(data.Signature, &abstractionData)
				if err != nil {
					return ctx, err
				}
				if err := abstractionData.Validate(); err != nil {
					return ctx, err
				}

				signBytes, err := svd.signModeHandler.GetSignBytes(ctx, initiatx.Signingv1beta1_SignMode_ACCOUNT_ABSTRACTION, signerData, txData)
				if err != nil {
					return ctx, err
				}

				digest := sha3.Sum256(signBytes)
				digestBytes := digest[:]
				expectedDigest := abstractionData.SigningMessageDigest()
				if !bytes.Equal(digestBytes, expectedDigest) {
					return ctx, fmt.Errorf("signing message digest mismatch: expected %x, got %x", expectedDigest, digestBytes)
				}

				_, err = svd.moveKeeper.VerifyAccountAbstractionSignature(ctx, signerData.Address, abstractionData)
				if err != nil {
					return ctx, fmt.Errorf("failed to verify account abstraction signature: %w", err)
				}
			} else {
				err = verifySignature(ctx, pubKey, signerData, sig.Data, svd.signModeHandler, txData)
				if err != nil {
					var errMsg string
					if authante.OnlyLegacyAminoSigners(sig.Data) {
						// If all signers are using SIGN_MODE_LEGACY_AMINO, we rely on VerifySignature to check account sequence number,
						// and therefore communicate sequence number as a potential cause of error.
						errMsg = fmt.Sprintf("signature verification failed; please verify account number (%d), sequence (%d) and chain-id (%s)", accNum, acc.GetSequence(), chainID)
					} else {
						errMsg = fmt.Sprintf("signature verification failed; please verify account number (%d) and chain-id (%s): (%s)", accNum, chainID, err.Error())
					}
					return ctx, errorsmod.Wrap(sdkerrors.ErrUnauthorized, errMsg)
				}
			}
		}
	}

	if next != nil {
		return next(ctx, tx, simulate)
	}

	return ctx, nil
}

// DefaultSigVerificationGasConsumer is the default implementation of SignatureVerificationGasConsumer. It consumes gas
// for signature verification based upon the public key type. The cost is fetched from the given params and is matched
// by the concrete type.
func DefaultSigVerificationGasConsumer(
	meter storetypes.GasMeter, sig signing.SignatureV2, params types.Params,
) error {
	pubkey := sig.PubKey
	switch pubkey := pubkey.(type) {
	case *ed25519.PubKey:
		meter.ConsumeGas(params.SigVerifyCostED25519, "ante verify: ed25519")
		return errorsmod.Wrap(sdkerrors.ErrInvalidPubKey, "ED25519 public keys are unsupported")

	case *secp256k1.PubKey, *ethsecp256k1.PubKey:
		meter.ConsumeGas(params.SigVerifyCostSecp256k1, "ante verify: secp256k1")
		return nil

	case *secp256r1.PubKey:
		meter.ConsumeGas(params.SigVerifyCostSecp256r1(), "ante verify: secp256r1")
		return nil

	case multisig.PubKey:
		multisignature, ok := sig.Data.(*signing.MultiSignatureData)
		if !ok {
			return fmt.Errorf("expected %T, got, %T", &signing.MultiSignatureData{}, sig.Data)
		}
		err := consumeMultiSignatureVerificationGas(meter, multisignature, pubkey, params, sig.Sequence)
		if err != nil {
			return err
		}
		return nil

	case *forwardingtypes.ForwardingPubKey:
		return nil

	default:
		return errorsmod.Wrapf(sdkerrors.ErrInvalidPubKey, "unrecognized public key type: %T", pubkey)
	}
}

// consumeMultiSignatureVerificationGas consumes gas from a GasMeter for verifying a multisig pubkey signature
func consumeMultiSignatureVerificationGas(
	meter storetypes.GasMeter, sig *signing.MultiSignatureData, pubkey multisig.PubKey,
	params types.Params, accSeq uint64,
) error {
	size := sig.BitArray.Count()
	sigIndex := 0

	for i := 0; i < size; i++ {
		if !sig.BitArray.GetIndex(i) {
			continue
		}
		sigV2 := signing.SignatureV2{
			PubKey:   pubkey.GetPubKeys()[i],
			Data:     sig.Signatures[sigIndex],
			Sequence: accSeq,
		}
		err := DefaultSigVerificationGasConsumer(meter, sigV2, params)
		if err != nil {
			return err
		}
		sigIndex++
	}

	return nil
}
