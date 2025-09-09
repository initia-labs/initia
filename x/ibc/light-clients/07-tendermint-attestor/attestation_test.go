package tendermintattestor_test

import (
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"

	ibctmattestor "github.com/initia-labs/initia/x/ibc/light-clients/07-tendermint-attestor"
)

func (suite *TMAttestorTestSuite) TestVerifySignatures() {
	var (
		clientState  *ibctmattestor.ClientState
		proofBytes   []byte
		attestations []*ibctmattestor.Attestation
	)

	testCases := []struct {
		name     string
		malleate func()
		expPass  bool
		expError error
	}{
		{
			"success with threshold 0",
			func() {
				clientState.Threshold = 0
				attestations = []*ibctmattestor.Attestation{}
			},
			true,
			nil,
		},
		{
			"success with valid attestations",
			func() {
				// Create test keys
				privKey1 := ed25519.GenPrivKey()
				privKey2 := ed25519.GenPrivKey()
				pubKey1 := privKey1.PubKey()
				pubKey2 := privKey2.PubKey()

				// Set up client state with attestor public keys
				pubKey1Bytes := pubKey1.Bytes()
				pubKey2Bytes := pubKey2.Bytes()

				clientState.AttestorPubkeys = [][]byte{pubKey1.Bytes(), pubKey2.Bytes()}
				clientState.Threshold = 2

				// Create proof bytes to sign
				proofBytes = []byte("test proof")

				// Create valid signatures
				sig1, err := privKey1.Sign(proofBytes)
				suite.Require().NoError(err)
				sig2, err := privKey2.Sign(proofBytes)
				suite.Require().NoError(err)

				// Create attestations
				attestation1 := &ibctmattestor.Attestation{
					PubKey:    pubKey1Bytes,
					Signature: sig1,
				}
				attestation2 := &ibctmattestor.Attestation{
					PubKey:    pubKey2Bytes,
					Signature: sig2,
				}

				attestations = []*ibctmattestor.Attestation{attestation1, attestation2}
			},
			true,
			nil,
		},
		{
			"failure: not enough attestations",
			func() {
				// Create test key
				privKey := ed25519.GenPrivKey()
				pubKey := privKey.PubKey()

				pubKeyBytes := pubKey.Bytes()

				clientState.AttestorPubkeys = [][]byte{pubKey.Bytes()}
				clientState.Threshold = 2 // Require 2 but only provide 1

				proofBytes = []byte("test proof")
				sig, err := privKey.Sign(proofBytes)
				suite.Require().NoError(err)

				attestation := &ibctmattestor.Attestation{
					PubKey:    pubKeyBytes,
					Signature: sig,
				}
				attestations = []*ibctmattestor.Attestation{attestation}
			},
			false,
			ibctmattestor.ErrUnauthorizedAttestation,
		},
		{
			"failure: unauthorized attestor",
			func() {
				// Create authorized key
				authorizedPrivKey := ed25519.GenPrivKey()
				authorizedPubKey := authorizedPrivKey.PubKey()
				authorizedPubKeyBytes := authorizedPubKey.Bytes()

				// Create unauthorized key
				unauthorizedPrivKey := ed25519.GenPrivKey()
				unauthorizedPubKey := unauthorizedPrivKey.PubKey()
				unauthorizedPubKeyBytes := unauthorizedPubKey.Bytes()

				clientState.AttestorPubkeys = [][]byte{authorizedPubKeyBytes}
				clientState.Threshold = 1

				proofBytes = []byte("test proof")
				sig, err := unauthorizedPrivKey.Sign(proofBytes)
				suite.Require().NoError(err)

				// Use unauthorized key for attestation
				attestation := &ibctmattestor.Attestation{
					PubKey:    unauthorizedPubKeyBytes,
					Signature: sig,
				}
				attestations = []*ibctmattestor.Attestation{attestation}
			},
			false,
			ibctmattestor.ErrUnauthorizedAttestation,
		},
		{
			"failure: invalid signature",
			func() {
				privKey := ed25519.GenPrivKey()
				pubKey := privKey.PubKey()

				pubKeyBytes := pubKey.Bytes()

				clientState.AttestorPubkeys = [][]byte{pubKey.Bytes()}
				clientState.Threshold = 1

				proofBytes = []byte("test proof")
				// Create invalid signature by signing different data
				invalidSig, err := privKey.Sign([]byte("different data"))
				suite.Require().NoError(err)

				attestation := &ibctmattestor.Attestation{
					PubKey:    pubKeyBytes,
					Signature: invalidSig,
				}
				attestations = []*ibctmattestor.Attestation{attestation}
			},
			false,
			ibctmattestor.ErrInvalidAttestation,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			suite.SetupTest() // reset

			// Create a basic client state
			clientState = &ibctmattestor.ClientState{}
			proofBytes = []byte("default proof")
			attestations = []*ibctmattestor.Attestation{}

			tc.malleate()

			ctx := sdk.Context{}
			err := clientState.VerifySignatures(ctx, proofBytes, attestations)

			if tc.expPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
				if tc.expError != nil {
					suite.Require().ErrorIs(err, tc.expError)
				}
			}
		})
	}
}
