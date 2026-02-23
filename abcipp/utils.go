package abcipp

import (
	"encoding/hex"
	"fmt"
	"strings"

	comettypes "github.com/cometbft/cometbft/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/signing"
)

// GetDecodedTxs returns the decoded transactions from the given bytes.
func GetDecodedTxs(txDecoder sdk.TxDecoder, txs [][]byte) ([]sdk.Tx, error) {
	var decodedTxs []sdk.Tx
	for _, txBz := range txs {
		tx, err := txDecoder(txBz)
		if err != nil {
			return nil, fmt.Errorf("failed to decode transaction: %w", err)
		}

		decodedTxs = append(decodedTxs, tx)
	}

	return decodedTxs, nil
}

// TxHash returns the string hash representation of the given transactions.
func TxHash(txBytes []byte) string {
	return strings.ToUpper(hex.EncodeToString(comettypes.Tx(txBytes).Hash()))
}

// DecodeAddress decodes a string address which can be either hex or bech32 encoded.
func DecodeAddress(sender string) (sdk.AccAddress, error) {
	if strings.HasPrefix(sender, "0x") {
		raw, err := hex.DecodeString(strings.Replace(sender, "0x", "", 1))
		if err != nil {
			return nil, fmt.Errorf("invalid hex address: %w", err)
		}
		return sdk.AccAddress(raw), nil
	}
	return sdk.AccAddressFromBech32(sender)
}

// FirstSignature extracts the first signature's address and sequence from the given transaction.
func FirstSignature(tx sdk.Tx) (sdk.AccAddress, uint64, error) {
	sigTx, ok := tx.(signing.SigVerifiableTx)
	if !ok {
		return nil, 0, fmt.Errorf("transaction must implement SigVerifiableTx")
	}

	sigs, err := sigTx.GetSignaturesV2()
	if err != nil {
		return nil, 0, err
	}
	if len(sigs) == 0 {
		return nil, 0, fmt.Errorf("transaction must have at least one signer")
	}
	if sigs[0].PubKey == nil {
		return nil, 0, fmt.Errorf("first signature pubkey is nil")
	}

	addr := sdk.AccAddress(sigs[0].PubKey.Address())
	return addr, sigs[0].Sequence, nil
}

// fetchSequence queries the on-chain sequence for a sender.
func fetchSequence(ctx sdk.Context, ak AccountKeeper, sender string) (uint64, bool) {
	addr, err := sdk.AccAddressFromBech32(sender)
	if err != nil {
		return 0, false
	}

	seq, err := ak.GetSequence(ctx, addr)
	if err != nil {
		// AccountKeeper.GetSequence returns an error only when the account does not
		// exist yet. Treat that as sequence 0 and mark the lookup as usable.
		return 0, true
	}

	return seq, true
}
