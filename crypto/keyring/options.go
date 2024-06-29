package keyring

import (
	cosmoshd "github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/types"

	"github.com/initia-labs/initia/crypto/ethsecp256k1"
	"github.com/initia-labs/initia/crypto/hd"
)

// AppName defines the Ledger app used for signing. Evmos uses the Ethereum app
const AppName = "Ethereum"

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

// EthSecp256k1Option defines a function keys options for the ethereum Secp256k1 curve.
// It supports eth_secp256k1 keys for accounts.
func Option() keyring.Option {
	return func(options *keyring.Options) {
		options.SupportedAlgos = SupportedAlgorithms
		options.SupportedAlgosLedger = SupportedAlgorithmsLedger
	}
}

// EthSecp256k1Option defines a function keys options for the ethereum Secp256k1 curve.
// It supports eth_secp256k1 keys for accounts.
func EthSecp256k1Option() keyring.Option {
	return func(options *keyring.Options) {
		options.SupportedAlgos = SupportedAlgorithms
		options.SupportedAlgosLedger = SupportedAlgorithmsLedger
		options.LedgerCreateKey = func(key []byte) types.PubKey { return &ethsecp256k1.PubKey{Key: key} }
		options.LedgerAppName = "Ethereum"
		options.LedgerSigSkipDERConv = true
	}
}
