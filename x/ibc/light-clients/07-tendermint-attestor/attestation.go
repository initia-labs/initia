package tendermintattestor

import (
	"slices"

	sdk "github.com/cosmos/cosmos-sdk/types"

	errorsmod "cosmossdk.io/errors"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
)

func (cs ClientState) VerifySignatures(
	ctx sdk.Context,
	proofBytes []byte,
	attestations []*Attestation,
) error {
	if cs.Threshold == 0 {
		return nil
	} else if len(attestations) < int(cs.Threshold) {
		return errorsmod.Wrapf(ErrUnauthorizedAttestation, "not enough attestations: %d < %d", len(attestations), cs.Threshold)
	}
	for _, attestation := range attestations {
		var attestationPubKey cryptotypes.PubKey
		err := PubkeyCdc.UnpackAny(&attestation.PubKey, &attestationPubKey)
		if err != nil {
			return err
		}

		if !slices.ContainsFunc(cs.AttestorPubkeys, func(registeredPubkeyAny codectypes.Any) bool {
			var registeredPubkey cryptotypes.PubKey
			err := PubkeyCdc.UnpackAny(&registeredPubkeyAny, &registeredPubkey)
			if err != nil {
				return false
			}

			return attestationPubKey.Equals(registeredPubkey)
		}) {
			return ErrUnauthorizedAttestation
		}

		if !attestationPubKey.VerifySignature(proofBytes, attestation.Signature) {
			return errorsmod.Wrap(ErrInvalidAttestation, err.Error())
		}
	}
	return nil
}
