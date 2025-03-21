//go:build ledger && test_ledger_mock
// +build ledger,test_ledger_mock

package ledger

import (
	"github.com/cosmos/cosmos-sdk/crypto/ledger"
	"github.com/pkg/errors"
)

func FindLedgerEthereumApp() (ledger.SECP256K1, error) {
	return nil, errors.New("support for ledger devices is not available in this executable")
}
