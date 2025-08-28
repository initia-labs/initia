package tendermintattestor

import (
	"testing"

	"github.com/stretchr/testify/require"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	clienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	commitmenttypes "github.com/cosmos/ibc-go/v8/modules/core/23-commitment/types"
	tmlightclient "github.com/cosmos/ibc-go/v8/modules/light-clients/07-tendermint"
)

func Test_hasSameAttestorsAndThreshold(t *testing.T) {
	// Generate test public keys
	pubkey1 := secp256k1.GenPrivKey().PubKey()
	pubkey2 := secp256k1.GenPrivKey().PubKey()
	pubkey3 := secp256k1.GenPrivKey().PubKey()
	pubkey4 := secp256k1.GenPrivKey().PubKey()

	// Helper function to create attestor pubkeys
	createAttestorPubkeys := func(pubkeys ...cryptotypes.PubKey) []*codectypes.Any {
		result := make([]*codectypes.Any, len(pubkeys))
		for i, pk := range pubkeys {
			any, err := codectypes.NewAnyWithValue(pk)
			require.NoError(t, err)
			result[i] = any
		}
		return result
	}

	// Helper function to create client state
	createClientState := func(attestorPubkeys []*codectypes.Any, threshold uint32) ClientState {
		return ClientState{
			ClientState: &tmlightclient.ClientState{
				ChainId:         "test-chain",
				TrustLevel:      tmlightclient.Fraction{Numerator: 1, Denominator: 3},
				TrustingPeriod:  86400,
				UnbondingPeriod: 172800,
				MaxClockDrift:   1000,
				LatestHeight:    clienttypes.Height{RevisionNumber: 1, RevisionHeight: 100},
				ProofSpecs:      commitmenttypes.GetSDKSpecs(),
				UpgradePath:     []string{"upgrade", "upgradedClient"},
			},
			AttestorPubkeys: attestorPubkeys,
			Threshold:       threshold,
		}
	}

	tests := []struct {
		name     string
		cs1      ClientState
		cs2      ClientState
		expected bool
	}{
		{
			name: "same attestors and threshold",
			cs1: createClientState(
				createAttestorPubkeys(pubkey1, pubkey2, pubkey3),
				2,
			),
			cs2: createClientState(
				createAttestorPubkeys(pubkey1, pubkey2, pubkey3),
				2,
			),
			expected: true,
		},
		{
			name: "same attestors in different order and threshold",
			cs1: createClientState(
				createAttestorPubkeys(pubkey1, pubkey2, pubkey3),
				2,
			),
			cs2: createClientState(
				createAttestorPubkeys(pubkey3, pubkey1, pubkey2),
				2,
			),
			expected: true,
		},
		{
			name: "different threshold",
			cs1: createClientState(
				createAttestorPubkeys(pubkey1, pubkey2, pubkey3),
				2,
			),
			cs2: createClientState(
				createAttestorPubkeys(pubkey1, pubkey2, pubkey3),
				3,
			),
			expected: false,
		},
		{
			name: "different number of attestors",
			cs1: createClientState(
				createAttestorPubkeys(pubkey1, pubkey2, pubkey3),
				2,
			),
			cs2: createClientState(
				createAttestorPubkeys(pubkey1, pubkey2),
				2,
			),
			expected: false,
		},
		{
			name: "different attestor sets",
			cs1: createClientState(
				createAttestorPubkeys(pubkey1, pubkey2, pubkey3),
				2,
			),
			cs2: createClientState(
				createAttestorPubkeys(pubkey1, pubkey2, pubkey4),
				2,
			),
			expected: false,
		},
		{
			name: "duplicate attestors in first set",
			cs1: createClientState(
				createAttestorPubkeys(pubkey1, pubkey1, pubkey2),
				2,
			),
			cs2: createClientState(
				createAttestorPubkeys(pubkey1, pubkey2, pubkey3),
				2,
			),
			expected: false,
		},
		{
			name: "duplicate attestors in second set",
			cs1: createClientState(
				createAttestorPubkeys(pubkey1, pubkey2, pubkey3),
				2,
			),
			cs2: createClientState(
				createAttestorPubkeys(pubkey1, pubkey2, pubkey2),
				2,
			),
			expected: false,
		},
		{
			name: "empty attestor sets with same threshold",
			cs1: createClientState(
				[]*codectypes.Any{},
				0,
			),
			cs2: createClientState(
				[]*codectypes.Any{},
				0,
			),
			expected: true,
		},
		{
			name: "empty attestor sets with different threshold",
			cs1: createClientState(
				[]*codectypes.Any{},
				0,
			),
			cs2: createClientState(
				[]*codectypes.Any{},
				1,
			),
			expected: false,
		},
		{
			name: "single attestor with same threshold",
			cs1: createClientState(
				createAttestorPubkeys(pubkey1),
				1,
			),
			cs2: createClientState(
				createAttestorPubkeys(pubkey1),
				1,
			),
			expected: true,
		},
		{
			name: "single attestor with different threshold",
			cs1: createClientState(
				createAttestorPubkeys(pubkey1),
				1,
			),
			cs2: createClientState(
				createAttestorPubkeys(pubkey1),
				2,
			),
			expected: false,
		},
		{
			name: "same attestors but different counts (duplicates)",
			cs1: createClientState(
				createAttestorPubkeys(pubkey1, pubkey2, pubkey3),
				2,
			),
			cs2: createClientState(
				createAttestorPubkeys(pubkey1, pubkey2, pubkey3, pubkey3),
				2,
			),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cs1.hasSameAttestorsAndThreshold(tt.cs2)
			require.Equal(t, tt.expected, result, "test case: %s", tt.name)
		})
	}
}

