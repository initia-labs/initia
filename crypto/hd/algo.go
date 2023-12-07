package hd

import (
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/types"

	"github.com/initia-labs/initia/crypto/keys/eth/secp256k1"
)

var (
	// EthSecp256k1Type uses the ethereum secp256k1 ECDSA parameters.
	EthSecp256k1Type = hd.PubKeyType("eth_secp256k1")
)

// EthSecp256k1 uses the Bitcoin secp256k1 ECDSA parameters.
var EthSecp256k1 = ethSecp256k1Algo{}

type ethSecp256k1Algo struct{}

func (s ethSecp256k1Algo) Name() hd.PubKeyType {
	return EthSecp256k1Type
}

// Derive derives and returns the secp256k1 private key for the given seed and HD path.
func (s ethSecp256k1Algo) Derive() hd.DeriveFn {
	return hd.Secp256k1.Derive()
}

// Generate generates a eth_secp256k1 private key from the given bytes.
func (s ethSecp256k1Algo) Generate() hd.GenerateFn {
	return func(bz []byte) types.PrivKey {
		bzArr := make([]byte, secp256k1.PrivKeySize)
		copy(bzArr, bz)

		return &secp256k1.PrivKey{Key: bzArr}
	}
}
