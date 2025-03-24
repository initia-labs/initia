package keyring

import (
	"fmt"

	cosmosledger "github.com/cosmos/ledger-cosmos-go"

	"github.com/cosmos/cosmos-sdk/client/flags"
	cosmoshd "github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/crypto/ledger"
	"github.com/cosmos/cosmos-sdk/crypto/types"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/initia-labs/initia/crypto/ethsecp256k1"
	"github.com/initia-labs/initia/crypto/hd"
	ethhd "github.com/initia-labs/initia/crypto/hd"
	ethledger "github.com/initia-labs/initia/crypto/ledger"
)

var (
	// SupportedAlgorithms defines the list of signing algorithms used on Injective:
	//  - eth_secp256k1 (Ethereum)
	//  - secp256k1 (Cosmos SDK)
	SupportedAlgorithms = keyring.SigningAlgoList{ethhd.EthSecp256k1, cosmoshd.Secp256k1}
	// SupportedAlgorithmsLedger defines the list of signing algorithms used for the Ledger device:
	//  - secp256k1 (in order to comply with Cosmos SDK)
	// The Ledger derivation function is responsible for all signing and address generation.
	SupportedAlgorithmsLedger = keyring.SigningAlgoList{ethhd.EthSecp256k1, cosmoshd.Secp256k1}
)

// EthSecp256k1Option defines a function keys options for the ethereum Secp256k1 curve.
// It supports eth_secp256k1 keys for accounts.
func EthSecp256k1Option() keyring.Option {
	return func(options *keyring.Options) {
		isCosmosLedger := false

		options.SupportedAlgos = SupportedAlgorithms
		options.SupportedAlgosLedger = SupportedAlgorithmsLedger
		options.LedgerDerivation = func() (ledger.SECP256K1, error) {
			ledger.SetAppName("Ethereum")
			if !isCosmosLedger {
				if device, err := ethledger.FindLedgerEthereumApp(); err == nil {
					fmt.Println("Ethereum ledger found")
					ledger.SetSkipDERConversion()

					// ethereum ledger should have coin type 60 and key type eth_secp256k1
					if err := validateFlags(60, string(ethhd.EthSecp256k1Type)); err != nil {
						fmt.Printf(`
Failed to validate flags for Ethereum ledger:
%s

Please make sure you have the correct coin type and key type set for the Ethereum ledger.

You can use the following command to set the correct flags:
--coin-type 60 --key-type %s

`, err.Error(), ethhd.EthSecp256k1Type)
						return nil, err
					}

					return device, nil
				}

				fmt.Println("Ethereum ledger is offline")
			}

			isCosmosLedger = true
			ledger.SetAppName("Cosmos")
			if device, err := cosmosledger.FindLedgerCosmosUserApp(); err == nil {
				fmt.Println("Cosmos ledger found")

				// cosmos ledger should have coin type 118 and key type secp256k1
				if err := validateFlags(118, string(cosmoshd.Secp256k1Type)); err != nil {
					fmt.Printf(`
Failed to validate flags for Cosmos ledger:
%s

Please make sure you have the correct coin type and key type set for the Cosmos ledger.

You can use the following command to set the correct flags:
--coin-type 118 --key-type %s

`, err.Error(), cosmoshd.Secp256k1Type)
					return nil, err
				}

				return device, nil
			}

			fmt.Println("Cosmos ledger is offline")

			return nil, fmt.Errorf("Failed to load ledger device Ethereum or Cosmos")
		}
		options.LedgerCreateKey = func(key []byte) types.PubKey {
			if !isCosmosLedger {
				fmt.Println("Using Ethereum Pubkey")
				return &ethsecp256k1.PubKey{Key: key}
			}

			fmt.Println("Using Cosmos Pubkey")
			return &secp256k1.PubKey{Key: key}
		}
		options.LedgerAppName = "Cosmos"
		options.LedgerSigSkipDERConv = false
	}
}

const flagCoinType = "coin-type"

func validateFlags(eCoinType uint32, eKeyType string) error {
	keyType := viper.GetString(flags.FlagKeyType)
	if keyType != "" && keyType != eKeyType {
		return fmt.Errorf("expected key type %s, got %s", eKeyType, keyType)
	}

	coinType := viper.GetUint32(flagCoinType)
	if coinType != 0 && coinType != eCoinType {
		return fmt.Errorf("expected coin type %d, got %d", eCoinType, coinType)
	}

	return nil
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