// Test_hasSameAttestorsAndThreshold_EdgeCases tests edge cases and boundary conditions
func Test_hasSameAttestorsAndThreshold_EdgeCases(t *testing.T) {
	// Generate test public keys
	pubkey1 := secp256k1.GenPrivKey().PubKey()
	pubkey2 := secp256k1.GenPrivKey().PubKey()

	// Helper function to create attestor pubkeys
	createAttestorPubkeys := func(pubkeys ...cryptotypes.PubKey) []*codectypes.Any {
		result := make([]*codectypes.Any, len(pubkeys))
		for i, pk := range pubkeys {
			any, err := codectypes.NewAnyWithValue(pk)
			require.NoError(t, err)
			result[i] = any
		}
		return result
	}

	// Helper function to create client state
	createClientState := func(attestorPubkeys []*codectypes.Any, threshold uint32) ClientState {
		return ClientState{
			ClientState: &tmlightclient.ClientState{
				ChainId:         "test-chain",
				TrustLevel:      tmlightclient.Fraction{Numerator: 1, Denominator: 3},
				TrustingPeriod:  86400,
				UnbondingPeriod: 172800,
				MaxClockDrift:   1000,
				LatestHeight:    clienttypes.Height{RevisionNumber: 1, RevisionHeight: 100},
				ProofSpecs:      commitmenttypes.GetSDKSpecs(),
				UpgradePath:     []string{"upgrade", "upgradedClient"},
			},
			AttestorPubkeys: attestorPubkeys,
			Threshold:       threshold,
		}
	}

	tests := []struct {
		name     string
		cs1      ClientState
		cs2      ClientState
		expected bool
	}{
		{
			name: "zero threshold with empty attestors",
			cs1: createClientState(
				[]*codectypes.Any{},
				0,
			),
			cs2: createClientState(
				[]*codectypes.Any{},
				0,
			),
			expected: true,
		},
		{
			name: "zero threshold with non-empty attestors",
			cs1: createClientState(
				createAttestorPubkeys(pubkey1),
				0,
			),
			cs2: createClientState(
				createAttestorPubkeys(pubkey1),
				0,
			),
			expected: true,
		},
		{
			name: "threshold equals attestor count",
			cs1: createClientState(
				createAttestorPubkeys(pubkey1, pubkey2),
				2,
			),
			cs2: createClientState(
				createAttestorPubkeys(pubkey1, pubkey2),
				2,
			),
			expected: true,
		},
		{
			name: "threshold greater than attestor count",
			cs1: createClientState(
				createAttestorPubkeys(pubkey1),
				2,
			),
			cs2: createClientState(
				createAttestorPubkeys(pubkey1),
				2,
			),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.cs1.hasSameAttestorsAndThreshold(tt.cs2)
			require.Equal(t, tt.expected, result, "test case: %s", tt.name)
		})
	}
}
