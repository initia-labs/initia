package secp256k1

import "golang.org/x/crypto/sha3"

func Keccak256(msg []byte) []byte {
	hasher := sha3.NewLegacyKeccak256()
	if _, err := hasher.Write(msg); err != nil {
		panic(err)
	}
	return hasher.Sum(nil)
}
