package tendermintattestor_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
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
				pubKeyAny1, err := codectypes.NewAnyWithValue(pubKey1)
				suite.Require().NoError(err)
				pubKeyAny2, err := codectypes.NewAnyWithValue(pubKey2)
				suite.Require().NoError(err)

				clientState.AttestorPubkeys = []*codectypes.Any{pubKeyAny1, pubKeyAny2}
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
					PubKey:    pubKeyAny1,
					Signature: sig1,
				}
				attestation2 := &ibctmattestor.Attestation{
					PubKey:    pubKeyAny2,
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

				pubKeyAny, err := codectypes.NewAnyWithValue(pubKey)
				suite.Require().NoError(err)

				clientState.AttestorPubkeys = []*codectypes.Any{pubKeyAny}
				clientState.Threshold = 2 // Require 2 but only provide 1

				proofBytes = []byte("test proof")
				sig, err := privKey.Sign(proofBytes)
				suite.Require().NoError(err)

				attestation := &ibctmattestor.Attestation{
					PubKey:    pubKeyAny,
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
				authorizedPubKeyAny, err := codectypes.NewAnyWithValue(authorizedPubKey)
				suite.Require().NoError(err)

				// Create unauthorized key
				unauthorizedPrivKey := ed25519.GenPrivKey()
				unauthorizedPubKey := unauthorizedPrivKey.PubKey()
				unauthorizedPubKeyAny, err := codectypes.NewAnyWithValue(unauthorizedPubKey)
				suite.Require().NoError(err)

				clientState.AttestorPubkeys = []*codectypes.Any{authorizedPubKeyAny}
				clientState.Threshold = 1

				proofBytes = []byte("test proof")
				sig, err := unauthorizedPrivKey.Sign(proofBytes)
				suite.Require().NoError(err)

				// Use unauthorized key for attestation
				attestation := &ibctmattestor.Attestation{
					PubKey:    unauthorizedPubKeyAny,
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

				pubKeyAny, err := codectypes.NewAnyWithValue(pubKey)
				suite.Require().NoError(err)

				clientState.AttestorPubkeys = []*codectypes.Any{pubKeyAny}
				clientState.Threshold = 1

				proofBytes = []byte("test proof")
				// Create invalid signature by signing different data
				invalidSig, err := privKey.Sign([]byte("different data"))
				suite.Require().NoError(err)

				attestation := &ibctmattestor.Attestation{
					PubKey:    pubKeyAny,
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

func (suite *TMAttestorTestSuite) TestAttestationGetPubKey() {
	testCases := []struct {
		name        string
		attestation *ibctmattestor.Attestation
		expPubKey   cryptotypes.PubKey
		expNil      bool
	}{
		{
			"success with ed25519 key",
			func() *ibctmattestor.Attestation {
				privKey := ed25519.GenPrivKey()
				pubKey := privKey.PubKey()
				pubKeyAny, err := codectypes.NewAnyWithValue(pubKey)
				suite.Require().NoError(err)

				return &ibctmattestor.Attestation{
					PubKey:    pubKeyAny,
					Signature: []byte("signature"),
				}
			}(),
			ed25519.GenPrivKey().PubKey(), // We'll check the type, not the exact key
			false,
		},
		{
			"success with secp256k1 key",
			func() *ibctmattestor.Attestation {
				privKey := secp256k1.GenPrivKey()
				pubKey := privKey.PubKey()
				pubKeyAny, err := codectypes.NewAnyWithValue(pubKey)
				suite.Require().NoError(err)

				return &ibctmattestor.Attestation{
					PubKey:    pubKeyAny,
					Signature: []byte("signature"),
				}
			}(),
			secp256k1.GenPrivKey().PubKey(),
			false,
		},
		{
			"failure with nil pub key any",
			&ibctmattestor.Attestation{
				PubKey:    nil,
				Signature: []byte("signature"),
			},
			nil,
			true,
		},
		{
			"failure with invalid cached value",
			func() *ibctmattestor.Attestation {
				// Create an Any with invalid cached value
				pubKeyAny := &codectypes.Any{
					TypeUrl: "/cosmos.crypto.ed25519.PubKey",
					Value:   []byte("invalid"),
				}
				// Don't set cached value properly

				return &ibctmattestor.Attestation{
					PubKey:    pubKeyAny,
					Signature: []byte("signature"),
				}
			}(),
			nil,
			true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			pubKey := tc.attestation.GetPubKey()

			if tc.expNil {
				suite.Require().Nil(pubKey)
			} else {
				suite.Require().NotNil(pubKey)
				// Check that the returned key is of the expected type
				suite.Require().IsType(tc.expPubKey, pubKey)
			}
		})
	}
}

func (suite *TMAttestorTestSuite) TestAttestationUnpackInterfaces() {
	testCases := []struct {
		name        string
		attestation *ibctmattestor.Attestation
		expPass     bool
	}{
		{
			"success with valid ed25519 pub key",
			func() *ibctmattestor.Attestation {
				privKey := ed25519.GenPrivKey()
				pubKey := privKey.PubKey()
				pubKeyAny, err := codectypes.NewAnyWithValue(pubKey)
				suite.Require().NoError(err)

				return &ibctmattestor.Attestation{
					PubKey:    pubKeyAny,
					Signature: []byte("signature"),
				}
			}(),
			true,
		},
		{
			"success with valid secp256k1 pub key",
			func() *ibctmattestor.Attestation {
				privKey := secp256k1.GenPrivKey()
				pubKey := privKey.PubKey()
				pubKeyAny, err := codectypes.NewAnyWithValue(pubKey)
				suite.Require().NoError(err)

				return &ibctmattestor.Attestation{
					PubKey:    pubKeyAny,
					Signature: []byte("signature"),
				}
			}(),
			true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			// Use the chain's codec for unpacking
			unpacker := suite.chainA.Codec

			err := tc.attestation.UnpackInterfaces(unpacker)

			if tc.expPass {
				suite.Require().NoError(err)
				// Verify that after unpacking, GetPubKey works
				pubKey := tc.attestation.GetPubKey()
				suite.Require().NotNil(pubKey)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *TMAttestorTestSuite) TestMerkleProofBytesWithAttestationsUnpackInterfaces() {
	testCases := []struct {
		name                             string
		merkleProofBytesWithAttestations *ibctmattestor.MerkleProofBytesWithAttestations
		expPass                          bool
	}{
		{
			"success with valid attestations",
			func() *ibctmattestor.MerkleProofBytesWithAttestations {
				// Create multiple valid attestations
				privKey1 := ed25519.GenPrivKey()
				pubKey1 := privKey1.PubKey()
				pubKeyAny1, err := codectypes.NewAnyWithValue(pubKey1)
				suite.Require().NoError(err)

				privKey2 := secp256k1.GenPrivKey()
				pubKey2 := privKey2.PubKey()
				pubKeyAny2, err := codectypes.NewAnyWithValue(pubKey2)
				suite.Require().NoError(err)

				attestation1 := &ibctmattestor.Attestation{
					PubKey:    pubKeyAny1,
					Signature: []byte("signature1"),
				}
				attestation2 := &ibctmattestor.Attestation{
					PubKey:    pubKeyAny2,
					Signature: []byte("signature2"),
				}

				return &ibctmattestor.MerkleProofBytesWithAttestations{
					ProofBytes:   []byte("proof"),
					Attestations: []*ibctmattestor.Attestation{attestation1, attestation2},
				}
			}(),
			true,
		},
		{
			"success with empty attestations",
			&ibctmattestor.MerkleProofBytesWithAttestations{
				ProofBytes:   []byte("proof"),
				Attestations: []*ibctmattestor.Attestation{},
			},
			true,
		},
	}

	for _, tc := range testCases {
		tc := tc
		suite.Run(tc.name, func() {
			// Use the chain's codec for unpacking
			unpacker := suite.chainA.Codec

			err := tc.merkleProofBytesWithAttestations.UnpackInterfaces(unpacker)

			if tc.expPass {
				suite.Require().NoError(err)
				// Verify that after unpacking, GetPubKey works for all attestations
				for _, attestation := range tc.merkleProofBytesWithAttestations.Attestations {
					if attestation.PubKey != nil {
						pubKey := attestation.GetPubKey()
						suite.Require().NotNil(pubKey)
					}
				}
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func TestAttestationUnitTests(t *testing.T) {
	// Unit tests that don't require the full test suite setup

	t.Run("Attestation GetPubKey with nil Any", func(t *testing.T) {
		attestation := &ibctmattestor.Attestation{
			PubKey:    nil,
			Signature: []byte("signature"),
		}

		pubKey := attestation.GetPubKey()
		require.Nil(t, pubKey)
	})

	t.Run("MerkleProofBytesWithAttestations UnpackInterfaces with nil attestations", func(t *testing.T) {
		proof := &ibctmattestor.MerkleProofBytesWithAttestations{
			ProofBytes:   []byte("proof"),
			Attestations: nil,
		}

		// Create a mock unpacker - since attestations is nil, this should not be called
		err := proof.UnpackInterfaces(nil)
		require.NoError(t, err)
	})
}
