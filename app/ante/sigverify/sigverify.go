package sigverify

import (
	"encoding/hex"
	"fmt"

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
)

// Simulation signature values used to estimate gas consumption.
var (
	key                = make([]byte, secp256k1.PubKeySize)
	simSecp256k1Pubkey = &secp256k1.PubKey{Key: key}
)

func init() {
	// Decode a valid hex string into a secp256k1 public key for transaction simulation.
	bz, _ := hex.DecodeString("035AD6810A47F073553FF30D2FCC7E0D3B1C0B74B61A1AAA2582344037151E143A")
	copy(key, bz)
	simSecp256k1Pubkey.Key = key
}

// SigVerificationDecorator verifies all signatures in a transaction and ensures their validity.
// Note: This decorator skips signature verification during ReCheckTx.
type SigVerificationDecorator struct {
	accountKeeper    authante.AccountKeeper
	signModeHandler *txsigning.HandlerMap
}

// NewSigVerificationDecorator creates a new SigVerificationDecorator.
func NewSigVerificationDecorator(accountKeeper authante.AccountKeeper, signModeHandler *txsigning.HandlerMap) SigVerificationDecorator {
	return SigVerificationDecorator{
		accountKeeper:    accountKeeper,
		signModeHandler: signModeHandler,
	}
}

// AnteHandle verifies signatures and enforces sequence numbers in transactions.
func (svd SigVerificationDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	sigTx, ok := tx.(authsigning.Tx)
	if !ok {
		return ctx, errorsmod.Wrap(sdkerrors.ErrTxDecode, "invalid transaction type")
	}

	// Get signatures and signers from the transaction.
	signatures, err := sigTx.GetSignaturesV2()
	if err != nil {
		return ctx, err
	}
	signers, err := sigTx.GetSigners()
	if err != nil {
		return ctx, err
	}

	// Ensure the number of signatures matches the number of signers.
	if len(signatures) != len(signers) {
		return ctx, errorsmod.Wrapf(sdkerrors.ErrUnauthorized, "mismatch in number of signers and signatures: expected %d, got %d", len(signers), len(signatures))
	}

	for i, signature := range signatures {
		if err := svd.verifySignature(ctx, signers[i], signature, tx, simulate); err != nil {
			return ctx, err
		}
	}

	return next(ctx, tx, simulate)
}

// verifySignature performs validation on a single signature.
func (svd SigVerificationDecorator) verifySignature(ctx sdk.Context, signer sdk.AccAddress, signature signing.SignatureV2, tx sdk.Tx, simulate bool) error {
	// Get the signer account.
	account, err := authante.GetSignerAcc(ctx, svd.accountKeeper, signer)
	if err != nil {
		return err
	}

	pubKey := account.GetPubKey()
	if !simulate && pubKey == nil {
		return errorsmod.Wrap(sdkerrors.ErrInvalidPubKey, "account pubkey is not set")
	}

	// Check sequence number.
	if signature.Sequence != account.GetSequence() {
		return errorsmod.Wrapf(sdkerrors.ErrWrongSequence, "sequence mismatch: expected %d, got %d", account.GetSequence(), signature.Sequence)
	}

	// Skip signature verification for ReCheckTx.
	if !simulate && !ctx.IsReCheckTx() {
		return svd.performSignatureVerification(ctx, account, pubKey, signature, tx)
	}

	return nil
}

// performSignatureVerification verifies the signature using the provided pubkey and signer data.
func (svd SigVerificationDecorator) performSignatureVerification(ctx sdk.Context, account types.AccountI, pubKey sdk.PubKey, signature signing.SignatureV2, tx sdk.Tx) error {
	genesis := ctx.BlockHeight() == 0
	chainID := ctx.ChainID()
	accountNumber := account.GetAccountNumber()
	if genesis {
		accountNumber = 0
	}

	anyPk, _ := codectypes.NewAnyWithValue(pubKey)
	signerData := txsigning.SignerData{
		Address:       account.GetAddress().String(),
		ChainID:       chainID,
		AccountNumber: accountNumber,
		Sequence:      account.GetSequence(),
		PubKey: &anypb.Any{
			TypeUrl: anyPk.TypeUrl,
			Value:   anyPk.Value,
		},
	}

	adaptableTx, ok := tx.(authsigning.V2AdaptableTx)
	if !ok {
		return fmt.Errorf("expected tx to implement V2AdaptableTx, got %T", tx)
	}

	txData := adaptableTx.GetSigningTxData()
	err := verifySignature(ctx, pubKey, signerData, signature.Data, svd.signModeHandler, txData)
	if err != nil {
		return wrapSignatureError(err, signerData, account.GetSequence())
	}

	return nil
}

// wrapSignatureError provides a detailed error message for signature verification failures.
func wrapSignatureError(err error, signerData txsigning.SignerData, sequence uint64) error {
	return errorsmod.Wrap(sdkerrors.ErrUnauthorized, fmt.Sprintf(
		"signature verification failed: verify account number (%d), sequence (%d), and chain-id (%s): %s",
		signerData.AccountNumber, sequence, signerData.ChainID, err.Error(),
	))
}

// DefaultSigVerificationGasConsumer calculates gas consumption based on the public key type.
func DefaultSigVerificationGasConsumer(
	meter storetypes.GasMeter, sig signing.SignatureV2, params types.Params,
) error {
	switch pubkey := sig.PubKey.(type) {
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
		return consumeMultisignatureVerificationGas(meter, sig.Data.(*signing.MultiSignatureData), pubkey, params, sig.Sequence)

	default:
		return errorsmod.Wrapf(sdkerrors.ErrInvalidPubKey, "unrecognized public key type: %T", pubkey)
	}
}

// consumeMultisignatureVerificationGas calculates gas for multisignature verification.
func consumeMultisignatureVerificationGas(
	meter storetypes.GasMeter, sig *signing.MultiSignatureData, pubkey multisig.PubKey,
	params types.Params, accSeq uint64,
) error {
	size := sig.BitArray.Count()
	for i, sigIndex := 0, 0; i < size; i++ {
		if !sig.BitArray.GetIndex(i) {
			continue
		}
		subSig := signing.SignatureV2{
			PubKey:   pubkey.GetPubKeys()[i],
			Data:     sig.Signatures[sigIndex],
			Sequence: accSeq,
		}
		if err := DefaultSigVerificationGasConsumer(meter, subSig, params); err != nil {
			return err
		}
		sigIndex++
	}
	return nil
}
