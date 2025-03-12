//go:build !cgo || !ledger
// +build !cgo !ledger

// test_ledger_mock

package keyring

import (
	"github.com/cosmos/cosmos-sdk/crypto/ledger"
	"github.com/pkg/errors"
)

func LedgerDerivationFn() func() (ledger.SECP256K1, error) {
	return func() (ledger.SECP256K1, error) {
		return nil, errors.New("support for ledger devices is not available in this executable")
	}
}
