package secp256k1_test

import (
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/hex"
	"math/big"
	"testing"

	btcSecp256k1 "github.com/btcsuite/btcd/btcec/v2"
	btcecdsa "github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"github.com/cometbft/cometbft/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"

	ethsecp256k1 "github.com/initia-labs/initia/crypto/keys/eth/secp256k1"
)

type keyData struct {
	priv string
	pub  string
	addr string
}

var secpDataTable = []keyData{
	{
		priv: "afdfd9c3d2095ef696594f6cedcae59e72dcd697e2a7521b1578140422a4f890",
		pub:  "02585b8820efe01a0cc841fefda079dbdc6471ccf51c4f4b86c9f9dc2ee46f2944",
		addr: "06A85356DCb5b307096726FB86A78c59D38e08ee",
	},
}

func TestPubKeySecp256k1Address(t *testing.T) {
	for _, d := range secpDataTable {
		privB, _ := hex.DecodeString(d.priv)
		pubB, _ := hex.DecodeString(d.pub)
		addrBbz, _ := hex.DecodeString(d.addr)
		addrB := crypto.Address(addrBbz)

		priv := secp256k1.PrivKey{Key: privB}

		pubKey := &ethsecp256k1.PubKey{Key: priv.PubKey().Bytes()}

		addr := pubKey.Address()
		assert.Equal(t, pubKey, &ethsecp256k1.PubKey{Key: pubB}, "Expected pub keys to match")
		assert.Equal(t, addr, addrB, "Expected addresses to match")
	}
}

func TestSignAndValidateSecp256k1(t *testing.T) {
	privKey := secp256k1.GenPrivKey()
	pubKey := &ethsecp256k1.PubKey{Key: privKey.PubKey().Bytes()}

	msg := crypto.CRandBytes(1000)
	sig, err := privKey.Sign(msg)
	require.Nil(t, err)
	assert.True(t, pubKey.VerifySignature(msg, sig))

	// ----
	// Test cross packages verification
	msgHash := crypto.Sha256(msg)
	btcPrivKey, btcPubKey := btcSecp256k1.PrivKeyFromBytes(privKey.Key)
	// This fails: malformed signature: no header magic
	//   btcSig, err := secp256k1.ParseSignature(sig, secp256k1.S256())
	//   require.NoError(t, err)
	//   assert.True(t, btcSig.Verify(msgHash, btcPubKey))
	// So we do a hacky way:
	r := new(big.Int)
	s := new(big.Int)
	r.SetBytes(sig[:32])
	s.SetBytes(sig[32:])
	ok := ecdsa.Verify(btcPubKey.ToECDSA(), msgHash, r, s)
	require.True(t, ok)

	sig2, err := btcecdsa.SignCompact(btcPrivKey, msgHash, false)
	// Chop off compactSigRecoveryCode.
	sig2 = sig2[1:]
	require.NoError(t, err)
	pubKey.VerifySignature(msg, sig2)

	// ----
	// Mutate the signature, just one bit.
	sig[3] ^= byte(0x01)
	assert.False(t, pubKey.VerifySignature(msg, sig))
}

// This test is intended to justify the removal of calls to the underlying library
// in creating the privkey.
func TestSecp256k1LoadPrivkeyAndSerializeIsIdentity(t *testing.T) {
	numberOfTests := 256
	for i := 0; i < numberOfTests; i++ {
		// Seed the test case with some random bytes
		privKeyBytes := [32]byte{}
		copy(privKeyBytes[:], crypto.CRandBytes(32))

		// This function creates a private and public key in the underlying libraries format.
		// The private key is basically calling new(big.Int).SetBytes(pk), which removes leading zero bytes
		priv, _ := btcSecp256k1.PrivKeyFromBytes(privKeyBytes[:])
		// this takes the bytes returned by `(big int).Bytes()`, and if the length is less than 32 bytes,
		// pads the bytes from the left with zero bytes. Therefore these two functions composed
		// result in the identity function on privKeyBytes, hence the following equality check
		// always returning true.
		serializedBytes := priv.Serialize()
		require.Equal(t, privKeyBytes[:], serializedBytes)
	}
}

func TestPubKeyEquals(t *testing.T) {
	secp256K1PubKey := &ethsecp256k1.PubKey{Key: secp256k1.GenPrivKey().PubKey().Bytes()}

	testCases := []struct {
		msg      string
		pubKey   cryptotypes.PubKey
		other    cryptotypes.PubKey
		expectEq bool
	}{
		{
			"different bytes",
			secp256K1PubKey,
			&ethsecp256k1.PubKey{Key: secp256k1.GenPrivKey().PubKey().Bytes()},
			false,
		},
		{
			"equals",
			secp256K1PubKey,
			&ethsecp256k1.PubKey{
				Key: secp256K1PubKey.Key,
			},
			true,
		},
		{
			"different types",
			secp256K1PubKey,
			ed25519.GenPrivKey().PubKey(),
			false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			eq := tc.pubKey.Equals(tc.other)
			require.Equal(t, eq, tc.expectEq)
		})
	}
}

func TestMarshalAmino(t *testing.T) {
	aminoCdc := codec.NewLegacyAmino()
	privKey := secp256k1.GenPrivKey()
	pubKey := &ethsecp256k1.PubKey{Key: privKey.PubKey().Bytes()}

	testCases := []struct {
		desc      string
		msg       codec.AminoMarshaler
		typ       interface{}
		expBinary []byte
		expJSON   string
	}{
		{
			"secp256k1 private key",
			privKey,
			&secp256k1.PrivKey{},
			append([]byte{32}, privKey.Bytes()...), // Length-prefixed.
			"\"" + base64.StdEncoding.EncodeToString(privKey.Bytes()) + "\"",
		},
		{
			"secp256k1 public key",
			pubKey,
			&ethsecp256k1.PubKey{},
			append([]byte{33}, pubKey.Bytes()...), // Length-prefixed.
			"\"" + base64.StdEncoding.EncodeToString(pubKey.Bytes()) + "\"",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			// Do a round trip of encoding/decoding binary.
			bz, err := aminoCdc.Marshal(tc.msg)
			require.NoError(t, err)
			require.Equal(t, tc.expBinary, bz)

			err = aminoCdc.Unmarshal(bz, tc.typ)
			require.NoError(t, err)

			require.Equal(t, tc.msg, tc.typ)

			// Do a round trip of encoding/decoding JSON.
			bz, err = aminoCdc.MarshalJSON(tc.msg)
			require.NoError(t, err)
			require.Equal(t, tc.expJSON, string(bz))

			err = aminoCdc.UnmarshalJSON(bz, tc.typ)
			require.NoError(t, err)

			require.Equal(t, tc.msg, tc.typ)
		})
	}
}
