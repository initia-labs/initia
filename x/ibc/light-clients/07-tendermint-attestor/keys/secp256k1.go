package keys

import (
	"errors"

	"crypto/sha256"

	secp256k1 "github.com/decred/dcrd/dcrec/secp256k1/v4"
	"github.com/decred/dcrd/dcrec/secp256k1/v4/ecdsa"
)

var _ PubKey = (*Secp256K1PubKey)(nil)

type Secp256K1PubKey struct {
	Key []byte
}

func NewSecp256K1PubKey(key []byte) *Secp256K1PubKey {
	return &Secp256K1PubKey{
		Key: key,
	}
}

func (sp *Secp256K1PubKey) Verify(message []byte, sigBytes []byte) error {
	pubKey, err := secp256k1.ParsePubKey(sp.Key)
	if err != nil {
		return err
	}

	signature, err := signatureFromBytes(sigBytes)
	if err != nil {
		return err
	}

	if !signature.Verify(sha256bz(message), pubKey) {
		return errors.New("invalid signature")
	}
	return nil
}

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

func sha256bz(bytes []byte) []byte {
	hasher := sha256.New()
	hasher.Write(bytes)
	return hasher.Sum(nil)
}
