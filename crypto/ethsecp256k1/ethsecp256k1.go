package ethsecp256k1

import (
	"bytes"
	"crypto/subtle"
	"errors"
	"fmt"
	io "io"
	"math/big"

	errorsmod "cosmossdk.io/errors"
	"github.com/cometbft/cometbft/crypto"
	"golang.org/x/crypto/sha3"

	"github.com/cosmos/cosmos-sdk/codec"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"

	secp256k1 "github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
)

const (
	// PrivKeySize defines the size of the PrivKey bytes
	PrivKeySize = 32
	// PubKeySize defines the size of the PubKey bytes
	PubKeySize = 33
	// UncompressedPubKeySize defines the size of the uncompressed PubKey bytes
	UncompressedPubKeySize = 65
	// SignatureSize defines the size of the ECDSA signature with the recovery ID
	SignatureSize = 65
	// KeyType is the string constant for the Secp256k1 algorithm
	KeyType = "eth_secp256k1"
)

// Amino encoding names
const (
	// PrivKeyName defines the amino encoding name for the EthSecp256k1 private key
	PrivKeyName = "initia/PrivKeyEthSecp256k1"
	// PubKeyName defines the amino encoding name for the EthSecp256k1 public key
	PubKeyName = "initia/PubKeyEthSecp256k1"
)

// ----------------------------------------------------------------------------
// secp256k1 Private Key

var (
	_ cryptotypes.PrivKey  = &PrivKey{}
	_ codec.AminoMarshaler = &PrivKey{}
)

// GenerateKey generates a new random private key. It returns an error upon
// failure.
func GenerateKey() *PrivKey {
	return &PrivKey{Key: genPrivKey(crypto.CReader())}
}

// genPrivKey generates a new secp256k1 private key using the provided reader.
func genPrivKey(rand io.Reader) []byte {
	var privKeyBytes [PrivKeySize]byte
	d := new(big.Int)
	for {
		privKeyBytes = [PrivKeySize]byte{}
		_, err := io.ReadFull(rand, privKeyBytes[:])
		if err != nil {
			panic(err)
		}

		d.SetBytes(privKeyBytes[:])
		// break if we found a valid point (i.e. > 0 and < N == curverOrder)
		isValidFieldElement := 0 < d.Sign() && d.Cmp(secp256k1.S256().N) < 0
		if isValidFieldElement {
			break
		}
	}

	return privKeyBytes[:]
}

// Bytes returns the byte representation of the ECDSA Private Key.
func (privKey PrivKey) Bytes() []byte {
	bz := make([]byte, len(privKey.Key))
	copy(bz, privKey.Key)

	return bz
}

// PubKey returns the ECDSA private key's public key. If the privkey is not valid
// it returns a nil value.
func (privKey PrivKey) PubKey() cryptotypes.PubKey {
	pubkeyObject := secp256k1.PrivKeyFromBytes(privKey.Key).PubKey()
	pk := pubkeyObject.SerializeCompressed()

	return &PubKey{Key: pk}
}

// Equals returns true if two ECDSA private keys are equal and false otherwise.
func (privKey PrivKey) Equals(other cryptotypes.LedgerPrivKey) bool {
	return privKey.Type() == other.Type() && subtle.ConstantTimeCompare(privKey.Bytes(), other.Bytes()) == 1
}

// Type returns eth_secp256k1
func (privKey PrivKey) Type() string {
	return KeyType
}

// MarshalAmino overrides Amino binary marshaling.
func (privKey PrivKey) MarshalAmino() ([]byte, error) {
	return privKey.Key, nil
}

// UnmarshalAmino overrides Amino binary marshaling.
func (privKey *PrivKey) UnmarshalAmino(bz []byte) error {
	if len(bz) != PrivKeySize {
		return fmt.Errorf("invalid privkey size, expected %d got %d", PrivKeySize, len(bz))
	}
	privKey.Key = bz

	return nil
}

// MarshalAminoJSON overrides Amino JSON marshaling.
func (privKey PrivKey) MarshalAminoJSON() ([]byte, error) {
	// When we marshal to Amino JSON, we don't marshal the "key" field itself,
	// just its contents (i.e. the key bytes).
	return privKey.MarshalAmino()
}

// UnmarshalAminoJSON overrides Amino JSON marshaling.
func (privKey *PrivKey) UnmarshalAminoJSON(bz []byte) error {
	return privKey.UnmarshalAmino(bz)
}

// Sign creates a recoverable ECDSA signature on the secp256k1 curve over the
// provided hash of the message. The produced signature is 65 bytes
// where the last byte contains the recovery ID.
func (privKey PrivKey) Sign(msg []byte) ([]byte, error) {
	priv := secp256k1.PrivKeyFromBytes(privKey.Key)

	sig := ecdsa.SignCompact(priv, keccak256(msg), false)
	recid := sig[0]

	// remove the first byte which is compactSigRecoveryCode
	return append(sig[1:], recid), nil
}

