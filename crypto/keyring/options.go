package keyring

import (
	cosmoshd "github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/initia-labs/initia/crypto/hd"
)

var (
	// SupportedAlgorithms defines the list of signing algorithms used on Injective:
	//  - eth_secp256k1 (Ethereum)
	//  - secp256k1 (Cosmos SDK)
	SupportedAlgorithms = keyring.SigningAlgoList{hd.EthSecp256k1, cosmoshd.Secp256k1}
	// SupportedAlgorithmsLedger defines the list of signing algorithms used for the Ledger device:
	//  - secp256k1 (in order to comply with Cosmos SDK)
	// The Ledger derivation function is responsible for all signing and address generation.
	SupportedAlgorithmsLedger = keyring.SigningAlgoList{hd.EthSecp256k1, cosmoshd.Secp256k1}
)

// Option EthSecp256k1Option defines a function keys options for the ethereum Secp256k1 curve.
// It supports eth_secp256k1 keys for accounts.
func Option() keyring.Option {
	return func(options *keyring.Options) {
		options.SupportedAlgos = SupportedAlgorithms
		options.SupportedAlgosLedger = SupportedAlgorithmsLedger
	}
}
