package keyring

import (
	"errors"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	cosmoskeyring "github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/ledger"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"

	"github.com/initia-labs/initia/crypto/ethsecp256k1"
	"github.com/initia-labs/initia/tx"
)

// unsafeExporter is implemented by key stores that support unsafe export
// of private keys' material.
type unsafeExporter interface {
	// ExportPrivateKeyObject returns a private key in unarmored format.
	ExportPrivateKeyObject(uid string) (cryptotypes.PrivKey, error)
}

var _ unsafeExporter = (*Keyring)(nil)

type Keyring struct {
	cosmoskeyring.Keyring
}

func NewKeyring(ctx client.Context, backend string) (*Keyring, error) {
	kr, err := cosmoskeyring.New(sdk.KeyringServiceName(), backend, ctx.KeyringDir, ctx.Input, ctx.Codec, ctx.KeyringOptions...)
	if err != nil {
		return nil, err
	}

	return &Keyring{Keyring: kr}, nil
}

func (ks Keyring) Sign(uid string, msg []byte, signMode signing.SignMode) ([]byte, cryptotypes.PubKey, error) {
	k, err := ks.Key(uid)
	if err != nil {
		return nil, nil, err
	}

	switch {
	case k.GetLedger() != nil:
		return SignWithLedger(k, msg, signMode)
	default:
		return ks.Keyring.Sign(uid, msg, signMode)
	}
}

// SignWithLedger signs a binary message with the ledger device referenced by an Info object
// and returns the signed bytes and the public key. It returns an error if the device could
// not be queried or it returned an error.
func SignWithLedger(k *cosmoskeyring.Record, msg []byte, signMode signing.SignMode) (sig []byte, pub cryptotypes.PubKey, err error) {
	pubKey, err := k.GetPubKey()
	if err != nil {
		return nil, nil, err
	}

	// validate flags
	_, isEthPubKey := pubKey.(*ethsecp256k1.PubKey)
	if isEthPubKey && signMode != signing.SignMode_SIGN_MODE_EIP_191 {
		return nil, nil, errors.New("must use --sign-mode=eip-191 for Ethereum Ledger")
	} else if !isEthPubKey && signMode != signing.SignMode_SIGN_MODE_LEGACY_AMINO_JSON {
		return nil, nil, errors.New("must use --sign-mode=amino-json for Cosmos Ledger")
	}

	signMsg := msg
	if isEthPubKey {
		// Remove EIP191 prefix from message since the Ledger device will
		// automatically add the prefix before signing. This avoids double-prefixing
		// which would result in an invalid signature.
		signMsg = tx.RemoveEIP191Prefix(msg)
	}

	ledgerInfo := k.GetLedger()
	if ledgerInfo == nil {
		return nil, nil, cosmoskeyring.ErrNotLedgerObj
	}

	path := ledgerInfo.GetPath()
	priv, err := ledger.NewPrivKeySecp256k1Unsafe(*path)
	if err != nil {
		return
	}
	ledgerPubKey := priv.PubKey()
	if !pubKey.Equals(ledgerPubKey) {
		return nil, nil, fmt.Errorf("the public key that the user attempted to sign with does not match the public key on the ledger device. %v does not match %v", pubKey.String(), ledgerPubKey.String())
	}

	sig, err = priv.SignLedgerAminoJSON(signMsg)
	if err != nil {
		return nil, nil, err
	}

	if !priv.PubKey().VerifySignature(msg, sig) {
		return nil, nil, cosmoskeyring.ErrLedgerInvalidSignature
	}

	return sig, priv.PubKey(), nil
}

// ExportPrivateKeyObject implements the unsafeExporter interface.
func (k *Keyring) ExportPrivateKeyObject(uid string) (cryptotypes.PrivKey, error) {
	return k.Keyring.(unsafeExporter).ExportPrivateKeyObject(uid)
}
