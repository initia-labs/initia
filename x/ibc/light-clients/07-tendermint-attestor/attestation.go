package tendermintattestor

import (
	"bytes"
	"slices"

	sdked25519 "github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"

	errorsmod "cosmossdk.io/errors"
)

func (cs ClientState) VerifySignatures(
	_ sdk.Context,
	proofBytes []byte,
	attestations []*Attestation,
) error {
	if cs.Threshold == 0 {
		return nil
	} else if len(attestations) < int(cs.Threshold) {
		return errorsmod.Wrapf(ErrUnauthorizedAttestation, "not enough attestations: %d < %d", len(attestations), cs.Threshold)
	}

	seenPubKeys := make(map[string]struct{})
	for _, attestation := range attestations {
		if _, ok := seenPubKeys[string(attestation.PubKey)]; ok {
			return errorsmod.Wrapf(ErrUnauthorizedAttestation, "duplicate attestation public key: %s", string(attestation.PubKey))
		} else {
			seenPubKeys[string(attestation.PubKey)] = struct{}{}
		}

		attestorPubkeys := cs.AttestorPubkeys
		if !slices.ContainsFunc(attestorPubkeys, func(registeredPubkey []byte) bool {
			return bytes.Equal(attestation.PubKey, registeredPubkey)
		}) {
			return errorsmod.Wrapf(ErrUnauthorizedAttestation, "unauthorized attestation public key: %s", string(attestation.PubKey))
		}

		attestationPubKey := sdked25519.PubKey{Key: attestation.PubKey}
		if !attestationPubKey.VerifySignature(proofBytes, attestation.Signature) {
			return errorsmod.Wrap(ErrInvalidAttestation, "failed to verify attestation signature")
		}
	}
	return nil
}
