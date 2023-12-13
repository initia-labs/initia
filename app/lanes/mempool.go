package lanes

import (
	"context"
	"errors"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"
	signer_extraction "github.com/skip-mev/block-sdk/adapters/signer_extraction_adapter"
	blockbase "github.com/skip-mev/block-sdk/block/base"
)

type (
	txKey struct {
		nonce  uint64
		sender string
	}

	// Mempool defines a mempool that orders transactions based on the
	// txPriority. The mempool is a wrapper on top of the SDK's Priority Nonce mempool.
	// It include's additional helper functions that allow users to determine if a
	// transaction is already in the mempool and to compare the priority of two
	// transactions.
	Mempool[C comparable] struct {
		// index defines an index of transactions.
		index sdkmempool.Mempool

		// signerExtractor defines the signer extraction adapter that allows us to
		// extract the signer from a transaction.
		extractor signer_extraction.Adapter

		// txCache is a map of all transactions in the mempool. It is used
		// to quickly check if a transaction is already in the mempool.
		txCache map[txKey]struct{}
	}
)

// NewMempool returns a new Mempool.
func NewMempool[C comparable](txPriority blockbase.TxPriority[C], extractor signer_extraction.Adapter, maxTx int) *Mempool[C] {
	return &Mempool[C]{
		index: blockbase.NewPriorityMempool(
			blockbase.PriorityNonceMempoolConfig[C]{
				TxPriority: txPriority,
				MaxTx:      maxTx,
			},
			extractor,
		),
		extractor: extractor,
		txCache:   make(map[txKey]struct{}),
	}
}

// Priority returns the priority of the transaction.
func (cm *Mempool[C]) Priority(ctx sdk.Context, tx sdk.Tx) any {
	return 1
}

// CountTx returns the number of transactions in the mempool.
func (cm *Mempool[C]) CountTx() int {
	return cm.index.CountTx()
}

// Select returns an iterator of all transactions in the mempool. NOTE: If you
// remove a transaction from the mempool while iterating over the transactions,
// the iterator will not be aware of the removal and will continue to iterate
// over the removed transaction. Be sure to reset the iterator if you remove a transaction.
func (cm *Mempool[C]) Select(ctx context.Context, txs [][]byte) sdkmempool.Iterator {
	return cm.index.Select(ctx, txs)
}

// Compare return 0 to ignore priority check in ProcessLaneHandler.
func (cm *Mempool[C]) Compare(ctx sdk.Context, this sdk.Tx, other sdk.Tx) (int, error) {
	return 0, nil
}

// Contains returns true if the transaction is contained in the mempool.
func (cm *Mempool[C]) Contains(tx sdk.Tx) bool {
	if key, err := cm.getTxKey(tx); err != nil {
		return false
	} else {
		if _, ok := cm.txCache[key]; ok {
			return true
		} else {
			return false
		}
	}
}

// Insert inserts a transaction into the mempool.
func (cm *Mempool[C]) Insert(ctx context.Context, tx sdk.Tx) error {
	if err := cm.index.Insert(ctx, tx); err != nil {
		return fmt.Errorf("failed to insert tx into auction index: %w", err)
	}

	if key, err := cm.getTxKey(tx); err != nil {
		return err
	} else {
		cm.txCache[key] = struct{}{}
	}

	return nil
}

// Remove removes a transaction from the mempool.
func (cm *Mempool[C]) Remove(tx sdk.Tx) error {
	if err := cm.index.Remove(tx); err != nil && !errors.Is(err, sdkmempool.ErrTxNotFound) {
		return fmt.Errorf("failed to remove transaction from the mempool: %w", err)
	}

	if key, err := cm.getTxKey(tx); err != nil {
		return err
	} else {
		delete(cm.txCache, key)
	}

	return nil
}

func (cm *Mempool[C]) getTxKey(tx sdk.Tx) (txKey, error) {
	signers, err := cm.extractor.GetSigners(tx)
	if err != nil {
		return txKey{}, err
	}
	if len(signers) == 0 {
		return txKey{}, fmt.Errorf("attempted to remove a tx with no signatures")
	}
	sig := signers[0]
	sender := sig.Signer.String()
	nonce := sig.Sequence
	return txKey{nonce, sender}, nil
}
