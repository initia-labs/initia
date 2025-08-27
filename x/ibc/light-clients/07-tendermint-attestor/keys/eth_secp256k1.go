package keys

import (
	"errors"

	secp256k1 "github.com/decred/dcrd/dcrec/secp256k1/v4"
	"golang.org/x/crypto/sha3"
)

const (
	EthSecp256K1SignatureSize = 65
)

var _ PubKey = (*EthSecp256K1PubKey)(nil)

type EthSecp256K1PubKey struct {
	Key []byte
}

func NewEthSecp256K1PubKey(key []byte) *EthSecp256K1PubKey {
	return &EthSecp256K1PubKey{
		Key: key,
	}
}

func (sp *EthSecp256K1PubKey) Verify(message []byte, sigBytes []byte) error {
	if len(sigBytes) == EthSecp256K1SignatureSize {
		// remove recovery ID (V) if contained in the signature
		sigBytes = sigBytes[:len(sigBytes)-1]
	}

	if len(sigBytes) != 64 {
		return errors.New("invalid signature size")
	}

	pub, err := secp256k1.ParsePubKey(sp.Key)
	if err != nil {
		return err
	}
	signature, err := signatureFromBytes(sigBytes)
	if err != nil {
		return err
	}

	if !signature.Verify(keccak256(message), pub) {
		return errors.New("invalid signature")
	}
	return nil
}

func keccak256(bytes []byte) []byte {
	hasher := sha3.NewLegacyKeccak256()
	hasher.Write(bytes)
	return hasher.Sum(nil)
}