// ----------------------------------------------------------------------------
// secp256k1 Public Key

var (
	_ cryptotypes.PubKey   = &PubKey{}
	_ codec.AminoMarshaler = &PubKey{}
)

// Create a new PubKey object from a compressed or uncompressed byte slice.
func NewPubKeyFromBytes(key []byte) (*PubKey, error) {
	if len(key) == UncompressedPubKeySize {
		// compress the key
		pub, err := secp256k1.ParsePubKey(key)
		if err != nil {
			return nil, err
		}
		key = pub.SerializeCompressed()
	} else if len(key) != PubKeySize {
		return nil, errorsmod.Wrapf(errortypes.ErrInvalidPubKey, "invalid pubkey size, expected %d, got %d", PubKeySize, len(key))
	}

	return &PubKey{Key: key}, nil
}

// Address returns the address of the ECDSA public key.
// The function will return an empty address if the public key is invalid.
func (pubKey PubKey) Address() crypto.Address {
	if len(pubKey.Key) != PubKeySize {
		panic("length of pubKey is incorrect")
	}

	pub, err := secp256k1.ParsePubKey(pubKey.Key)
	if err != nil {
		panic(err)
	}

	pubBytes := pub.SerializeUncompressed()
	return crypto.Address(keccak256(pubBytes[1:])[12:])
}

// Bytes returns the raw bytes of the ECDSA public key.
func (pubKey PubKey) Bytes() []byte {
	bz := make([]byte, len(pubKey.Key))
	copy(bz, pubKey.Key)

	return bz
}

// String implements the fmt.Stringer interface.
func (pubKey PubKey) String() string {
	return fmt.Sprintf("EthPubKeySecp256k1{%X}", pubKey.Key)
}

// Type returns eth_secp256k1
func (pubKey PubKey) Type() string {
	return KeyType
}

// Equals returns true if the pubkey type is the same and their bytes are deeply equal.
func (pubKey PubKey) Equals(other cryptotypes.PubKey) bool {
	return pubKey.Type() == other.Type() && bytes.Equal(pubKey.Bytes(), other.Bytes())
}

// MarshalAmino overrides Amino binary marshaling.
func (pubKey PubKey) MarshalAmino() ([]byte, error) {
	return pubKey.Key, nil
}

// UnmarshalAmino overrides Amino binary marshaling.
func (pubKey *PubKey) UnmarshalAmino(bz []byte) error {
	if len(bz) != PubKeySize {
		return errorsmod.Wrapf(errortypes.ErrInvalidPubKey, "invalid pubkey size, expected %d, got %d", PubKeySize, len(bz))
	}
	pubKey.Key = bz

	return nil
}

// MarshalAminoJSON overrides Amino JSON marshaling.
func (pubKey PubKey) MarshalAminoJSON() ([]byte, error) {
	// When we marshal to Amino JSON, we don't marshal the "key" field itself,
	// just its contents (i.e. the key bytes).
	return pubKey.MarshalAmino()
}

// UnmarshalAminoJSON overrides Amino JSON marshaling.
func (pubKey *PubKey) UnmarshalAminoJSON(bz []byte) error {
	return pubKey.UnmarshalAmino(bz)
}

// VerifySignature verifies that the ECDSA public key created a given signature over
// the provided message.
//
// CONTRACT: The signature should be in [R || S] format.
func (pubKey PubKey) VerifySignature(msg, sig []byte) bool {
	return pubKey.verifySignatureECDSA(msg, sig)
}

// Perform standard ECDSA signature verification for the given raw bytes and signature.
func (pubKey PubKey) verifySignatureECDSA(msg, sigStr []byte) bool {
	if len(sigStr) == SignatureSize {
		// remove recovery ID (V) if contained in the signature
		sigStr = sigStr[:len(sigStr)-1]
	}

	if len(sigStr) != 64 {
		return false
	}

	pub, err := secp256k1.ParsePubKey(pubKey.Key)
	if err != nil {
		return false
	}
	// parse the signature, will return error if it is not in lower-S form
	signature, err := signatureFromBytes(sigStr)
	if err != nil {
		return false
	}

	return signature.Verify(keccak256(msg), pub)
}

// Read Signature struct from R || S. Caller needs to ensure
// that len(sigStr) == 64.
// Rejects malleable signatures (if S value if it is over half order).
func signatureFromBytes(sigStr []byte) (*ecdsa.Signature, error) {
	var r secp256k1.ModNScalar
	r.SetByteSlice(sigStr[:32])
	var s secp256k1.ModNScalar
	s.SetByteSlice(sigStr[32:64])
	if s.IsOverHalfOrder() {
		return nil, errors.New("signature is not in lower-S form")
	}

	return ecdsa.NewSignature(&r, &s), nil
}

func keccak256(bytes []byte) []byte {
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(bytes)
	return hasher.Sum(nil)
}
