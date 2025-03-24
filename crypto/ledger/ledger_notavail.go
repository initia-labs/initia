//go:build !cgo || !ledger
// +build !cgo !ledger

package ledger

import (
	"github.com/pkg/errors"

	"github.com/cosmos/cosmos-sdk/crypto/ledger"
)

func FindLedgerEthereumApp() (ledger.SECP256K1, error) {
	return nil, errors.New("support for ledger devices is not available in this executable")
}
