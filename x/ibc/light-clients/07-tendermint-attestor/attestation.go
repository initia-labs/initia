package tendermintattestor

import (
	"slices"

	sdk "github.com/cosmos/cosmos-sdk/types"

	errorsmod "cosmossdk.io/errors"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
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

	seenPubKeys := make([]*cryptotypes.PubKey, 0, len(attestations))
	for _, attestation := range attestations {
		attestationPubKey := attestation.GetPubKey()
		if slices.Contains(seenPubKeys, &attestationPubKey) {
			return errorsmod.Wrapf(ErrUnauthorizedAttestation, "duplicate attestation public key: %s", attestationPubKey.String())
		} else {
			seenPubKeys = append(seenPubKeys, &attestationPubKey)
		}

		attestorPubkeys, err := cs.GetAttestorPubkeys()
		if err != nil {
			return err
		}
		if !slices.ContainsFunc(attestorPubkeys, func(registeredPubkey cryptotypes.PubKey) bool {
			return attestationPubKey.Equals(registeredPubkey)
		}) {
			return errorsmod.Wrapf(ErrUnauthorizedAttestation, "unauthorized attestation public key: %s", attestationPubKey.String())
		}

		if !attestationPubKey.VerifySignature(proofBytes, attestation.Signature) {
			return errorsmod.Wrap(ErrInvalidAttestation, "failed to verify attestation signature")
		}
	}
	return nil
}

// UnpackInterfaces implements UnpackInterfacesMessage.UnpackInterfaces
func (m *MerkleProofBytesWithAttestations) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	if m == nil {
		return nil
	}

	for i := range m.Attestations {
		err := m.Attestations[i].UnpackInterfaces(unpacker)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetPubKey returns the public key from the attestation.
func (at Attestation) GetPubKey() (pk cryptotypes.PubKey) {
	if at.PubKey == nil {
		return nil
	}
	content, ok := at.PubKey.GetCachedValue().(cryptotypes.PubKey)
	if !ok {
		return nil
	}
	return content
}

// UnpackInterfaces implements UnpackInterfacesMessage.UnpackInterfaces
func (at *Attestation) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	var pubKey cryptotypes.PubKey
	return unpacker.UnpackAny(at.PubKey, &pubKey)
}
