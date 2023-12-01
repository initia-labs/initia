package codec

import (
	"github.com/cosmos/cosmos-sdk/codec"

	ethsecp256k1 "github.com/initia-labs/initia/crypto/keys/eth/secp256k1"
)

// RegisterCrypto registers all crypto dependency types with the provided Amino
// codec.
func RegisterCrypto(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&ethsecp256k1.PubKey{}, ethsecp256k1.PubKeyName, nil)
}
