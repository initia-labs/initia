package keyring

import (
	"github.com/cosmos/cosmos-sdk/client/flags"
	cosmoshd "github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/spf13/cobra"

	"github.com/initia-labs/initia/crypto/ethsecp256k1"
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

// EthSecp256k1Option defines a function keys options for the ethereum Secp256k1 curve.
// It supports eth_secp256k1 keys for accounts.
func EthSecp256k1Option() keyring.Option {
	return func(options *keyring.Options) {
		options.SupportedAlgos = SupportedAlgorithms
		options.SupportedAlgosLedger = SupportedAlgorithmsLedger
		options.LedgerDerivation = LedgerDerivationFn()
		options.LedgerCreateKey = func(key []byte) types.PubKey { return &ethsecp256k1.PubKey{Key: key} }
		options.LedgerAppName = "Ethereum"
		options.LedgerSigSkipDERConv = true
	}
}

// OverrideDefaultKeyType overrides the default key type for the given command.
// It is used to set the default key type to eth_secp256k1 for the given command.
func OverrideDefaultKeyType(cmd *cobra.Command) *cobra.Command {
	for _, cmd := range cmd.Commands() {
		f := cmd.Flag(flags.FlagKeyType)
		if f == nil {
			continue
		}

		f.DefValue = string(hd.EthSecp256k1Type)
		err := f.Value.Set(string(hd.EthSecp256k1Type))
		if err != nil {
			panic(err)
		}
	}

	return cmd
}
