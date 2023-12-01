//go:build libsecp256k1_sdk
// +build libsecp256k1_sdk

package secp256k1

import (
	"github.com/cometbft/cometbft/crypto"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1/internal/secp256k1"
)

// VerifySignature validates the signature.
// The msg will be hashed prior to signature verification.
func (pubKey *PubKey) VerifySignature(msg []byte, sigStr []byte) bool {
	return secp256k1.VerifySignature(pubKey.Bytes(), crypto.Sha256(msg), sigStr)
}
