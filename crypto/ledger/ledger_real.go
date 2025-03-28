//go:build cgo && ledger && !test_ledger_mock
// +build cgo,ledger,!test_ledger_mock

package ledger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"

	"github.com/cosmos/cosmos-sdk/crypto/ledger"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/initia-labs/initia/crypto/ledger/accounts"
	"github.com/initia-labs/initia/crypto/ledger/usbwallet"
)

var _ ledger.SECP256K1 = &InitiaLedger{}

var initiaLedger *InitiaLedger

func FindLedgerEthereumApp() (ledger.SECP256K1, error) {
	if initiaLedger == nil {
		initiaLedger = new(InitiaLedger)
	}

	return initiaLedger.connect()
}

type InitiaLedger struct {
	*usbwallet.Hub
	wallet accounts.Wallet
}

func (i *InitiaLedger) connect() (*InitiaLedger, error) {
	if i.Hub != nil {
		return i, nil
	}

	hub, err := usbwallet.NewLedgerHub()
	if err != nil {
		return nil, err
	}

	wallets := hub.Wallets()
	if len(wallets) == 0 {
		return nil, errors.New("no wallets found")
	}

	wallet := wallets[0]
	err = wallet.Open("")
	if err != nil && !strings.Contains(err.Error(), "already open") {
		return nil, errors.Wrap(err, "failed to open wallet")
	}

	// check if ethereum app is offline or not
	status, err := wallet.Status()
	if err != nil {
		wallet.Close()
		return nil, errors.Wrap(err, "failed to get wallet status")
	} else if status == "Ethereum app offline" {
		wallet.Close()
		return nil, errors.New(status)
	}

	return &InitiaLedger{
		Hub:    hub,
		wallet: wallet,
	}, nil
}

// Close implements ledger.SECP256K1.
func (i *InitiaLedger) Close() error {
	return i.wallet.Close()
}

// GetAddressPubKeySECP256K1 implements ledger.SECP256K1.
func (i *InitiaLedger) GetAddressPubKeySECP256K1(hdPath []uint32, hrp string) ([]byte, string, error) {
	formattedHDPath := formatHDPathToEthereumCompatible(hdPath)

	err := i.wallet.Open("")
	if err != nil && !strings.Contains(err.Error(), "already open") {
		return nil, "", errors.Wrap(err, "failed to open wallet")
	}

	account, err := i.wallet.Derive(formattedHDPath, true)
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to derive account")
	}

	address, err := sdk.Bech32ifyAddressBytes(hrp, account.Address.Bytes())
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to bech32ify address")
	}

	if account.Pubkey == nil {
		return nil, "", errors.New("pubkey is nil")
	}

	pubkeyBz := crypto.FromECDSAPub(account.Pubkey)
	return pubkeyBz, address, nil
}

// GetPublicKeySECP256K1 implements ledger.SECP256K1.
func (i *InitiaLedger) GetPublicKeySECP256K1(hdPath []uint32) ([]byte, error) {
	formattedHDPath := formatHDPathToEthereumCompatible(hdPath)

	err := i.wallet.Open("")
	if err != nil && !strings.Contains(err.Error(), "already open") {
		return nil, errors.Wrap(err, "failed to open wallet")
	}

	account, err := i.wallet.Derive(formattedHDPath, true)
	if err != nil {
		return nil, errors.Wrap(err, "failed to derive account")
	}

	if account.Pubkey == nil {
		return nil, errors.New("pubkey is nil")
	}

	pubkeyBz := crypto.FromECDSAPub(account.Pubkey)
	return pubkeyBz, nil
}

// SignSECP256K1 implements ledger.SECP256K1.
func (i *InitiaLedger) SignSECP256K1(hdPath []uint32, signBytes []byte, _ byte) ([]byte, error) {
	formattedHDPath := formatHDPathToEthereumCompatible(hdPath)

	err := i.wallet.Open("")
	if err != nil && !strings.Contains(err.Error(), "already open") {
		return nil, errors.Wrap(err, "failed to open wallet")
	}

	account, err := i.wallet.Derive(formattedHDPath, true)
	if err != nil {
		return nil, errors.Wrap(err, "failed to derive account")
	}

	// pretty print the sign bytes

	var prettySignBytes bytes.Buffer
	err = json.Indent(&prettySignBytes, signBytes, "", "  ")
	if err != nil {
		return nil, errors.Wrap(err, "failed to indent sign bytes")
	}

	fmt.Printf(`
################################################
Please check your Ledger device for confirmation

Signing message:
%s
################################################

`, prettySignBytes.String())

	sig, err := i.wallet.SignText(account, signBytes)
	if err != nil {
		return nil, errors.Wrap(err, "failed to sign message")
	}

	return sig, nil
}

// formatHDPathToEthereumCompatible formats the HD path to be compatible with the Ethereum ledger app.
func formatHDPathToEthereumCompatible(hdPath []uint32) []uint32 {
	formattedHDPath := make([]uint32, len(hdPath))
	copy(formattedHDPath, hdPath)
	for i := range 3 {
		formattedHDPath[i] += 0x80000000
	}

	return formattedHDPath
}
